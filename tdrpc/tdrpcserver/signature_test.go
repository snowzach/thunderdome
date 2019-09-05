package tdrpcserver

// This file is copied from backend/btc-api/common/signature

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// TestPubKeyString example public key used for signature testing
	TestPubKeyString = "02262233847a69026f8f3ae027af347f2501adf008fe4f6087d31a1d975fd41473"
	// TestSigString example signature used for signature testing
	TestSigString = "3045022100aef1851655cd6e7ccc77afc3cd6c8f7a99de855571cea2dce9e94b17b392228f02206b37f35397018eb64d3f68995e6500d3c761c284d6a67a2509947da9137558d1"
	// TestBodyString example payload used for signature testing
	TestBodyString = "Hello World"
)

func TestParsePubKeyHexString(t *testing.T) {
	pubKey, err := ParsePubKeyHexString(TestPubKeyString)
	assert.Nil(t, err)
	assert.NotNil(t, pubKey)
}

func TestParseSignatureHexString(t *testing.T) {
	sig, err := ParseSignatureHexString(TestSigString)
	assert.Nil(t, err)
	assert.NotNil(t, sig)
}

func TestSignatureVerification(t *testing.T) {
	pubKey, err := ParsePubKeyHexString(TestPubKeyString)
	require.Nil(t, err)

	sig, err := ParseSignatureHexString(TestSigString)
	require.Nil(t, err)

	msg := chainhash.DoubleHashB([]byte(TestBodyString))

	valid := sig.Verify(msg, pubKey)
	assert.True(t, valid)
}

func TestValidateTimestampSigntature(t *testing.T) {

	key, err := NewKey()
	require.Nil(t, err)
	pubKeyHexString := HexEncodedPublicKey(key)

	timeString := time.Now().UTC().Format(time.RFC3339)
	sig, err := key.Sign(chainhash.DoubleHashB([]byte(timeString)))
	require.Nil(t, err)
	sigHexString := hex.EncodeToString(sig.Serialize())

	err = ValidateTimestampSigntature(timeString, pubKeyHexString, sigHexString, time.Now().UTC())
	assert.Nil(t, err)
}

func TestHexEncodedPublicKey(t *testing.T) {
	key, err := NewKey()
	require.Nil(t, err)

	keyString := HexEncodedPublicKey(key)
	require.NotEmpty(t, keyString)

	pubKey, err := ParsePubKeyHexString(keyString)
	assert.Nil(t, err)
	assert.NotNil(t, pubKey)

	if pubKey.IsEqual(key.PubKey()) != true {
		spew.Dump(pubKey)
		spew.Dump(key.PubKey())
		assert.Fail(t, "keys are not equal")
	}
}
