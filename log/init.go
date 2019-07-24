package log

import (
	"os"
)

func init()  {
	logrusplus = New()
	StdLogger = logrusplus.StandardLogger()

	logsDir := "./logs2/"

	_, err := os.Stat(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(logsDir, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	DefaultLogger = logrusplus.Logger(logsDir + "default")
	ConsensusLogger = logrusplus.Logger(logsDir + "consensus")
	ConsensusStdLogger = logrusplus.Logger(logsDir + "consensus_std")
	CoreLogger = logrusplus.Logger(logsDir + "core")
	BlockSyncLogger = logrusplus.Logger(logsDir + "block_sync")
	GroupLogger = logrusplus.Logger(logsDir + "group")
	MiddlewareLogger = logrusplus.Logger(logsDir + "middleware")
	TxSyncLogger = logrusplus.Logger(logsDir + "tx_sync")
	P2PLogger = logrusplus.Logger(logsDir + "p2p")
	ForkLogger = logrusplus.Logger(logsDir + "fork")
	VRFLogger = logrusplus.Logger(logsDir + "vrf")
	StatisticsLogger = logrusplus.Logger(logsDir + "statistics")
	TVMLogger = logrusplus.Logger(logsDir + "tvm")
	PerformLogger = logrusplus.Logger(logsDir + "perform")
}