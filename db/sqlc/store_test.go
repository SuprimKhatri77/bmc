package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// runs several transfers between the same two accounts at the same time, then checks every
// result and that its rows were really saved; running them concurrently is how real traffic
// looks, and it's what will expose locking problems once the balance update is added
func TestTransferTx(t *testing.T) {
	// a Store on the same pool as testQueries (testDB from main_test.go), so we get execTx and TransferTx
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	// n transfers of the same fixed amount, all from account1 to account2
	// a fixed amount (not random) makes the totals easy to check later: account1 should lose n*amount
	n := 5
	amount := int64(10)

	// require can't be used inside a goroutine: when a check fails it stops the test with t.FailNow,
	// which only works from the test's own goroutine. so each goroutine sends its err and result back
	// through these channels and the checks run below, in the test itself
	// unbuffered: a send waits until the loop below receives it
	errs := make(chan error)
	results := make(chan TransferTxResult)

	// start n goroutines, each doing one transfer, so the transactions overlap in the db
	for range n {
		go func() {
			result, err := store.TransactionTx(context.Background(), TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result
		}()
	}

	// receive one err and one result per goroutine and check them
	// existed records which k (1..n, see the balance check below) we've already seen
	existed := make(map[int]bool)
	for range n {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// the transfer row: right accounts and amount, and the db filled in id and created_at
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		// fetch it again outside the transaction: proves it was committed, not just returned
		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// sender's entry: money going out, so -amount
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		// receiver's entry: money coming in, so +amount (the + does nothing, it's just there to read clearly)
		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, +amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// the accounts returned by the transaction (balances right after this transfer)
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		// check the balances: account1/account2 still hold the starting balances (from before the loop)
		fmt.Println(">> tx:", fromAccount.Balance, toAccount.Balance)
		// diff1: how much account1 has lost so far, diff2: how much account2 has gained so far
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance

		// money only moves, so what one lost the other gained
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		// transfers finish one after another, so after the k-th one the total moved is k * amount
		// (1*amount, 2*amount, ..., n*amount), the goroutines just finish in any order
		require.True(t, diff1%amount == 0)

		// k = how many transfers had finished when this one did; it must be between 1 and n,
		// and no two transfers can share a k: a repeat means two read the same balance (a lost update)
		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// after all n transfers, read both accounts fresh from the db:
	// account1 lost exactly n*amount and account2 gained exactly n*amount
	// int64(n) because n is an int and amount is an int64, and go won't multiply mixed types
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)

}

func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	n := 10
	amount := int64(10)

	errs := make(chan error)

	for i := range n {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		go func() {
			_, err := store.TransactionTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err
		}()
	}

	for range n {
		err := <-errs
		require.NoError(t, err)
	}

	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)

}
