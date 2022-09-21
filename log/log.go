package log

import (
	"fmt"
	"github.com/Jack-ZL/go_rookie/internal/grstrings"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
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

type LoggerLevel int // 日志级别初始化
/**
 * Level
 * @Author：Jack-Z
 * @Description: 不同级别日志的关键字
 * @receiver l
 * @return string
 */
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

type Fields map[string]any

// Logger 日志
type Logger struct {
	Formatter    LoggingFormatter // 格式化
	Level        LoggerLevel      // 级别
	Outs         []*LoggerWriter  // 输出
	LoggerFields Fields           // 额外的信息
	logPath      string           // 日志文件存放目录
	LogFileSize  int64            // 日志文件大小
}

type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

// 定义一个格式化接口（抽离）
type LoggingFormatter interface {
	Format(param *LoggingFormatParam) string
}

// 格式化数据传参定义
type LoggingFormatParam struct {
	Level        LoggerLevel // 日志级别
	IsColor      bool        // 是否显示颜色
	LoggerFields Fields      // 额外的信息
	Msg          any
}

type LoggerFormatter struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
}

/**
 * New
 * @Author：Jack-Z
 * @Description: 实例化一个logger
 * @return *Logger
 */
func New() *Logger {
	return &Logger{}
}

/**
 * Default
 * @Author：Jack-Z
 * @Description: 默认日志输出引擎
 * @return *Logger
 */
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	w := &LoggerWriter{
		Level: LevelDebug,
		Out:   os.Stdout,
	}
	logger.Outs = append(logger.Outs, w)
	logger.Formatter = &TextFormatter{}
	return logger
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
 * Print
 * @Author：Jack-Z
 * @Description: 输出打印操作
 * @receiver l
 * @param level
 * @param msg
 */
func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level {
		// 当前的级别大于输入级别 不打印对应的级别日志
		return
	}
	param := &LoggingFormatParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	str := l.Formatter.Format(param)
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			param.IsColor = true
			str = l.Formatter.Format(param)
			fmt.Fprintln(out.Out, str)
		}
		if out.Level == -1 || level == out.Level {
			fmt.Fprintln(out.Out, str)
			l.CheckFileSize(out)
		}
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
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
	}
}

/**
 * SetLogPath
 * @Author：Jack-Z
 * @Description: 设置日志文件存放路径
 * @receiver l
 * @param logPath
 */
func (l *Logger) SetLogPath(logPath string) {
	l.logPath = logPath
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: -1,
		Out:   FileWriter(path.Join(logPath, "all.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelDebug,
		Out:   FileWriter(path.Join(logPath, "debug.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelInfo,
		Out:   FileWriter(path.Join(logPath, "info.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelError,
		Out:   FileWriter(path.Join(logPath, "error.log")),
	})
}

/**
 * CheckFileSize
 * @Author：Jack-Z
 * @Description: 校验日志文件大小
 * @receiver l
 * @param w
 */
func (l *Logger) CheckFileSize(w *LoggerWriter) {
	// 判断对应的文件大小
	logFile := w.Out.(*os.File)
	if logFile != nil {
		stat, err := logFile.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		size := stat.Size()
		if l.LogFileSize <= 0 {
			l.LogFileSize = 100 << 20
		}
		// 日志文件过大时，进行切片
		if size >= l.LogFileSize {
			_, name := path.Split(stat.Name())
			fileName := name[0:strings.Index(name, ".")]
			writer := FileWriter(path.Join(l.logPath, grstrings.JoinStrings(fileName, ".", time.Now().UnixMilli(), ".log")))
			w.Out = writer
		}
	}

}

/**
 * FileWriter
 * @Author：Jack-Z
 * @Description: 文件写入
 * @param name
 * @return io.Writer
 */
func FileWriter(name string) io.Writer {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return w
}

// func (f *LoggerFormatter) format(msg any) string {
// 	now := time.Now()
// 	if f.IsColor {
// 		// 要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
// 		levelColor := f.LevelColor()
// 		msgColor := f.MsgColor()
// 		return fmt.Sprintf("%s [go_rookie] %s %s%v%s | level= %s %s %s | msg=%s %#v %s | fields=%v ",
// 			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
// 			levelColor, f.Level.Level(), reset, msgColor, msg, reset, f.LoggerFields,
// 		)
// 	}
// 	return fmt.Sprintf("[go_rookie] %v | level=%s | msg=%#v | fields=%#v",
// 		now.Format("2006/01/02 - 15:04:05"),
// 		f.Level.Level(), msg, f.LoggerFields)
// }
//
// func (f *LoggerFormatter) LevelColor() string {
// 	switch f.Level {
// 	case LevelDebug:
// 		return blue
// 	case LevelInfo:
// 		return green
// 	case LevelError:
// 		return red
// 	default:
// 		return cyan
// 	}
// }
//
// func (f *LoggerFormatter) MsgColor() string {
// 	switch f.Level {
// 	case LevelError:
// 		return red
// 	default:
// 		return ""
// 	}
// }
