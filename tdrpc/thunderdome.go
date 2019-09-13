package tdrpc

import (
	"context"
	"time"
)

// Various Const Definitions
const (
	// InternalIdSuffix used to track internal transactions
	InternalIdSuffix = ":int"

	// ValueSweep is used to indicate we are going to empty the account
	ValueSweep int64 = -1

	// TempLedgerRecordIdPrefix is used to temporary store ledger record IDs
	TempLedgerRecordIdPrefix = "temp:"

	// These are the metadata fields that we will use to authenticate requests
	MetadataAuthPubKeyString = "cn-auth-pubkeystring"
	MetadataAuthSignature    = "cn-auth-signature"
	MetadataAuthTimestamp    = "cn-auth-timestamp"
)

type Store interface {
	GetAccounts(ctx context.Context, filter map[string]string, offset int, limit int) ([]*Account, error)
	GetAccountByID(ctx context.Context, accountID string) (*Account, error)
	GetAccountByAddress(ctx context.Context, address string) (*Account, error)
	SaveAccount(ctx context.Context, account *Account) (*Account, error)
	ProcessLedgerRecord(ctx context.Context, lr *LedgerRecord) error
	ProcessInternal(ctx context.Context, id string) (*LedgerRecord, error)
	UpdateLedgerRecordID(ctx context.Context, oldID string, newID string) error
	GetLedger(ctx context.Context, filter map[string]string, after time.Time, offset int, limit int) ([]*LedgerRecord, error)
	GetLedgerRecord(ctx context.Context, id string, direction LedgerRecord_Direction) (*LedgerRecord, error)
	GetActiveGeneratedLightningLedgerRequest(ctx context.Context, accountID string) (*LedgerRecord, error)
	ExpireLedgerRequests(ctx context.Context) error
}
