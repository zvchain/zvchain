// +build !release

package log

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
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
var WorkingDirName = getCurrentDirectory()

const (
	MaxFileSize = 1024 * 1024 * 200
	Level       = logrus.DebugLevel
)

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}
	absPath := strings.Replace(dir, "\\", "/", -1)
	s := strings.Split(absPath, "/")
	if len(s) > 0 {
		return s[len(s)-1]
	}
	return ""
}
