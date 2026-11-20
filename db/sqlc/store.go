package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store wraps the generated Queries and adds transactions on top:
// single queries go through the embedded Queries, multi-step operations (like TransferTx) through execTx
type Store struct {
	// embedded (no field name), so all the generated query methods are callable straight on the store,
	// e.g. store.GetAccount(...), and they run normally (outside any transaction)
	*Queries
	// the connection pool, kept so execTx can start transactions on it
	db *sql.DB
}

// NewStore creates a Store on top of a connection pool,
// used both for normal queries (New(db)) and for starting transactions (db)
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx runs fn inside a database transaction
// it's a wrapper so we don't write begin/rollback/commit by hand every time:
// fn holds the queries that must succeed or fail together, execTx commits or rolls back based on what fn returns
// lowercase, so only code in this package uses it; outside code calls ready-made ones like TransferTx
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	// start a transaction (nil = default options); if it can't start, nothing ran, so just return
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// same generated queries, but running on tx instead of the pool:
	// Queries only needs a DBTX, and both *sql.DB and *sql.Tx satisfy it
	q := New(tx)
	err = fn(q)
	if err != nil {
		// fn failed, so undo everything it did; if the rollback fails too, return both errors
		// note: this only rolls back on a returned error, not if fn panics (defer tx.Rollback() would cover that)
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	// fn succeeded, so save all its changes at once
	return tx.Commit()
}

// TransferTxParams is the input for a transfer: who sends, who receives, and how much
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account"`
	ToAccountID   int64 `json:"to_account"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult holds everything a transfer creates or changes:
// the transfer row, both accounts after the balance update, and the two entries
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx moves money from one account to another
// it creates a transfer record, adds an entry for each account and updates both balances,
// all in one database transaction, so either every step is saved or none of them is
func (store *Store) TransactionTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	// declared outside the callback so the callback can fill it in and we can return it after
	var result TransferTxResult

	// everything inside the callback runs in one transaction: all of it is saved or none of it
	// always use q in here, not store: store's methods run outside the transaction and wouldn't be rolled back
	err := store.execTx(ctx, func(q *Queries) error {

		// declared once so the lines below can use = and assign straight into result's fields
		var err error

		// 1. record the transfer itself
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		// returning an error makes execTx roll back everything done so far
		if err != nil {
			return err
		}

		// 2. entry for money going out of the sender (negative amount)
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// 3. entry for money coming into the receiver (positive amount)
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 4. take the money out of the sender: one UPDATE (balance = balance + amount) with a negative amount
		// it reads and changes the balance in the same statement, and the UPDATE locks the row until commit,
		// so a concurrent transfer on this account waits instead of reading a stale balance
		// (no separate SELECT ... FOR NO KEY UPDATE needed anymore)
		result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.FromAccountID,
			Amount: -arg.Amount,
		})
		if err != nil {
			return err
		}

		// 5. put the money into the receiver (positive amount)
		// note: two transfers in opposite directions (A->B and B->A at the same time) can still deadlock here,
		// each holding the row lock the other one needs next
		result.ToAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.ToAccountID,
			Amount: arg.Amount,
		})
		if err != nil {
			return err
		}

		// nil tells execTx everything worked, so it commits
		return nil
	})

	// err is nil if committed, or the error that caused the rollback
	return result, err
}
