package rpc

var PruneModeSupportMethods = map[string]struct{}{
	"CheckPointAt":       struct{}{},
	"MinerPoolInfo":      struct{}{},
	"ProposalTotalStake": struct{}{},
	"BalanceByHeight":    struct{}{},
}

func IsNotSuppotedMethod(method string) bool {
	_, ok := PruneModeSupportMethods[method]
	return ok
}
