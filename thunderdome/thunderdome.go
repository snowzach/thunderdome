package thunderdome

import (
	"context"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

const (
	// InternalIdSuffix used to track internal transactions
	InternalIdSuffix = ":int"

	// TempLedgerRecordIdPrefix is used to temporary store ledger record IDs
	TempLedgerRecordIdPrefix = "temp:"

	// These are the metadata fields that we will use to authenticate requests
	MetadataAuthPubKeyString = "cn-auth-pubkeystring"
	MetadataAuthSignature    = "cn-auth-signature"
	MetadataAuthTimestamp    = "cn-auth-timestamp"
)

type Store interface {
	AccountGetByID(ctx context.Context, accountID string) (*tdrpc.Account, error)
	AccountGetByAddress(ctx context.Context, address string) (*tdrpc.Account, error)
	AccountSave(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error)
	ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error
	ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error)
	UpdateLedgerRecordID(ctx context.Context, oldID string, newID string) error
	GetLedger(ctx context.Context, accountID string) ([]*tdrpc.LedgerRecord, error)
	GetLedgerRecord(ctx context.Context, id string, direction tdrpc.LedgerRecord_Direction) (*tdrpc.LedgerRecord, error)
}
