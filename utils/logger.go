package utils

import (
	"fmt"
)

type LogLevel struct {
	label    string
	priority int
}

const (
	baseFormat string = "%s [%s::%s] "
)

var (
	Debug = LogLevel{"DEBUG", 0}
	Info  = LogLevel{"INFO", 1}
	Warn  = LogLevel{"WARN", 2}
	Error = LogLevel{"ERROR", 3}
	Fatal = LogLevel{"FATAL", 4}
)

type Logger interface {
	GetLogLevel() LogLevel
	SetLogLevel(level LogLevel)
	Log(level LogLevel, format string, a ...any)
	Debug(format string, a ...any)
	Info(format string, a ...any)
	Warn(format string, a ...any)
	Error(format string, a ...any)
	Fatal(format string, a ...any)
}

type LoggerImpl struct {
	Output chan string

	level LogLevel
}

func InitLogger(level LogLevel) LoggerImpl {
	return LoggerImpl{make(chan string, 100), level}
}

func (l *LoggerImpl) GetLogLevel() LogLevel {
	return l.level
}

func (l *LoggerImpl) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *LoggerImpl) log(level LogLevel, format string, a ...any) {
	if level.priority < l.level.priority {
		return
	}
	caller, file, _ := Funcinfo(3)
	base := fmt.Sprintf(baseFormat, level.label, file, caller)
	l.Output <- fmt.Sprintf(base+format, a...)
}

func (l *LoggerImpl) Log(level LogLevel, format string, a ...any) {
	l.log(level, format, a...)
}

func (l *LoggerImpl) Debug(format string, a ...any) {
	l.log(Debug, format, a...)
}

func (l *LoggerImpl) Info(format string, a ...any) {
	l.log(Info, format, a...)
}

func (l *LoggerImpl) Warn(format string, a ...any) {
	l.log(Warn, format, a...)
}

func (l *LoggerImpl) Error(format string, a ...any) {
	l.log(Error, format, a...)
}

func (l *LoggerImpl) Fatal(format string, a ...any) {
	l.log(Fatal, format, a...)
}
