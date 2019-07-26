package log

import (
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strconv"
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
	file *os.File
	maxSize int64
	fileName string
	counter int
}

func newLogFileWriter(fileName string, maxSize int64) *logFileWriter {
	writer := &logFileWriter{
		maxSize: maxSize,
		fileName: fileName,
	}

	err := removeFile(fileName + "_0.log")
	if err != nil {
		return nil
	}
	file, err := os.OpenFile(fileName + "_0.log", os.O_CREATE|os.O_WRONLY, 0755)
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
		file, e := os.OpenFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter), 10) + ".log",
			os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			return e
		}
		p.file = file

		if p.counter >= 2 {
			e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter - 2), 10) + ".log")
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
		file, e := os.OpenFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter), 10) + ".log",
			os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			return n, e
		}
		p.file = file

		if p.counter >= 2 {
			e = removeFile(p.fileName + "_" + strconv.FormatInt(int64(p.counter - 2), 10) + ".log")
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

func (lrs *Logrusplus) StandardLogger() *logrus.Logger {
	return logrus.StandardLogger()
}

func (lrs *Logrusplus) Logger(fileName string, maxSize int64, level logrus.Level) *logrus.Logger {
	var logger *logrus.Logger

	if _logger, ok := lrs.loggers[fileName]; ok {
		logger = _logger
	} else {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})

		fileWriter := newLogFileWriter(fileName, maxSize)
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