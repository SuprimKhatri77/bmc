package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/suprimkhatri77/bmc/util"
)

// helper that creates a random transfer from one account to another, checks it was saved right and returns it
// it takes both accounts so the caller decides who sends and who receives (same idea as createRandomEntry)
// this only adds a row to transfers; no money actually moves (no entries, balances unchanged)
func createRandomTransfer(t *testing.T, fromAccount, toAccount Account) Transfer {
	// both ids reference accounts.id (foreign keys), so they have to be real accounts
	arg := CreateTransferParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	// the saved values match what we sent
	require.Equal(t, arg.FromAccountID, transfer.FromAccountID)
	require.Equal(t, arg.ToAccountID, transfer.ToAccountID)
	require.Equal(t, arg.Amount, transfer.Amount)

	// the db fills these in (bigserial id, default now()), so they shouldn't be their zero value
	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t, createRandomAccount(t), createRandomAccount(t))
}

// creates a transfer between two new accounts, fetches it back by id and checks every field matches
func TestGetTransfer(t *testing.T) {
	transfer1 := createRandomTransfer(t, createRandomAccount(t), createRandomAccount(t))

	transfer2, err := testQueries.GetTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transfer2)

	require.Equal(t, transfer1.ID, transfer2.ID)
	require.Equal(t, transfer1.FromAccountID, transfer2.FromAccountID)
	require.Equal(t, transfer1.ToAccountID, transfer2.ToAccountID)
	require.Equal(t, transfer1.Amount, transfer2.Amount)
	// same moment within 1 second (safer than Equal for times)
	require.WithinDuration(t, transfer1.CreatedAt, transfer2.CreatedAt, time.Second)
}

// creates 10 transfers between two accounts, then lists the second page of 5 twice:
// once for the sender and once for the receiver, so both halves of the OR in ListTransfers get tested
func TestListTransfer(t *testing.T) {
	// all 10 transfers go fromAccount -> toAccount
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)

	for range 10 {

		createRandomTransfer(t, fromAccount, toAccount)
	}

	// ListTransfers returns transfers where AccountID is the sender OR the receiver
	// sender side: fromAccount sent all 10, so they match through from_account_id
	// Offset skips the first 5 (ordered by id), Limit returns the next 5
	arg := ListTransfersParams{
		Limit:     5,
		Offset:    5,
		AccountID: fromAccount.ID,
	}

	transfers, err := testQueries.ListTransfers(context.Background(), arg)
	require.NoError(t, err)
	// exactly 5 came back, because of the limit
	require.Len(t, transfers, 5)

	// every listed transfer was sent by fromAccount
	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, fromAccount.ID, transfer.FromAccountID)
	}

	// receiver side: same query, now with toAccount, which received all 10,
	// so they match through to_account_id (the other half of the OR)
	// = not := because arg, transfers and err already exist from the sender side
	arg = ListTransfersParams{Limit: 5, Offset: 5, AccountID: toAccount.ID}
	transfers, err = testQueries.ListTransfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)
	// every listed transfer was received by toAccount
	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, toAccount.ID, transfer.ToAccountID)
	}
}
