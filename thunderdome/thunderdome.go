package thunderdome

import (
	"context"
	"time"

	"git.coinninja.net/backend/thunderdome/tdrpc"
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
	GetAccounts(ctx context.Context, filter map[string]string, offset int, limit int) ([]*tdrpc.Account, error)
	GetAccountByID(ctx context.Context, accountID string) (*tdrpc.Account, error)
	GetAccountByAddress(ctx context.Context, address string) (*tdrpc.Account, error)
	SaveAccount(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error)
	ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error
	ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error)
	UpdateLedgerRecordID(ctx context.Context, oldID string, newID string) error
	GetLedger(ctx context.Context, filter map[string]string, after time.Time, offset int, limit int) ([]*tdrpc.LedgerRecord, error)
	GetLedgerRecord(ctx context.Context, id string, direction tdrpc.LedgerRecord_Direction) (*tdrpc.LedgerRecord, error)
	GetActiveGeneratedLightningLedgerRequest(ctx context.Context, accountID string) (*tdrpc.LedgerRecord, error)
	ExpireLedgerRequests(ctx context.Context) error
}
