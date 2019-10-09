package log

import "os"

func Init() {
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

	DefaultLogger = RusPlus.Logger(logsDir+"default", MaxFileSize, MaxFiles, Level)
	ConsensusLogger = RusPlus.Logger(logsDir+"consensus", MaxFileSize, MaxFiles, Level)
	ConsensusStdLogger = RusPlus.Logger(logsDir+"consensus_std", MaxFileSize, MaxFiles, Level)
	//core log should keep 12 files to ensure 3 day's logs
	CoreLogger = RusPlus.Logger(logsDir+"core", MaxFileSize, MaxFiles*6, Level)
	BlockSyncLogger = RusPlus.Logger(logsDir+"block_sync", MaxFileSize, MaxFiles, Level)
	GroupLogger = RusPlus.Logger(logsDir+"group", MaxFileSize, MaxFiles, Level)
	MiddlewareLogger = RusPlus.Logger(logsDir+"middleware", MaxFileSize, MaxFiles, Level)
	TxSyncLogger = RusPlus.Logger(logsDir+"tx_sync", MaxFileSize, MaxFiles, Level)
	P2PLogger = RusPlus.Logger(logsDir+"p2p", MaxFileSize, MaxFiles, Level)
	ForkLogger = RusPlus.Logger(logsDir+"fork", MaxFileSize, MaxFiles, Level)
	StatisticsLogger = RusPlus.Logger(logsDir+"statistics", MaxFileSize, MaxFiles, Level)
	TVMLogger = RusPlus.Logger(logsDir+"tvm", MaxFileSize, MaxFiles, Level)
	PerformLogger = RusPlus.Logger(logsDir+"perform", MaxFileSize, MaxFiles, Level)
	ELKLogger = RusPlus.Logger(logsDir+"ELK", MaxFileSize, MaxFiles, Level)
}
