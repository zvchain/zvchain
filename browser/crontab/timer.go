package crontab

import (
	"fmt"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"sync"
	"time"
)

var VoteTimerSet sync.Map
var voterLock sync.Mutex

type VoteTimer struct {
	VoteStage          uint8
	StartTime, EndTime time.Time
}

func NewVoteTimer() *VoteTimer {
	return new(VoteTimer)
}

func (v *VoteTimer) SetVoteStage(status uint8) *VoteTimer {
	v.VoteStage = status
	return v
}

func (v *VoteTimer) SetStartTime(start time.Time) *VoteTimer {
	v.StartTime = start
	return v
}

func (v *VoteTimer) SetEndTime(end time.Time) *VoteTimer {
	v.EndTime = end
	return v
}

func HandleVoteTimer(key uint64, voteTimer *VoteTimer) {

	VoteTimerSet.Store(key, struct{}{})
	if voteTimer.VoteStage == models.VoteStatusNotBegin {
		go UpdateVoteStatus(key, voteTimer.StartTime.Sub(time.Now()), false)
		go UpdateVoteStatus(key, voteTimer.EndTime.Sub(time.Now()), true)
	} else if voteTimer.VoteStage == models.VoteStatusInProcess {
		go UpdateVoteStatus(key, voteTimer.EndTime.Sub(time.Now()), true)
	}

}

func UpdateVoteStatus(key uint64, d time.Duration, end bool) {
	voterLock.Lock()
	defer voterLock.Unlock()

	var sql string
	if end {
		sql = fmt.Sprintf("update votes set status = %d  where vote_id = %d", models.VoteStatusEnded, key)
	} else {
		sql = fmt.Sprintf("update votes set status = %d  where vote_id = %d", models.VoteStatusInProcess, key)
	}

	updateVoteStatus := func() {
		db := GlobalCrontab
		if db.storage != nil {
			err := db.storage.GetDB().Model(&models.Vote{}).Exec(sql).Error
			if err != nil {
				browserlog.BrowserLog.Error("updateVoteStatus err: ", err)
				return
			}
		}

		// count the votes
		if end {
			CountVotes(key)
		}
		VoteTimerSet.Delete(key)
	}
	time.AfterFunc(d, updateVoteStatus)
}

func ResetVoteTimer() {
	voteNotBegin := make([]models.Vote, 0)
	db := GlobalCrontab
	db.storage.GetDB().Model(&models.Vote{}).
		Where("valid = ? and status = ? ", true, models.VoteStatusNotBegin).
		Find(&voteNotBegin)
	for _, v := range voteNotBegin {
		if _, exist := VoteTimerSet.Load(v.VoteId); !exist {
			VoteTimerSet.Store(v.VoteId, struct{}{})
			go UpdateVoteStatus(v.VoteId, v.StartTime.Sub(time.Now()), false)
			go UpdateVoteStatus(v.VoteId, v.EndTime.Sub(time.Now()), true)
		}
	}
	voteInProcess := make([]models.Vote, 0)
	db.storage.GetDB().Model(&models.Vote{}).
		Where("valid = ? and status = ? ", true, models.VoteStatusInProcess).
		Find(&voteInProcess)
	for _, v := range voteInProcess {
		if _, exist := VoteTimerSet.Load(v.VoteId); !exist {
			VoteTimerSet.Store(v.VoteId, struct{}{})
			go UpdateVoteStatus(v.VoteId, v.EndTime.Sub(time.Now()), true)
		}
	}
}
