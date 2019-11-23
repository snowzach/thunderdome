package tdrpc

import (
	"database/sql/driver"
	"fmt"
)

// Scan decodes the database ledger direction
func (direction *LedgerRecord_Direction) Scan(src interface{}) error {

	var directionString string
	switch st := src.(type) {
	case string:
		directionString = st
	case []uint8:
		directionString = string(st)
	default:
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

// MarshalJSON implements the json marshaler
func (direction LedgerRecord_Direction) MarshalJSON() ([]byte, error) {

	switch direction {
	case IN:
		return []byte(`"in"`), nil
	case OUT:
		return []byte(`"out"`), nil
	}

	return nil, fmt.Errorf("Unknown direction %v", direction)

}

// UnmarshalJSON implements the json unmarshaller
func (direction *LedgerRecord_Direction) UnmarshalJSON(in []byte) error {

	switch string(in) {
	case `"in"`:
		*direction = IN
		return nil
	case `"out"`:
		*direction = OUT
		return nil
	}

	return fmt.Errorf("Unknown direction %s", in)

}

// String implements the stringer interface
func (direction LedgerRecord_Direction) String() string {

	switch direction {
	case IN:
		return "in"
	case OUT:
		return "out"
	}

	return "unknown"
}
