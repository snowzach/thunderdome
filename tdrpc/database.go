package tdrpc

import (
	"database/sql/driver"
	"fmt"
)

// Scan decodes the database ledger status
func (status *LedgerRecord_Status) Scan(src interface{}) error {

	statusString, ok := src.(string)
	if !ok {
		return fmt.Errorf("Invalid status type: %T=%v", src, src)
	}

	switch statusString {
	case "pending":
		*status = PENDING
	case "completed":
		*status = COMPLETED
	case "expired":
		*status = EXPIRED
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
	}

	return nil, fmt.Errorf("Unknown status %v", status)

}

// Scan decodes the database ledger type
func (t *LedgerRecord_Type) Scan(src interface{}) error {

	typeString, ok := src.(string)
	if !ok {
		return fmt.Errorf("Invalid type type: %T=%v", src, src)
	}

	switch typeString {
	case "btc":
		*t = BTC
	case "lightning":
		*t = LIGHTNING
	default:
		return fmt.Errorf("Unknown type %s", typeString)
	}

	return nil

}

// Value encodes the database ledger type
func (t LedgerRecord_Type) Value() (driver.Value, error) {

	switch t {
	case BTC:
		return "btc", nil
	case LIGHTNING:
		return "lightning", nil
	}

	return nil, fmt.Errorf("Unknown type %v", t)

}

// Scan decodes the database ledger direction
func (direction *LedgerRecord_Direction) Scan(src interface{}) error {

	directionString, ok := src.(string)
	if !ok {
		return fmt.Errorf("Invalid direction type: %T=%v", src, src)
	}

	switch directionString {
	case "in":
		*direction = IN
	case "out":
		*direction = OUT
	default:
		return fmt.Errorf("Unknown direction %s", directionString)
	}

	return nil

}

// Value encodes the database ledger direction
func (direction LedgerRecord_Direction) Value() (driver.Value, error) {

	switch direction {
	case IN:
		return "in", nil
	case OUT:
		return "out", nil
	}

	return nil, fmt.Errorf("Unknown type %v", direction)

}
