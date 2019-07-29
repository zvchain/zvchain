package core

import (
	"testing"
)

func TestRewardManager_CalculateGasFeeVerifyRewards(t *testing.T) {
	rm := rewardManager{}
	gasFee := uint64(100)
	rewards := rm.calculateGasFeeVerifyRewards(gasFee)
	correctRewards := gasFee * gasFeeVerifyRewardsWeight / gasFeeTotalRewardsWeight
	if rewards != correctRewards {
		t.Errorf("calculateGasFeeVerifyRewards: rewards error, wanted: %d, got: %d",
			correctRewards, rewards)
	}
}

func TestRewardManager_CalculateGasFeeCastorRewards(t *testing.T) {
	rm := rewardManager{}
	gasFee := uint64(100)
	rewards := rm.calculateGasFeeCastorRewards(gasFee)
	correctRewards := gasFee * gasFeeCastorRewardsWeight / gasFeeTotalRewardsWeight
	if rewards != correctRewards {
		t.Errorf("calculateGasFeeVerifyRewards: rewards error, wanted: %d, got: %d",
			correctRewards, rewards)
	}
}

func TestRewardManager_Rewards(t *testing.T) {
	rm := NewRewardManager()
	for i := uint64(0); i < 120000000; i++ {
		blockRewards := rm.blockRewards(i)
		userNodeRewards := rm.userNodesRewards(i)
		correctUserNodeRewards := blockRewards * userNodeWeight / totalNodeWeight
		if userNodeRewards != correctUserNodeRewards {
			t.Errorf("userNodesRewards: rewards error, wanted: %d, got: %d",
				correctUserNodeRewards, userNodeRewards)
		}
		daemonNodeWeight := initialDaemonNodeWeight + i/adjustWeightPeriod*adjustWeight
		daemonNodeRewards := rm.daemonNodesRewards(i)
		correctDaemonNodeRewards := blockRewards * daemonNodeWeight / totalNodeWeight
		if daemonNodeRewards != correctDaemonNodeRewards {
			t.Errorf("daemonNodesRewards: rewards error, wanted: %d, got: %d",
				correctDaemonNodeRewards, userNodeRewards)
		}
		minerNodeRewards := rm.minerNodesRewards(i)
		minerNodeWeight := initialMinerNodeWeight - i/adjustWeightPeriod*adjustWeight
		correctMinerNodeRewards := blockRewards * minerNodeWeight / totalNodeWeight
		if minerNodeRewards != correctMinerNodeRewards {
			t.Errorf("minerNodesRewards: rewards error, wanted: %d, got: %d",
				correctMinerNodeRewards, minerNodeRewards)
		}
		castorRewards := rm.calculateCastorRewards(i)
		correctCastorRewards := minerNodeRewards * castorRewardsWeight / totalRewardsWeight
		if castorRewards != correctCastorRewards {
			t.Errorf("calculateCastorRewards: rewards error, wanted: %d, got: %d",
				correctCastorRewards, castorRewards)
		}
		packedRewards := rm.calculatePackedRewards(i)
		correctPackedRewards := minerNodeRewards * packedRewardsWeight / totalRewardsWeight
		if packedRewards != correctPackedRewards {
			t.Errorf("calculatePackedRewards: rewards error, wanted: %d, got: %d",
				correctPackedRewards, packedRewards)
		}
		verifyRewards := rm.calculateVerifyRewards(i)
		correctVerifyRewards := minerNodeRewards * verifyRewardsWeight / totalRewardsWeight
		if verifyRewards != correctVerifyRewards {
			t.Errorf("calculateVerifyRewards: rewards error, wanted: %d, got: %d",
				correctVerifyRewards, verifyRewards)
		}
	}
}
