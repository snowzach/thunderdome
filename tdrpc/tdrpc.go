package tdrpc

// These two endpoints are able to be called on behalf of a user using a privileged key
const (
	CreateGeneratedEndpoint = "/tdrpc.ThunderdomeRPC/CreateGenerated"
	AccountEndpoint         = "/tdrpc.ThunderdomeRPC/Account"
	DecodeEndpoint          = "/tdrpc.ThunderdomeRPC/Decode"
)

// ValueTotal will return the total value of the transaction
func (lr *LedgerRecord) ValueTotal() int64 {
	return lr.Value + lr.NetworkFee + lr.ProcessingFee
}
