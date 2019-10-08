package tdrpcserver

import (
	"encoding/hex"
	"time"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// ParsePubKeyHexString parses a hex encoded string to a public key struct
func ParsePubKeyHexString(str string) (*btcec.PublicKey, error) {

	if str == "" {
		return nil, tdrpc.ErrInvalidPubKey
	}

	pubKeyBytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}

	return btcec.ParsePubKey(pubKeyBytes, btcec.S256())
}

// ParseSignatureHexString parses a hex encoded string to a signature struct
func ParseSignatureHexString(str string) (*btcec.Signature, error) {

	if str == "" {
		return nil, tdrpc.ErrInvalidSig
	}

	sig, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}

	return btcec.ParseSignature(sig, btcec.S256())
}

// NewKey generates a new private key
func NewKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey(btcec.S256())
}

// HexEncodedPublicKey returns a hex encoded compressed public key
func HexEncodedPublicKey(key *btcec.PrivateKey) string {
	data := key.PubKey().SerializeCompressed()
	return hex.EncodeToString(data)
}

// ValidateSigntature will validate a signature of anything
func ValidateSigntature(payload string, pubKeyHexString string, sigHexString string) error {

	pubKey, err := ParsePubKeyHexString(pubKeyHexString)
	if err != nil {
		return tdrpc.ErrInvalidPubKey
	}

	sig, err := ParseSignatureHexString(sigHexString)
	if err != nil {
		return tdrpc.ErrInvalidSig
	}

	if !sig.Verify(chainhash.DoubleHashB([]byte(payload)), pubKey) {
		return tdrpc.ErrSigVerficationFailed
	}

	return nil

}

// ValidateTimestampSigntature will validate a timestamp string ensuring proper time window
func ValidateTimestampSigntature(timeString string, pubKeyHexString string, sigHexString string, referenceTime time.Time) error {

	if timeString == "" {
		return tdrpc.ErrInvalidTimestamp
	}

	t, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return tdrpc.ErrInvalidTimestamp
	}

	// If timestamp is +/- 10 mintues from server then fail
	dt := referenceTime.UTC().Sub(t)
	delta := 10 * time.Minute
	if dt < -delta || dt > delta {
		return tdrpc.ErrInvalidTimestampOffset
	}

	return ValidateSigntature(timeString, pubKeyHexString, sigHexString)

}
