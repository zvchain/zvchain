package crontab

import (
	"github.com/zvchain/zvchain/browser/models"
)

type Transfer struct {
}

func (transfer *Transfer) BlockRewardTOAccount(reward *ExploreBlockReward) []*models.Account {
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
