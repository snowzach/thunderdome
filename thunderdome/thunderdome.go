package thunderdome

import (
	"context"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type Store interface {
	AccountGetByID(ctx context.Context, accountID string) (*tdrpc.Account, error)
	AccountGetByAddress(ctx context.Context, address string) (*tdrpc.Account, error)
	AccountSave(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error)
	ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error
	ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error)
	GetLedger(ctx context.Context, accountID string) ([]*tdrpc.LedgerRecord, error)
	GetLedgerRecord(ctx context.Context, id string, direction tdrpc.LedgerRecord_Direction) (*tdrpc.LedgerRecord, error)
}
