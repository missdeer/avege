package common

import (
	"fmt"
	"github.com/op/go-logging"
	"log"
	"os"
)

type DebugLog int

const (
	PANIC = iota
	FATAL
	ERROR
	WARNING
	INFO
	DEBUG
)

var (
	DebugLevel DebugLog = INFO
	logger              = logging.MustGetLogger("avege")
	format              = logging.MustStringFormatter(
		"%{color}%{time:15:04:05.000} %{level:.1s} ▶%{color:reset} %{message}",
	)
)

type Password string

func (p Password) Redacted() interface{} {
	return logging.Redact(string(p))
}

func init() {
	backend := logging.NewLogBackend(os.Stdout, "", 0)

	backendFormatter := logging.NewBackendFormatter(backend, format)

	// Set the backends to be used.
	logging.SetBackend(backendFormatter)
}

func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	if DebugLevel >= FATAL {
		logger.Fatalf(format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if DebugLevel >= ERROR {
		logger.Errorf(format, args...)
	}
}

func Warningf(format string, args ...interface{}) {
	if DebugLevel >= WARNING {
		logger.Warningf(format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if DebugLevel >= INFO {
		logger.Infof(format, args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if DebugLevel >= DEBUG {
		logger.Debugf(format, args...)
	}
}

func Panic(args ...interface{}) {
	log.Panicln(args...)
}

func Fatal(args ...interface{}) {
	if DebugLevel >= FATAL {
		logger.Fatal(fmt.Sprintln(args...))
	}
}

func Error(args ...interface{}) {
	if DebugLevel >= ERROR {
		logger.Error(fmt.Sprintln(args...))
	}
}

func Warning(args ...interface{}) {
	if DebugLevel >= WARNING {
		logger.Warning(fmt.Sprintln(args...))
	}
}

func Info(args ...interface{}) {
	if DebugLevel >= INFO {
		logger.Info(fmt.Sprintln(args...))
	}
}

func Debug(args ...interface{}) {
	if DebugLevel >= DEBUG {
		logger.Debug(fmt.Sprintln(args...))
	}
}
