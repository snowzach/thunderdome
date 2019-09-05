package tdrpc

import (
	"database/sql/driver"
	"fmt"
)

// Scan decodes the database ledger status
func (status *LedgerRecord_Status) Scan(src interface{}) error {

	var statusString string
	switch st := src.(type) {
	case string:
		statusString = st
	case []uint8:
		statusString = string(st)
	default:
		return fmt.Errorf("Invalid status type: %T=%v", src, src)
	}

	switch statusString {
	case "pending":
		*status = PENDING
	case "completed":
		*status = COMPLETED
	case "expired":
		*status = EXPIRED
	case "failed":
		*status = FAILED
	default:
		return fmt.Errorf("Unknown status %s", statusString)
	}

	return nil

}

// Value encodes the database ledger status
func (status LedgerRecord_Status) Value() (driver.Value, error) {

	switch status {
	case PENDING:
		return "pending", nil
	case COMPLETED:
		return "completed", nil
	case EXPIRED:
		return "expired", nil
	case FAILED:
		return "failed", nil
	}

	return nil, fmt.Errorf("Unknown status %v", status)

}

// MarshalJSON implements the json marshaler
func (status LedgerRecord_Status) MarshalJSON() ([]byte, error) {

	switch status {
	case PENDING:
		return []byte(`"pending"`), nil
	case COMPLETED:
		return []byte(`"completed"`), nil
	case EXPIRED:
		return []byte(`"expired"`), nil
	case FAILED:
		return []byte(`"failed"`), nil
	}

	return nil, fmt.Errorf("Unknown type %v", status)

}

// UnmarshalJSON implements the json unmarshaller
func (status *LedgerRecord_Status) UnmarshalJSON(in []byte) error {

	switch string(in) {
	case `"pending"`:
		*status = PENDING
	case `"completed"`:
		*status = COMPLETED
	case `"expired"`:
		*status = EXPIRED
	case `"failed"`:
		*status = FAILED
	}

	return fmt.Errorf("Unknown status %s", in)

}

// String implements the stringer interface
func (status LedgerRecord_Status) String() string {

	switch status {
	case PENDING:
		return "pending"
	case COMPLETED:
		return "completed"
	case EXPIRED:
		return "expired"
	case FAILED:
		return "failed"
	}

	return "unknown"
}
