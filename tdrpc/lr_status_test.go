package tdrpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusUnmarshalJSON(t *testing.T) {

	exampleJSON := `"completed"`
	var value LedgerRecord_Status

	err := json.Unmarshal([]byte(exampleJSON), &value)
	assert.Nil(t, err)

	assert.Equal(t, COMPLETED, value)
}
