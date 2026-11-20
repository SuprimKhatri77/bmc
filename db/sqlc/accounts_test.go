package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/suprimkhatri77/bmc/util"
)

// helper (not a test: go test only runs functions starting with Test) that creates a random account,
// checks it was saved right and returns it, so other tests can use it instead of repeating the create
// it takes t so the require checks in here fail whichever test called it
func createRandomAccount(t *testing.T) Account {
	arg := CreateAccountParams{
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
	account, err := testQueries.CreateAccount(context.Background(), arg)
	// require (from stretchr/testify) stops the test right there if a check fails
	// NoError: the query didn't return an error
	require.NoError(t, err)
	// NotEmpty: the returned account isn't an empty struct
	require.NotEmpty(t, account)

	// Equal: the saved values match what we sent
	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	// NotZero: the value isn't its type's zero value (0, "", zero time), negative numbers would still pass
	// the db fills these in (bigserial id, default now()), so they should be set
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

// creates an account, fetches it back by id and checks it's the same account
func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	// WithinDuration: the two times are at most 1 second apart (the delta)
	// safer than Equal, which can fail on the same moment stored with different timezone details
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// creates an account, updates its balance, and checks only the balance changed
func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	arg := UpdateAccountParams{
		ID:      account1.ID,
		Balance: util.RandomMoney(),
	}

	account2, err := testQueries.UpdateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	// owner, currency and created_at stay the same
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Currency, account2.Currency)
	// balance is the new one we sent
	require.Equal(t, arg.Balance, account2.Balance)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// creates an account, deletes it, then checks that fetching it again finds nothing
func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	// this time we want an error
	require.Error(t, err)
	// sql.ErrNoRows is what a :one query returns when no row matched, i.e. the account is gone
	require.EqualError(t, err, sql.ErrNoRows.Error())
	// nothing was found, so account2 is an empty struct
	require.Empty(t, account2)

}

// creates 10 accounts so the table has at least 10, then lists one page of 5
func TestListAccount(t *testing.T) {
	for range 10 {
		createRandomAccount(t)
	}

	// Offset skips the first 5 rows of the whole table (ordered by id, so old accounts too),
	// Limit returns the next 5
	arg := ListAccountsParams{
		Limit:  5,
		Offset: 5,
	}

	accounts, err := testQueries.ListAccounts(context.Background(), arg)
	require.NoError(t, err)
	// Len: exactly 5 came back, because of the limit
	require.Len(t, accounts, 5)

	for _, account := range accounts {
		require.NotEmpty(t, account)
	}
}
