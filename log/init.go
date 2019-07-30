package log

import "os"

func Init()  {
	RusPlus = New()
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

	DefaultLogger = RusPlus.Logger(logsDir + "default", MaxFileSize, Level)
	ConsensusLogger = RusPlus.Logger(logsDir + "consensus", MaxFileSize, Level)
	ConsensusStdLogger = RusPlus.Logger(logsDir + "consensus_std", MaxFileSize, Level)
	CoreLogger = RusPlus.Logger(logsDir + "core", MaxFileSize, Level)
	BlockSyncLogger = RusPlus.Logger(logsDir + "block_sync", MaxFileSize, Level)
	GroupLogger = RusPlus.Logger(logsDir + "group", MaxFileSize, Level)
	MiddlewareLogger = RusPlus.Logger(logsDir + "middleware", MaxFileSize, Level)
	TxSyncLogger = RusPlus.Logger(logsDir + "tx_sync", MaxFileSize, Level)
	P2PLogger = RusPlus.Logger(logsDir + "p2p", MaxFileSize, Level)
	ForkLogger = RusPlus.Logger(logsDir + "fork", MaxFileSize, Level)
	VRFLogger = RusPlus.Logger(logsDir + "vrf", MaxFileSize, Level)
	StatisticsLogger = RusPlus.Logger(logsDir + "statistics", MaxFileSize, Level)
	TVMLogger = RusPlus.Logger(logsDir + "tvm", MaxFileSize, Level)
	PerformLogger = RusPlus.Logger(logsDir + "perform", MaxFileSize, Level)
}