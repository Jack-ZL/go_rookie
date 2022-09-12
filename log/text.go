package log

import (
	"fmt"
	"strings"
	"time"
)

type TextFormatter struct {
}

/**
 * Format
 * @Author：Jack-Z
 * @Description: 格式化为文本
 * @receiver f
 * @param param
 * @return string
 */
func (f *TextFormatter) Format(param *LoggingFormatParam) string {
	now := time.Now()
	fieldsString := ""
	if param.LoggerFields != nil {
		// name=xx,age=xxx
		var sb strings.Builder
		var count = 0
		var lens = len(param.LoggerFields)
		for k, v := range param.LoggerFields {
			fmt.Fprintf(&sb, "%s=%v", k, v)
			if count < lens-1 {
				fmt.Fprintf(&sb, ",")
				count++
			}
		}
		fieldsString = sb.String()
	}
	var msgInfo = "\n msg: "
	if param.Level == LevelError {
		msgInfo = "\n Error Cause By: "
	}
	if param.IsColor {
		levelColor := f.LevelColor(param.Level)
		msgColor := f.MsgColor(param.Level)
		return fmt.Sprintf("%s [go_rookie] %s %s%v%s | level= %s %s %s%s%s %v %s %s ",
			yellow,
			reset,
			blue,
			now.Format("2006/01/02 - 15:04:05"),
			reset,
			levelColor,
			param.Level.Level(),
			reset,
			msgColor,
			msgInfo,
			param.Msg,
			reset,
			fieldsString,
		)
	}
	return fmt.Sprintf("[go_rookie] %v | level=%s%s%v %s",
		now.Format("2006/01/02 - 15:04:05"),
		param.Level.Level(),
		msgInfo,
		param.Msg,
		fieldsString)
}

/**
 * LevelColor
 * @Author：Jack-Z
 * @Description: 不同级别的不同颜色
 * @receiver f
 * @param level
 * @return string
 */
func (f *TextFormatter) LevelColor(level LoggerLevel) string {
	switch level {
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
 * @param level
 * @return string
 */
func (f *TextFormatter) MsgColor(level LoggerLevel) string {
	switch level {
	case LevelError:
		return red
	default:
		return ""
	}
}
