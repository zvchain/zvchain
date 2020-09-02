package log

import (
	"errors"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func removeFile(fileName string) error {
	_, e := os.Stat(fileName)
	if e == nil || os.IsExist(e) {
		e = os.Remove(fileName)
		return e
	} else {
		return nil
	}
}

type logFileWriter struct {
	file     *os.File
	maxSize  int64
	maxFiles int
	fileName string
	counter  int
}

func newLogFileWriter(fileName string, maxSize int64, maxFiles int) *logFileWriter {
	writer := &logFileWriter{
		maxSize:  maxSize,
		maxFiles: maxFiles,
		fileName: fileName,
	}

	err := removeFile(fileName + "_0.log")
	if err != nil {
		return nil
	}
	file, err := os.OpenFile(fileName+"_0.log", os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return nil
	}
	writer.file = file

	return writer
}

func (p *logFileWriter) Fire(entry *logrus.Entry) error {
	if p == nil {
		return errors.New("logFileWriter is nil")
	}
	if p.file == nil {
		return errors.New("file not opened")
	}
	s, e := entry.String()
	if e != nil {
		return e
	}
	_, e = p.file.WriteString(s)
	if e != nil {
		return e
	}
	fileSize, e := p.file.Seek(0, io.SeekEnd)
	if e != nil {
		return e
	}

	if fileSize > p.maxSize {
		e = p.file.Close()
		if e != nil {
			return e
		}

		p.counter++
		e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter), 10) + ".log")
		if e != nil {
			return e
		}
		file, e := os.OpenFile(p.fileName+"_"+strconv.FormatInt(int64(p.counter), 10)+".log",
			os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			return e
		}
		p.file = file

		if p.counter >= p.maxFiles {
			e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter-p.maxFiles), 10) + ".log")
			if e != nil {
				return e
			}
		}
	}
	return e
}

func (p *logFileWriter) Write(data []byte) (n int, e error) {
	if p == nil {
		return 0, errors.New("logFileWriter is nil")
	}
	if p.file == nil {
		return 0, errors.New("file not opened")
	}
	n, e = p.file.Write(data)
	if e != nil {
		return n, e
	}
	fileSize, e := p.file.Seek(0, io.SeekEnd)
	if e != nil {
		return n, e
	}

	if fileSize > p.maxSize {
		e = p.file.Close()
		if e != nil {
			return n, e
		}

		p.counter++
		e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter), 10) + ".log")
		if e != nil {
			return n, e
		}
		file, e := os.OpenFile(p.fileName+"_"+strconv.FormatInt(int64(p.counter), 10)+".log",
			os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			return n, e
		}
		p.file = file

		if p.counter >= p.maxFiles {
			e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter-p.maxFiles), 10) + ".log")
			if e != nil {
				return n, e
			}
		}
	}
	return n, e
}

func (*logFileWriter) Levels() []logrus.Level {
	return logrus.AllLevels
}

type Logrusplus struct {
	loggers map[string]*logrus.Logger
}

func New() *Logrusplus {
	return &Logrusplus{
		loggers: make(map[string]*logrus.Logger),
	}
}

func (lrs *Logrusplus) Logger(fileName string, maxSize int64, maxFiles int, level logrus.Level) *logrus.Logger {
	var logger *logrus.Logger

	if _logger, ok := lrs.loggers[fileName]; ok {
		logger = _logger
	} else {
		logger = logrus.New()
		formatter := new(logrus.JSONFormatter)
		formatter.TimestampFormat = time.RFC3339Nano
		logger.Formatter = formatter

		fileWriter := newLogFileWriter(fileName, maxSize, maxFiles)
		if fileWriter != nil {
			logger.SetOutput(fileWriter)
			//logger.AddHook(fileWriter)
		} else {
			logger.Info("Failed to log to file, using default stderr")
		}
		logger.SetLevel(level)
		lrs.loggers[fileName] = logger
	}

	return logger
}
