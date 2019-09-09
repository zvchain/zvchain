package log

import (
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func Test_Main(t *testing.T) {
	Init()
	p2pLogger := P2PLogger //lrp.Logger("p2p")
	tvmLogger := TVMLogger //lrp.Logger("vm")
	stdLogger := logrus.StandardLogger()
	commonLogger := DefaultLogger //lrp.Logger("common")

	count := 0
	for {
		go func() {
			p2pLogger.WithFields(logrus.Fields{
				"test":  "p2p",
				"count": count,
			}).Info("hello world")
		}()

		go func() {
			tvmLogger.WithFields(logrus.Fields{
				"test":  "vm",
				"count": count,
			}).Info("hello world")
		}()

		go func() {
			stdLogger.WithFields(logrus.Fields{
				"test":  "std",
				"count": count,
			}).Info("hello world")
		}()

		go func() {
			commonLogger.WithFields(logrus.Fields{
				"test":  "common",
				"count": count,
			}).Info("hello world")
		}()

		count++
		if count == 10000 {
			break
		}
		time.Sleep(100) //1 * time.Second)
	}
	clearLogLevel()
}

func clearLogLevel() {
	StdLogger.SetLevel(logrus.PanicLevel)
	CoreLogger.SetLevel(logrus.PanicLevel)
	DefaultLogger.SetLevel(logrus.PanicLevel)
	ConsensusLogger.SetLevel(logrus.PanicLevel)
	ConsensusStdLogger.SetLevel(logrus.PanicLevel)
	GroupLogger.SetLevel(logrus.PanicLevel)

}

func Test_Hook(t *testing.T) {
	Init()

	StdLogger.Info("hello world!")
}
