package postgres

import (
	"context"
	"database/sql"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// AccountGetByID fetches a account by ID
func (c *Client) AccountGetByID(ctx context.Context, id string) (*tdrpc.Account, error) {

	account := new(tdrpc.Account)
	err := c.db.GetContext(ctx, account, `SELECT * FROM account WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return account, nil
}

// AccountGetByAddress fetches a account by address
func (c *Client) AccountGetByAddress(ctx context.Context, address string) (*tdrpc.Account, error) {

	account := new(tdrpc.Account)
	err := c.db.GetContext(ctx, account, `SELECT * FROM account WHERE address = $1`, address)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return account, nil
}

// AccountSave creates/updates a account
func (c *Client) AccountSave(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error) {

	err := c.db.GetContext(ctx, account, `
		INSERT INTO account (id, created_at, updated_at, address, balance, pending_in, pending_out)
		VALUES($1, NOW(), NOW(), $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE
		SET
		updated_at = NOW(),
		address = $2,
		balance = $3,
		pending_in = $4,
		pending_out = $5
		RETURNING *
	`, account.Id, account.Address, account.Balance, account.PendingIn, account.PendingOut)
	if err != nil {
		return account, err
	}
	return account, nil

}
