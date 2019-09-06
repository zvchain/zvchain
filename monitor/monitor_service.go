//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package monitor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gohouse/gorose"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/log"
)

type MonitorService struct {
	enable   bool
	cfg      *gorose.DbConfigSingle
	queue    []*LogEntry
	lastSend time.Time
	nodeID   string
	status   int32
	mu       sync.Mutex

	resStat    *NodeResStat
	nodeInfo   *NodeInfo
	lastUpdate time.Time

	internalNodeIds map[string]bool
}

const (
	LogTypeProposal                 = 1
	LogTypeBlockBroadcast           = 2
	LogTypeRewardBroadcast          = 3
	LogTypeCreateGroup              = 4
	LogTypeCreateGroupSignTimeout   = 5
	LogTypeInitGroupRevPieceTimeout = 6
	LogTypeGroupRecoverFromResponse = 7
)

const TableName = "logs"

type LogEntry struct {
	LogType  int
	Operator string
	OperTime time.Time
	Height   uint64
	Hash     string
	PreHash  string
	Proposer string
	Verifier string
	Ext      string
}

func (le *LogEntry) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["LogType"] = le.LogType
	m["Operator"] = le.Operator
	m["OperTime"] = le.OperTime
	m["Height"] = le.Height
	m["Hash"] = le.Hash
	m["PreHash"] = le.PreHash
	m["Proposer"] = le.Proposer
	m["Verifier"] = le.Verifier
	m["Ext"] = le.Ext
	return m
}

var Instance = &MonitorService{}

func InitLogService(nodeID string) {
	Instance = &MonitorService{
		nodeID:   nodeID,
		queue:    make([]*LogEntry, 0),
		lastSend: time.Now(),
		enable:   true,
		resStat:  initNodeResStat(),
	}
	rHost := common.GlobalConf.GetString("gzv", "log_db_host", "")
	rPort := common.GlobalConf.GetInt("gzv", "log_db_port", 0)
	rDB := common.GlobalConf.GetString("gzv", "log_db_db", "")
	rUser := common.GlobalConf.GetString("gzv", "log_db_user", "")
	rPass := common.GlobalConf.GetString("gzv", "log_db_password", "")
	Instance.cfg = &gorose.DbConfigSingle{
		Driver:          "mysql",                                                                                         // Drive: mysql/sqlite/oracle/mssql/postgres
		EnableQueryLog:  false,                                                                                           // Whether to open SQL log
		SetMaxOpenConns: 0,                                                                                               // (Connection pool) the maximum number of open connections, the default value of 0 means no limit
		SetMaxIdleConns: 0,                                                                                               // (Connection pool) number of idle connections
		Prefix:          "",                                                                                              // Prefix
		Dsn:             fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8&parseTime=true", rUser, rPass, rHost, rPort, rDB), // Database link -> username:password@protocol(address)/dbname?param=value
	}

	Instance.insertMinerID()

	Instance.loadInternalNodesIds()
}

func (ms *MonitorService) saveLogs(logs []*LogEntry) {
	var err error
	defer func() {
		if err != nil {
			log.DefaultLogger.Errorf("save logs fail, err=%v, size %v", err, len(logs))
		} else {
			log.DefaultLogger.Infof("save logs success, size %v", len(logs))
		}
		ms.lastSend = time.Now()
		atomic.StoreInt32(&ms.status, 0)
	}()
	if !atomic.CompareAndSwapInt32(&ms.status, 0, 1) {
		return
	}

	connection, err := gorose.Open(ms.cfg)
	if err != nil {
		return
	}
	if connection == nil {
		err = fmt.Errorf("nil connection")
		return
	}
	defer connection.Close()

	sess := connection.NewSession()

	dm := make([]map[string]interface{}, 0)
	for _, log := range logs {
		dm = append(dm, log.toMap())
	}
	_, err = sess.Table(TableName).Data(dm).Insert()
}

func (ms *MonitorService) doAddLog(log *LogEntry) {
	if !ms.MonitorEnable() {
		return
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.queue = append(ms.queue, log)
	if len(ms.queue) >= 5 || time.Since(ms.lastSend).Seconds() > 15 {
		go ms.saveLogs(ms.queue)
		ms.queue = make([]*LogEntry, 0)
	}
}

func (ms *MonitorService) AddLog(logEntry *LogEntry) {
	logEntry.Operator = ms.nodeID
	logEntry.OperTime = time.Now()
	ms.doAddLog(logEntry)
}

func (ms *MonitorService) MonitorEnable() bool {
	return ms.enable && ms.cfg != nil && ms.cfg.Dsn != ""
}

func (ms *MonitorService) loadInternalNodesIds() {
	connection, err := gorose.Open(ms.cfg)
	if err != nil {
		return
	}
	if connection == nil {
		err = fmt.Errorf("nil connection")
		return
	}
	defer connection.Close()

	sess := connection.NewSession()
	ret, err := sess.Table("nodes").Fields("MinerId").Limit(1000).Get()
	m := make(map[string]bool)

	ids := make([]string, 0)
	if ret != nil {
		for _, d := range ret {
			id := d["MinerId"].(string)
			m[id] = true
			ids = append(ids, id)
		}
	}
	ms.internalNodeIds = m

	log.StdLogger.Info("load internal nodes ", ids)
}

func (ms *MonitorService) AddLogIfNotInternalNodes(logEntry *LogEntry) {
	if _, ok := ms.internalNodeIds[logEntry.Proposer]; !ok {
		ms.AddLog(logEntry)
		log.DefaultLogger.Infof("addlog of not internal nodes %v", logEntry.Proposer)
	}
}

func (ms *MonitorService) IsFirstNInternalNodesInGroup(mems []groupsig.ID, n int) bool {
	cnt := 0
	for _, mem := range mems {
		if _, ok := ms.internalNodeIds[mem.GetAddrString()]; ok {
			cnt++
			if cnt >= n {
				break
			}
			if mem.GetAddrString() == ms.nodeID {
				return true
			}
		}
	}
	return false
}

func (ms *MonitorService) UpdateNodeInfo(ni *NodeInfo) {
	if !ms.MonitorEnable() {
		return
	}

	ms.nodeInfo = ni
	if time.Since(ms.lastUpdate).Seconds() > 2 {
		ms.lastUpdate = time.Now()
		connection, err := gorose.Open(ms.cfg)
		if err != nil {
			return
		}
		if connection == nil {
			err = fmt.Errorf("nil connection")
			return
		}
		defer connection.Close()

		sess := connection.NewSession()
		dm := make(map[string]interface{})
		dm["MinerId"] = ms.nodeID
		dm["NType"] = ms.nodeInfo.Type
		dm["VrfThreshold"] = ms.nodeInfo.VrfThreshold
		dm["PStake"] = ms.nodeInfo.PStake
		dm["BlockHeight"] = ms.nodeInfo.BlockHeight
		dm["GroupHeight"] = ms.nodeInfo.GroupHeight
		dm["TxPoolCount"] = ms.nodeInfo.TxPoolCount
		dm["CPU"] = ms.resStat.CPU
		dm["Mem"] = ms.resStat.Mem
		dm["RcvBps"] = ms.resStat.RcvBps
		dm["TxBps"] = ms.resStat.TxBps
		dm["UpdateTime"] = time.Now().UTC()
		dm["Instance"] = common.InstanceIndex

		affet, err := sess.Table("nodes").Where(fmt.Sprintf("MinerId='%v'", ms.nodeID)).Data(dm).Update()
		if err == nil {
			if affet <= 0 {
				sess.Table("nodes").Data(dm).Insert()
			}
		} else {
			log.ConsensusStdLogger.Errorf("update node info error:%v", err)
		}
	}
}

func (ms *MonitorService) insertMinerID() {
	if !ms.MonitorEnable() {
		return
	}

	connection, err := gorose.Open(ms.cfg)
	if err != nil {
		return
	}
	if connection == nil {
		err = fmt.Errorf("nil connection")
		return
	}
	defer connection.Close()

	sess := connection.NewSession()
	dm := make(map[string]interface{})
	dm["MinerId"] = ms.nodeID
	dm["UpdateTime"] = time.Now()
	dm["Instance"] = common.InstanceIndex

	affet, err := sess.Table("nodes").Data(dm).Insert()
	if err == nil {
		if affet <= 0 {
			sess.Table("nodes").Data(dm).Insert()
		}
	} else {
		fmt.Printf("insert nodes fail, sql=%v, err=%v\n", sess.LastSql, err)
	}
}
