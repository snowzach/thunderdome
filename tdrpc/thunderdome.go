package tdrpc

import (
	"context"
	fmt "fmt"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Various Const Definitions
const (
	// InternalIdSuffix used to track internal transactions
	InternalIdSuffix = ":int"

	// ValueSweep is used to indicate we are going to empty the account
	ValueSweep int64 = -1

	// Endpoints
	CreateGeneratedEndpoint = "/tdrpc.ThunderdomeRPC/CreateGenerated"
	AccountEndpoint         = "/tdrpc.ThunderdomeRPC/Account"
	DecodeEndpoint          = "/tdrpc.ThunderdomeRPC/Decode"

	// TempLedgerRecordIdPrefix is used to temporary store ledger record IDs
	TempLedgerRecordIdPrefix = "temp:"

	// These are the metadata fields that we will use to authenticate requests
	MetadataAuthPubKeyString = "cn-auth-pubkeystring"
	MetadataAuthSignature    = "cn-auth-signature"
	MetadataAuthTimestamp    = "cn-auth-timestamp"

	// This is used to determine language settings
	MetadataLocale = "cn-locale"

	// RequestInstant is used on a ledger request to denote that the transaction is an inprocess instant topup
	// The request field will be blanked once the transaction confirms
	RequestInstantPending   = "instant_pending"
	RequestInstantCompleted = "instant_completed"
)

type AccountStats struct {
	Count      int64 `db:"count"`
	Balance    int64 `db:"balance"`
	PendingIn  int64 `db:"pending_in"`
	PendingOut int64 `db:"pending_out"`
}

type LedgerRecordStats struct {
	Count int64 `db:"count"`
	Value int64 `db:"value"`
}

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
	GetLedgerRecordStats(ctx context.Context, filter map[string]string, after time.Time) (*LedgerRecordStats, error)
	GetActiveGeneratedLightningLedgerRequest(ctx context.Context, accountID string) (*LedgerRecord, error)
	ExpireLedgerRequests(ctx context.Context) error
	GetAccountStats(ctx context.Context) (*AccountStats, error)
}

// FormatsInt will format any integer type with commas. It attempts to determine the language from the
func FormatInt(ctx context.Context, n interface{}) string {
	return message.NewPrinter(language.English).Sprintf("%d", n)
}

// ValueTotal will return the total value of the transaction
func (lr *LedgerRecord) ValueTotal() int64 {

	// NetworkFee and ProcessingFee are not taken into account inbound
	if lr.Direction == IN {
		return lr.Value
	} else if lr.Direction == OUT {
		// Otherwise it's outbound and all fees are taken into account
		return lr.Value + lr.NetworkFee + lr.ProcessingFee
	}

	// Otherwise this should never be possible
	panic(fmt.Sprintf("invalid direction: %v", lr.Direction))

}
