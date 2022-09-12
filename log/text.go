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
 * @Description: 日志输出内容文本格式化
 * @receiver f
 * @param params
 * @return string
 */
func (f *TextFormatter) Format(params *LoggingFormatterParams) string {
	now := time.Now()

	fieldsStr := ""
	if params.LoggerFields != nil {
		// 额外信息的字符串化处理
		var sb strings.Builder
		var count = 0
		var lens = len(params.LoggerFields)
		for k, v := range params.LoggerFields {
			fmt.Fprintf(&sb, "%s=%v", k, v)
			if count < lens-1 {
				fmt.Fprintf(&sb, ",")
				count++
			}
		}
		fieldsStr = sb.String()
	}

	if params.IsDisplayColor {
		levelColor := f.LevelColor(params.Level)
		msgColor := f.MsgColor(params.Level)
		return fmt.Sprintf("%s [go_rookie] %s %s%v%s | level =%s %s %s | msg =%s %#v %s %s",
			yellow,
			reset,
			blue,
			now.Format("2006/01/02 - 15:04:05"),
			reset,
			levelColor,
			params.Level.Level(),
			reset,
			msgColor,
			params.Msg,
			reset,
			fieldsStr,
		)
	}
	return fmt.Sprintf("[go_rookie] %v | level =%s | msg =%#v %s",
		now.Format("2006/01/02 - 15:04:05"),
		params.Level.Level(),
		params.Msg,
		fieldsStr,
	)
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
