package main

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"log"
)

func main() {
	// create new client instance
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		HTTPPostMode: true,
		DisableTLS:   true,
		Host:         "127.0.0.1:18443",
		User:         "bitcoinrpc",
		Pass:         "bitcoinrpc",
	}, nil)
	if err != nil {
		log.Fatalf("error creating new btc client: %v", err)
	}

	// This should have 2 outputs
	hash, err := chainhash.NewHashFromStr("5bf7402ae9f5e193bada8c0bfa83fb9fb4daad4650f60c4a914e4e8cb65af6fb")
	if err != nil {
		log.Fatalf("Could not create hash: %v", err)
	}

	// // Create an array of chains such that we can pick the one we want
	// chains := []*chaincfg.Params{
	// 	&chaincfg.MainNetParams,
	// 	&chaincfg.RegressionNetParams,
	// 	&chaincfg.SimNetParams,
	// 	&chaincfg.TestNet3Params,
	// }
	// // Find the selected chain
	// for _, cp := range chains {
	// 	if config.GetString("extractor.btc.chain") == cp.Name {
	// 		e.chainParams = cp
	// 		break
	// 	}
	// }
	// if e.chainParams == nil {
	// 	return nil, fmt.Errorf("Could not find chain %s", config.GetString("extractor.btc.chain"))
	// }

	// list accounts
	rawTx, err := client.GetRawTransaction(hash)
	if err != nil {
		log.Fatalf("error getting tx: %v", err)
	}
	wTx := rawTx.MsgTx()

	// Parse all of the outputs
	for height, vout := range wTx.TxOut {
		// Attempt to parse simple addresses out of the script
		scriptType, addresses, _, err := txscript.ExtractPkScriptAddrs(vout.PkScript, &chaincfg.RegressionNetParams)
		if err != nil { // Could not decode
			log.Printf("H: %d TYPE: %v ", height, txscript.NonStandardTy.String())
		} else {
			log.Printf("H: %d TYPE: %v ADD: %v VALUE: %v", height, scriptType.String(), parseBTCAddresses(addresses), vout.Value)
		}
	}

	// // iterate over accounts (map[string]btcutil.Amount) and write to stdout
	// for label, amount := range accounts {
	// 	log.Printf("%s: %s", label, amount)
	// }

	// // prepare a sendMany transaction
	// receiver1, err := btcutil.DecodeAddress("1someAddressThatIsActuallyReal", &chaincfg.MainNetParams)
	// if err != nil {
	// 	log.Fatalf("address receiver1 seems to be invalid: %v", err)
	// }
	// receiver2, err := btcutil.DecodeAddress("1anotherAddressThatsPrettyReal", &chaincfg.MainNetParams)
	// if err != nil {
	// 	log.Fatalf("address receiver2 seems to be invalid: %v", err)
	// }
	// receivers := map[btcutil.Address]btcutil.Amount{
	// 	receiver1: 42,  // 42 satoshi
	// 	receiver2: 100, // 100 satoshi
	// }

	// // create and send the sendMany tx
	// txSha, err := client.SendMany("some-account-label-from-which-to-send", receivers)
	// if err != nil {
	// 	log.Fatalf("error sendMany: %v", err)
	// }
	// log.Printf("sendMany completed! tx sha is: %s", txSha.String())
}

// This coverts []btcutil.Address to []string
func parseBTCAddresses(in []btcutil.Address) []string {
	ret := make([]string, len(in), len(in))
	for x, y := range in {
		ret[x] = y.EncodeAddress()
	}
	return ret
}
