package browserlog

import (
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
)

var (
	logsDir    = "./logs/"
	BrowserLog *logrus.Logger
	OrmLog     *logrus.Logger
)

func InitLog() {
	BrowserLog = log.RusPlus.Logger(logsDir+"browser", log.MaxFileSize, log.DefaultMaxFiles, log.Level)
	OrmLog = log.RusPlus.Logger(logsDir+"orm", log.MaxFileSize, log.DefaultMaxFiles, log.Level)
}
