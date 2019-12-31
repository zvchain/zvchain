package rpc

var PruneModeSupportMethods = map[string]struct{}{
	"MinerPoolInfo":      {},
	"ProposalTotalStake": {},
	"BalanceByHeight":    {},
}

func IsNotSupportedMethod(method string) bool {
	_, ok := PruneModeSupportMethods[method]
	return ok
}
