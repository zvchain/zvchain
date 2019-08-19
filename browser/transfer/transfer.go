package transfer

import (
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/cmd/gtas/cli"
)

type Transfer struct {
}

func (transfer *Transfer) BlockRewardTOAccount(reward cli.ExploreBlockReward) []*models.Account {
	accounts := make([]*models.Account, 0)
	account := &models.Account{}
	account.Address = reward.ProposalID
	account.Rewards = reward.ProposalReward
	accounts = append(accounts, account)
	targets := reward.VerifierReward.TargetIDs
	for i := 0; i < len(targets); i++ {
		account := &models.Account{}
		account.Address = targets[i].GetHexString()
		account.Rewards = reward.VerifierReward.Value
		accounts = append(accounts, account)
	}
	return accounts

}
