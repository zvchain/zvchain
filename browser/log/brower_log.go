package browserlog

import (
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
)

var (
	logsDir    = "./logs/"
	BrowserLog *logrus.Logger
)

func InitLog() {
	BrowserLog = log.RusPlus.Logger(logsDir+"browser", log.MaxFileSize, 2, log.Level)
}
