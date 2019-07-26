// +build !release

package log

import (
	"github.com/sirupsen/logrus"
)

var RusPlus *Logrusplus

var StdLogger *logrus.Logger

var DefaultLogger *logrus.Logger
var ConsensusLogger *logrus.Logger
var ConsensusStdLogger *logrus.Logger
var CoreLogger *logrus.Logger
var BlockSyncLogger *logrus.Logger
var GroupLogger *logrus.Logger
var MiddlewareLogger *logrus.Logger
var TxSyncLogger *logrus.Logger
var P2PLogger *logrus.Logger
var ForkLogger *logrus.Logger
var VRFLogger *logrus.Logger
var StatisticsLogger *logrus.Logger
var TVMLogger *logrus.Logger
var PerformLogger *logrus.Logger

const (
	MaxFileSize = 1024 * 1024 * 20
	Level = logrus.DebugLevel
)
