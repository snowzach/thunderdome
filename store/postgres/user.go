package postgres

import (
	"context"
	"database/sql"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (c *Client) UserGetByID(ctx context.Context, id string) (*tdrpc.User, error) {

	user := new(tdrpc.User)
	err := c.db.GetContext(ctx, user, `SELECT * FROM "user" WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return user, nil
}

func (c *Client) UserGetByLogin(ctx context.Context, login string) (*tdrpc.User, error) {

	user := new(tdrpc.User)
	err := c.db.GetContext(ctx, user, `SELECT * FROM "user" WHERE login = $1`, login)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return user, nil
}

// UserSave creates/updates a user
func (c *Client) UserSave(ctx context.Context, user *tdrpc.User) (*tdrpc.User, error) {

	err := c.db.GetContext(ctx, user, `
		INSERT INTO "user" (login, address, balance)
		VALUES($1, $2, $3)
		ON CONFLICT (login) DO UPDATE
		SET 
		login = $1,
		address = $2,
		balance = $3
		RETURNING *
	`, user.Login, user.Address, user.Balance)
	if err != nil {
		return user, err
	}
	return user, nil

}
