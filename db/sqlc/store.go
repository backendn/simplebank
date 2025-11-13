package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (s *Store) execTx(ctx context.Context, fn func(q *Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := New(tx)

	err = fn(q)

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err :%v, rbErr :%v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_acount_id"`
	Amount        int64 `json:"amount"`
}

type TransferTxResultParams struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account_id"`
	ToAccount   Account  `json:"to_acount_id"`
	ToEntry     Entry    `json:"to_entry"`
	FromEntry   Entry    `json:"from_entry"`
}

func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResultParams, error) {
	var result TransferTxResultParams

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(arg))
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = AddMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)

		} else {
			result.ToAccount, result.FromAccount, err = AddMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}

		return nil
	})
	return result, err
}
func AddMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64) (account1 Account, account2 Account, err error) {
	account1, err = q.AddMoneyToAccount(ctx, AddMoneyToAccountParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}
	account2, err = q.AddMoneyToAccount(ctx, AddMoneyToAccountParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
