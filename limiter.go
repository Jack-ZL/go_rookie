package go_rookie

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

/**
 * Limiter
 * @Author：Jack-Z
 * @Description: 限流中间件
 * @param limit
 * @param cap
 * @return MiddlewareFunc
 */
func Limiter(limit, cap int) MiddlewareFunc {
	li := rate.NewLimiter(rate.Limit(limit), cap) // 创建一个限流器，限制每秒的请求数为limit，桶的容量为cap
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			con, cancelFunc := context.WithTimeout(context.Background(), time.Duration((1)*time.Second)) // 设置上下文的超时时间为1秒
			defer cancelFunc()                                                                           // 确保在函数结束时取消上下文

			err := li.WaitN(con, 1) // 等待令牌，如果在上下文超时之前获取到令牌，则继续执行，否则返回错误
			if err != nil {
				err := ctx.String(http.StatusForbidden, "限流了")
				if err != nil {
					ctx.Logger.Error(err)
					return
				}
				return
			}
			next(ctx)
		}
	}
}
