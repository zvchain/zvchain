package crontab

import (
	"github.com/zvchain/zvchain/browser/models"
)

type Transfer struct {
}

func (transfer *Transfer) RewardsToAccounts(rewards []*ExploreBlockReward) map[string]uint64 {
	explorerAccount := make([]*models.Account, 0, 0)
	mapData := make(map[string]uint64)
	if rewards != nil && len(rewards) > 0 {
		for _, reward := range rewards {
			account := transfer.blockRewardTOAccount(reward)

			explorerAccount = append(explorerAccount, account...)
		}
	}
	for _, account := range explorerAccount {
		if _, exists := mapData[account.Address]; exists {
			mapData[account.Address] += account.Rewards
		} else {
			mapData[account.Address] = account.Rewards
		}

	}
	return mapData

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
