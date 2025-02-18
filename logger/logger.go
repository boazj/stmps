// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package logger

import (
	"fmt"
	"runtime"
	"strings"
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

func Init(level LogLevel) *LoggerImpl {
	return &LoggerImpl{make(chan string, 100), level}
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
	caller, file, _ := funcinfo(3)
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

func funcinfo(skip int) (name, file string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if ok {
		name = runtime.FuncForPC(pc).Name()
		i := strings.Index(file, "stmps")
		if i > -1 {
			file = file[i:]
		}
		i = strings.Index(name, "stmps")
		if i > -1 {
			name = name[i:]
		}
	}
	return name, file, line
}

func funcname(skip int) string {
	name, _, _ := funcinfo(skip + 1)
	return name
}
