package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// GetAccounts fetches a accounts with filter and pagination
func (c *Client) GetAccounts(ctx context.Context, filter map[string]string, offset int, limit int) ([]*tdrpc.Account, error) {

	var queryClause string
	var queryParams = []interface{}{}

	// Validate the filters
	for filter, value := range filter {
		switch filter {
		case "id":
			if value == "" {
				return nil, fmt.Errorf("Invalid value for id")
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND id = $%d", len(queryParams))
		case "address":
			if value == "" {
				return nil, fmt.Errorf("Invalid value for address")
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND address = $%d", len(queryParams))
		default:
			return nil, fmt.Errorf("Unsupported filter %s", filter)

		}
	}

	if limit > 0 {
		queryClause += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		queryClause += fmt.Sprintf(" OFFSET %d", offset)
	}

	var accounts = make([]*tdrpc.Account, 0)
	err := c.db.SelectContext(ctx, &accounts, `SELECT * FROM account WHERE 1=1`+queryClause, queryParams...)
	if err != nil {
		return accounts, err
	}

	return accounts, nil
}

// GetAccountByID fetches a account by ID
func (c *Client) GetAccountByID(ctx context.Context, id string) (*tdrpc.Account, error) {

	account := new(tdrpc.Account)
	err := c.db.GetContext(ctx, account, `SELECT * FROM account WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountByAddress fetches a account by address
func (c *Client) GetAccountByAddress(ctx context.Context, address string) (*tdrpc.Account, error) {

	account := new(tdrpc.Account)
	err := c.db.GetContext(ctx, account, `SELECT * FROM account WHERE address = $1`, address)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return account, nil
}

// SaveAccount creates/updates a account
func (c *Client) SaveAccount(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error) {

	err := c.db.GetContext(ctx, account, `
		INSERT INTO account (id, created_at, updated_at, address, balance, pending_in, pending_out, locked)
		VALUES($1, NOW(), NOW(), $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET
		updated_at = NOW(),
		address = $2,
		balance = $3,
		pending_in = $4,
		pending_out = $5,
		locked = $6
		RETURNING *
	`, account.Id, account.Address, account.Balance, account.PendingIn, account.PendingOut, account.Locked)
	if err != nil {
		return account, err
	}
	return account, nil

}
