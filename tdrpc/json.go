package tdrpc

import (
	"encoding/json"
	"time"
)

// Time is a specialized handler for times
type Time time.Time

// MarshalJSON forces the formats for times
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02T15:04:05.000000Z07:00") + `"`), nil
}

// MarshalJSON for Account
func (a *Account) MarshalJSON() ([]byte, error) {
	type Alias Account
	return json.Marshal(&struct {
		CreatedAt *Time `json:"created_at"`
		UpdatedAt *Time `json:"updated_at"`
		*Alias
	}{
		CreatedAt: (*Time)(a.CreatedAt),
		UpdatedAt: (*Time)(a.UpdatedAt),
		Alias:     (*Alias)(a),
	})
}

// MarshalJSON for LedgerRecord
func (lr *LedgerRecord) MarshalJSON() ([]byte, error) {
	type Alias LedgerRecord
	return json.Marshal(&struct {
		CreatedAt *Time `json:"created_at"`
		UpdatedAt *Time `json:"updated_at"`
		ExpiresAt *Time `json:"expires_at,omitempty"`
		*Alias
	}{
		CreatedAt: (*Time)(lr.CreatedAt),
		UpdatedAt: (*Time)(lr.UpdatedAt),
		ExpiresAt: (*Time)(lr.ExpiresAt),
		Alias:     (*Alias)(lr),
	})
}

// MarshalJSON for DecodeResponse
func (dr *DecodeResponse) MarshalJSON() ([]byte, error) {
	type Alias DecodeResponse
	ts := time.Unix(dr.Timestamp, 0)
	return json.Marshal(&struct {
		Timestamp *Time `json:"timestamp"`
		*Alias
	}{
		Timestamp: (*Time)(&ts),
		Alias:     (*Alias)(dr),
	})
}
