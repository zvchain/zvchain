package log

import (
	"net"
	"os"

	"github.com/bshuster-repo/logrus-logstash-hook"
)

func Init() {

	RusPlus = New(nil)
	StdLogger.SetLevel(Level)
	logsDir := "./logs/"

	_, err := os.Stat(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(logsDir, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	DefaultLogger = RusPlus.Logger(logsDir+"default", MaxFileSize, Level)
	ConsensusLogger = RusPlus.Logger(logsDir+"consensus", MaxFileSize, Level)
	ConsensusStdLogger = RusPlus.Logger(logsDir+"consensus_std", MaxFileSize, Level)
	CoreLogger = RusPlus.Logger(logsDir+"core", MaxFileSize, Level)
	BlockSyncLogger = RusPlus.Logger(logsDir+"block_sync", MaxFileSize, Level)
	GroupLogger = RusPlus.Logger(logsDir+"group", MaxFileSize, Level)
	MiddlewareLogger = RusPlus.Logger(logsDir+"middleware", MaxFileSize, Level)
	TxSyncLogger = RusPlus.Logger(logsDir+"tx_sync", MaxFileSize, Level)
	P2PLogger = RusPlus.Logger(logsDir+"p2p", MaxFileSize, Level)
	ForkLogger = RusPlus.Logger(logsDir+"fork", MaxFileSize, Level)
	StatisticsLogger = RusPlus.Logger(logsDir+"statistics", MaxFileSize, Level)
	TVMLogger = RusPlus.Logger(logsDir+"tvm", MaxFileSize, Level)
	PerformLogger = RusPlus.Logger(logsDir+"perform", MaxFileSize, Level)
	ELKLogger = RusPlus.Logger(logsDir+"ELK", MaxFileSize, Level)
	//ELKLogger.Hooks.Add(logstash().(logrus.Hook))
}

func logstash() interface{} {
	conn, err := net.Dial("tcp", "47.252.87.139:5566")
	if err != nil {
		panic("")
	}

	hook, err := logrustash.NewHookWithConn(conn, "gzv")
	if err != nil {
		panic("")
	}

	return hook
}
