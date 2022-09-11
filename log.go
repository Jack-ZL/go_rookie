package go_rookie

import (
	"log"
	"net"
	"strings"
	"time"
)

type LoggingConfig struct {
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
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

		// 日志输出
		log.Printf("[msgo] %v | %3d | %13v | %15s |%-7s %#v",
			stop.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
