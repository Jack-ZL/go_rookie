package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

type LoggerLevel int // 日志级别初始化

func (l LoggerLevel) Level() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

// 日志级别，从上到下级别递增
const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

// 日志文字打印时的颜色
const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

type Fields map[string]any
type Logger struct {
	Formatter    LoggerFormatter // 格式化
	Level        LoggerLevel     // 级别
	Outs         []io.Writer     // 输入
	LoggerFields Fields          // 额外的信息
}

type LoggerFormatter struct {
	Level          LoggerLevel // 日志级别
	IsDisplayColor bool        // 是否显示颜色
	LoggerFields   Fields      // 额外的信息
}

func New() *Logger {
	return &Logger{}
}

/**
 * Default
 * @Author：Jack-Z
 * @Description: 默认输出
 * @return *Logger
 */
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	logger.Outs = append(logger.Outs, os.Stdout)
	logger.Formatter = LoggerFormatter{}
	return logger
}

/**
 * Print
 * @Author：Jack-Z
 * @Description: 打印操作
 * @receiver l
 * @param level
 * @param msg
 */
func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level {
		// 日志当前级别 大于 输入级别， 不打印日志
		return
	}
	l.Formatter.Level = level
	l.Formatter.LoggerFields = l.LoggerFields
	logStr := l.Formatter.format(msg)
	for _, out := range l.Outs {
		if out == os.Stdout {
			l.Formatter.IsDisplayColor = true
			logStr = l.Formatter.format(msg)
		}
		fmt.Fprintln(out, logStr)
	}
}

/**
 * WithFields
 * @Author：Jack-Z
 * @Description: 额外的信息赋值
 * @receiver l
 * @param fields
 * @return *Logger
 */
func (l *Logger) WithFields(fields Fields) *Logger {
	l.LoggerFields = fields
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
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

/**
 * format
 * @Author：Jack-Z
 * @Description: 格式化日志输出（颜色、级别等）
 * @receiver f
 * @param msg
 * @return string
 */
func (f *LoggerFormatter) format(msg any) string {
	// 要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
	now := time.Now()
	if f.IsDisplayColor {
		// 要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
		levelColor := f.LevelColor()
		msgColor := f.MsgColor()
		return fmt.Sprintf("%s [go_rookie] %s %s%v%s | level =%s %s %s | msg =%s %#v %s | fields = %v",
			yellow,
			reset,
			blue,
			now.Format("2006/01/02 - 15:04:05"),
			reset,
			levelColor,
			f.Level.Level(),
			reset,
			msgColor,
			msg,
			reset,
			f.LoggerFields,
		)
	}
	return fmt.Sprintf("[go_rookie] %v | level =%s | msg =%#v | fields =%v",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(),
		msg,
		f.LoggerFields,
	)
}

/**
 * LevelColor
 * @Author：Jack-Z
 * @Description: 不同级别的不同颜色
 * @receiver f
 * @return string
 */
func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

/**
 * MsgColor
 * @Author：Jack-Z
 * @Description: 日志文字的颜色：除了error级别为红色文字，其他默认
 * @receiver f
 * @return string
 */
func (f *LoggerFormatter) MsgColor() string {
	switch f.Level {
	case LevelError:
		return red
	default:
		return ""
	}
}
