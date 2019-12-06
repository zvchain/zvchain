package mysql

import (
	"github.com/zvchain/zvchain/browser/models"
)

func (storage *SlaveStorage) SlaveRewardDatas(address string, typeId uint64, maxHeight uint64) ([]int, []uint64) {
	total := make([]int, 0)
	idPrimarys := make([]uint64, 0)

	for i := 0; i < 30; i++ {
		list := make([]models.RewardHeightAndId, 0)
		storage.db.Model(&models.Reward{}).Where("type = ? and node_id = ? ", typeId, address).Offset(i * 5000).Limit(5000).Select("block_height,id").Scan(&list)
		//defer rows.Close()
		//rows,_:=storage.db.Model(&models.Reward{}).Where("type = ? and node_id = ? ",typeId, address).Offset(i*5000).Limit(5000).Select("block_height,id").Rows()
		s := make([]int, 0, 0)
		p := make([]uint64, 0, 0)
		for _, rewardheight := range list {
			if uint64(rewardheight.BlockHeight) > maxHeight {
				break
			}
			s = append(s, rewardheight.BlockHeight)
			p = append(p, rewardheight.Id)
		}
		total = append(total, s...)
		idPrimarys = append(idPrimarys, p...)
		if len(s) < 5000 {
			break
		}
	}
	return total, idPrimarys
}
