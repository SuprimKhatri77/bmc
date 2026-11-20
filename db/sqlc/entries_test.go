package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/suprimkhatri77/bmc/util"
)

// helper that creates a random entry for the given account, checks it was saved right and returns it
// it takes the account instead of creating one, so the caller decides which account the entry goes to:
// a fresh one (createRandomEntry(t, createRandomAccount(t))) or the same one many times (TestListEntry)
// an entry records money moving in (+) or out (-) of an account; inserting one doesn't change the account's balance
func createRandomEntry(t *testing.T, account Account) Entry {

	// account_id references accounts.id (foreign key), so it has to be a real account
	// RandomMoney is 0-1000, so these are always deposits, never withdrawals
	arg := CreateEntryParams{
		AccountID: account.ID,
		Amount:    util.RandomMoney(),
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	// the saved values match what we sent
	require.Equal(t, arg.AccountID, entry.AccountID)
	require.Equal(t, arg.Amount, entry.Amount)

	// the db fills these in (bigserial id, default now()), so they shouldn't be their zero value
	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)

	return entry
}

func TestCreateEntry(t *testing.T) {
	createRandomEntry(t, createRandomAccount(t))
}

// creates an entry, fetches it back by id and checks every field matches
func TestGetEntry(t *testing.T) {
	entry1 := createRandomEntry(t, createRandomAccount(t))

	entry2, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.AccountID, entry2.AccountID)
	require.Equal(t, entry1.Amount, entry2.Amount)

	// same moment within 1 second (safer than Equal for times)
	require.WithinDuration(t, entry1.CreatedAt, entry2.CreatedAt, time.Second)

}

// creates 10 entries for one account, then lists the second page of 5
func TestListEntry(t *testing.T) {

	// one account, 10 entries on it
	account := createRandomAccount(t)
	for range 10 {
		createRandomEntry(t, account)
	}

	// ListEntries filters by account_id, so this account has exactly our 10 entries:
	// Offset skips the first 5 of them (ordered by id), Limit returns the next 5
	arg := ListEntriesParams{
		Limit:     5,
		Offset:    5,
		AccountID: account.ID,
	}

	entries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	// exactly 5 came back, because of the limit
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.NotEmpty(t, entry)
		// every entry belongs to our account, proving the WHERE account_id filter works
		require.Equal(t, account.ID, entry.AccountID)
	}
}
