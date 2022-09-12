package go_rookie

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

// 输出
var DefaultWriter io.Writer = os.Stdout

type LoggingConfig struct {
	Formatrer LoggerFormatter
	out       io.Writer
}

type LoggerFormatter = func(params *LogFormatterParams) string

// 日志输出内容项预定义
type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIP       net.IP
	Method         string
	Path           string
	IsDisplayColor bool
}

func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

func (p *LogFormatterParams) ResetColor() string {
	return reset
}

var defaultFormatter = func(params *LogFormatterParams) string {
	statusCodeColor := params.StatusCodeColor()
	resetColor := params.ResetColor() // 结束符颜色
	// 如果时间差超过1分钟，仍然以秒显示
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}

	if params.IsDisplayColor {
		return fmt.Sprintf("%s [go_rookie] %s |%s %v %s| %s %3d %s |%s %13v %s| %15s  |%s %-7s %s %s %#v %s \n",
			yellow,
			resetColor,
			blue,
			params.TimeStamp.Format("2006/01/02 - 15:04:05"),
			resetColor,
			statusCodeColor,
			params.StatusCode,
			resetColor,
			red,
			params.Latency,
			resetColor,
			params.ClientIP,
			magenta,
			params.Method,
			resetColor,
			cyan,
			params.Path,
			resetColor,
		)
	}
	return fmt.Sprintf("[go_rookie] %v | %3d | %13v | %15s |%-7s %#v \n",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency,
		params.ClientIP,
		params.Method,
		params.Path,
	)
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatrer
	if formatter == nil {
		formatter = defaultFormatter
	}

	out := conf.out
	displayColor := false
	if out == nil {
		out = DefaultWriter
		displayColor = true
	}
	return func(ctx *Context) {
		start := time.Now()       // 开始时间
		path := ctx.R.URL.Path    // 请求路径
		raw := ctx.R.URL.RawQuery // 参数

		next(ctx) // 执行业务

		stop := time.Now()         // 截止时间
		latency := stop.Sub(start) // 时间差
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)  // ip地址
		method := ctx.R.Method       // 请求方式
		statusCode := ctx.StatusCode // 状态码
		if raw != "" {
			path = path + "?" + raw
		}

		param := &LogFormatterParams{
			Request:        ctx.R,
			IsDisplayColor: displayColor,
		}

		param.ClientIP = clientIP
		param.Latency = latency
		param.TimeStamp = stop
		param.Path = path
		param.Method = method
		param.StatusCode = statusCode
		fmt.Fprint(out, formatter(param))
	}
}

/**
 * Logging
 * @Author：Jack-Z
 * @Description: 对外暴露的调用函数
 * @param next
 * @return HandlerFunc
 */
func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
