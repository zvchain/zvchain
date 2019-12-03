// +build !release

package log

import (
	"github.com/sirupsen/logrus"
)

var RusPlus *Logrusplus

var StdLogger = logrus.StandardLogger()

var DefaultLogger = logrus.StandardLogger()
var ConsensusLogger = logrus.StandardLogger()
var ConsensusStdLogger = logrus.StandardLogger()
var CoreLogger = logrus.StandardLogger()
var BlockSyncLogger = logrus.StandardLogger()
var GroupLogger = logrus.StandardLogger()
var MiddlewareLogger = logrus.StandardLogger()
var TxSyncLogger = logrus.StandardLogger()
var P2PLogger = logrus.StandardLogger()
var ForkLogger = logrus.StandardLogger()
var StatisticsLogger = logrus.StandardLogger()
var TVMLogger = logrus.StandardLogger()
var PerformLogger = logrus.StandardLogger()
var ELKLogger = logrus.StandardLogger()
var MeterLogger = logrus.StandardLogger()

const (
	MaxFileSize     = 1024 * 1024 * 200
	DefaultMaxFiles = 2
	CoreMaxFiles    = 12 //core log should keep 12 files to ensure 3 day's logs
	Level           = logrus.DebugLevel
)

func InitElk(logsDir string) {
	ELKLogger = RusPlus.Logger(logsDir+"ELK", MaxFileSize, DefaultMaxFiles, Level)
}
