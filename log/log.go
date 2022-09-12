package log

import (
	"fmt"
	"io"
	"os"
)

type LoggerLevel int

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

type Logger struct {
	Formatter LoggerFormatter
	Level     LoggerLevel
	Outs      []io.Writer
}

type LoggerFormatter struct {
}

func New() *Logger {
	return &Logger{}
}

func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	logger.Outs = append(logger.Outs, os.Stdout)
	logger.Formatter = LoggerFormatter{}
	return logger
}

func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level {
		// 日志当前级别 大于 输入级别， 不打印日志
		return
	}
	for _, out := range l.Outs {
		fmt.Fprintln(out, msg)
	}
}

func (l *Logger) Info(msg any) {
	l.Print(LevelInfo, msg)
}

func (l *Logger) Debug(msg any) {
	l.Print(LevelDebug, msg)

}

func (l *Logger) Error(msg any) {
	l.Print(LevelError, msg)

}
