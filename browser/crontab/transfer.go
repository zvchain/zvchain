package crontab

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"strconv"
)

type Transfer struct {
}

func (transfer *Transfer) RewardsToAccounts(rewards []*ExploreBlockReward) *BlockReward {
	explorerAccount := make([]*models.AccountList, 0, 0)
	mapData := make(map[string]float64)
	mapCount := make([]map[string]map[string]uint64, 0, 0)
	if rewards != nil && len(rewards) > 0 {
		for _, reward := range rewards {
			account, mapdata := transfer.blockRewardTOAccount(reward)
			explorerAccount = append(explorerAccount, account...)
			mapCount = append(mapCount, mapdata)
		}
	}
	for _, account := range explorerAccount {
		if _, exists := mapData[account.Address]; exists {
			mapData[account.Address] += account.Rewards
		} else {
			mapData[account.Address] = account.Rewards
		}

	}
	mapCountplus := make(map[string]map[string]uint64)
	for _, count := range mapCount {
		for addr, data := range count {
			if _, exists := mapCountplus[addr]; exists {
				mapCountplus[addr]["proposal_count"] += data["proposal_count"]
				mapCountplus[addr]["verify_count"] += data["verify_count"]
			} else {
				mapCountplus[addr] = map[string]uint64{}
				mapCountplus[addr]["proposal_count"] = data["proposal_count"]
				mapCountplus[addr]["verify_count"] = data["verify_count"]
			}
		}
	}
	reward := &BlockReward{
		MapReward:     mapData,
		MapBlockCount: mapCountplus,
	}
	return reward

}

func (transfer *Transfer) blockRewardTOAccount(reward *ExploreBlockReward) ([]*models.AccountList, map[string]map[string]uint64) {
	accounts := make([]*models.AccountList, 0, 0)
	account := &models.AccountList{
		Address: reward.ProposalID,
		Rewards: float64(reward.ProposalReward + reward.ProposalGasFeeReward),
	}
	address := reward.ProposalID
	mapCount := make(map[string]map[string]uint64)
	if _, exists := mapCount[address]; exists {
		mapCount[address]["proposal_count"] += 1
	} else {
		mapCount[address] = map[string]uint64{}
		mapCount[address]["proposal_count"] = 1
		mapCount[address]["verify_count"] = 0

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
		addr := targets[i].GetAddrString()
		if _, exists := mapCount[addr]; exists {
			mapCount[addr]["verify_count"] += 1
		} else {
			mapCount[addr] = map[string]uint64{}
			mapCount[addr]["verify_count"] = 1
			mapCount[addr]["proposal_count"] = 0

		}
	}

	return accounts, mapCount

}
