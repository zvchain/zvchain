package log

import (
	"os"
	"sync"
)

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

	DefaultLogger = RusPlus.Logger(logsDir+"default", MaxFileSize, DefaultMaxFiles, Level)
	ConsensusLogger = RusPlus.Logger(logsDir+"consensus", MaxFileSize, DefaultMaxFiles, Level)
	ConsensusStdLogger = RusPlus.Logger(logsDir+"consensus_std", MaxFileSize, DefaultMaxFiles, Level)
	CoreLogger = RusPlus.Logger(logsDir+"core", MaxFileSize, CoreMaxFiles, Level)
	BlockSyncLogger = RusPlus.Logger(logsDir+"block_sync", MaxFileSize, DefaultMaxFiles, Level)
	GroupLogger = RusPlus.Logger(logsDir+"group", MaxFileSize, DefaultMaxFiles, Level)
	MiddlewareLogger = RusPlus.Logger(logsDir+"middleware", MaxFileSize, DefaultMaxFiles, Level)
	TxSyncLogger = RusPlus.Logger(logsDir+"tx_sync", MaxFileSize, DefaultMaxFiles, Level)
	P2PLogger = RusPlus.Logger(logsDir+"p2p", MaxFileSize, DefaultMaxFiles, Level)
	ForkLogger = RusPlus.Logger(logsDir+"fork", MaxFileSize, DefaultMaxFiles, Level)
	StatisticsLogger = RusPlus.Logger(logsDir+"statistics", MaxFileSize, DefaultMaxFiles, Level)
	TVMLogger = RusPlus.Logger(logsDir+"tvm", MaxFileSize, DefaultMaxFiles, Level)
	PerformLogger = RusPlus.Logger(logsDir+"perform", MaxFileSize, DefaultMaxFiles, Level)
	MeterLogger = RusPlus.Logger(logsDir+"meter", MaxFileSize, DefaultMaxFiles, Level)
	Recorder = TimeRecorder{m: sync.Map{}}
	InitElk(logsDir)
}
