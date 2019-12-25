package rpc

var PruneModeSupportMethods = map[string]struct{}{
	"CheckPointAt":       {},
	"MinerPoolInfo":      {},
	"ProposalTotalStake": {},
	"BalanceByHeight":    {},
}

func IsNotSupportedMethod(method string) bool {
	_, ok := PruneModeSupportMethods[method]
	return ok
}
