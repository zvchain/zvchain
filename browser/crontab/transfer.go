package crontab

import (
	"github.com/zvchain/zvchain/browser/models"
)

type Transfer struct {
}

func (transfer *Transfer) RewardsToAccounts(rewards []*ExploreBlockReward) []*models.Account {
	exploreraccount := make([]*models.Account, 0, 0)
	if rewards != nil && len(rewards) > 0 {
		for _, reward := range rewards {
			account := transfer.blockRewardTOAccount(reward)
			exploreraccount = append(exploreraccount, account...)
		}
	}
	return exploreraccount

}
func (transfer *Transfer) blockRewardTOAccount(reward *ExploreBlockReward) []*models.Account {
	accounts := make([]*models.Account, 0, 0)
	account := &models.Account{
		Address: reward.ProposalID,
		Rewards: reward.ProposalReward + reward.ProposalGasFeeReward,
	}

	accounts = append(accounts, account)
	targets := reward.VerifierReward.TargetIDs
	for i := 0; i < len(targets); i++ {
		account := &models.Account{
			Address: targets[i].GetAddrString(),
			Rewards: reward.VerifierReward.Value,
		}
		accounts = append(accounts, account)
	}
	return accounts

}
