package tdrpc

// ValueTotal will return the total value of the transaction
func (lr *LedgerRecord) ValueTotal() int64 {
	return lr.Value + lr.NetworkFee + lr.ProcessingFee
}
