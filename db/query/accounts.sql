-- name: CreateAccount :one
INSERT INTO accounts (owner, balance, currency)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- name: GetAccount :one
SELECT
    *
FROM
    accounts
WHERE
    id = $1
LIMIT 1;

-- Same as GetAccount, but also locks the row until the transaction ends, so another
-- transaction that wants to lock it waits instead of reading a balance that's about to change.
-- FOR NO KEY UPDATE, not FOR UPDATE: inserting a transfer or entry makes postgres take a
-- FOR KEY SHARE lock on the referenced account (the foreign key check), and FOR UPDATE conflicts
-- with that, which deadlocked two concurrent transfers. NO KEY UPDATE promises we won't change
-- the id, so it doesn't conflict with the foreign key lock.
-- Not used by TransferTx anymore: AddAccountBalance's UPDATE takes the same lock itself.
-- name: GetAccountForUpdate :one
SELECT
    *
FROM
    accounts
WHERE
    id = $1
LIMIT 1
FOR NO KEY UPDATE;

-- name: ListAccounts :many
SELECT
    *
FROM
    accounts
ORDER BY
    id
LIMIT $1 OFFSET $2;

-- name: UpdateAccount :one
UPDATE
    accounts
SET
    balance = $2
WHERE
    id = $1
RETURNING
    *;

-- Adds amount to the balance in one statement: postgres reads the current balance and writes
-- the new one itself, locking the row until commit, so concurrent transfers can't overwrite
-- each other (no separate SELECT first). Pass a negative amount to take money out.
-- sqlc.arg(amount) names the Go field Amount; with $2 sqlc would name it Balance (after the
-- column it's added to), which reads as if it sets the balance.
-- name: AddAccountBalance :one
UPDATE
    accounts
SET
    balance = balance + sqlc.arg (amount)
WHERE
    id = sqlc.arg (id)
RETURNING
    *;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;

