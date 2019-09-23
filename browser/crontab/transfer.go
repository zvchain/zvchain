package crontab

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"strconv"
)

type Transfer struct {
}

func (transfer *Transfer) RewardsToAccounts(rewards []*ExploreBlockReward) map[string]float64 {
	explorerAccount := make([]*models.AccountList, 0, 0)
	mapData := make(map[string]float64)
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
func (transfer *Transfer) blockRewardTOAccount(reward *ExploreBlockReward) []*models.AccountList {
	accounts := make([]*models.AccountList, 0, 0)
	account := &models.AccountList{
		Address: reward.ProposalID,
		Rewards: float64(reward.ProposalReward + reward.ProposalGasFeeReward),
	}

	accounts = append(accounts, account)
	targets := reward.VerifierReward.TargetIDs
	gas := fmt.Sprintf("%.9f", float64(reward.VerifierGasFeeReward)/float64(len(targets)))
	rewarMoney, _ := strconv.ParseFloat(gas, 64)
	for i := 0; i < len(targets); i++ {
		account := &models.AccountList{
			Address: targets[i].GetAddrString(),
			Rewards: float64(reward.VerifierReward.Value) + rewarMoney,
		}
		accounts = append(accounts, account)
	}
	return accounts

}
