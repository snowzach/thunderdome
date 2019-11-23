package tdrpc

import (
	"database/sql/driver"
	"fmt"
)

// Scan decodes the database ledger type
func (t *LedgerRecord_Type) Scan(src interface{}) error {

	var typeString string
	switch st := src.(type) {
	case string:
		typeString = st
	case []uint8:
		typeString = string(st)
	default:
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

// MarshalJSON implements the json marshaler
func (t LedgerRecord_Type) MarshalJSON() ([]byte, error) {

	switch t {
	case BTC:
		return []byte(`"btc"`), nil
	case LIGHTNING:
		return []byte(`"lightning"`), nil
	}

	return nil, fmt.Errorf("Unknown type %v", t)

}

// UnmarshalJSON implements the json unmarshaller
func (t *LedgerRecord_Type) UnmarshalJSON(in []byte) error {

	switch string(in) {
	case `"btc"`:
		*t = BTC
		return nil
	case `"lightning"`:
		*t = LIGHTNING
		return nil
	}

	return fmt.Errorf("Unknown type %s", in)

}

// String implements the stringer interface
func (t LedgerRecord_Type) String() string {

	switch t {
	case BTC:
		return "btc"
	case LIGHTNING:
		return "lightning"
	}

	return "unknown"
}
