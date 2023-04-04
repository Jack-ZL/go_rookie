package go_rookie

import (
	"context"
	"golang.org/x/time/rate"
	"net/http"
	"time"
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
	li := rate.NewLimiter(rate.Limit(limit), cap)
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			con, cancelFunc := context.WithTimeout(context.Background(), time.Duration((1)*time.Second))
			defer cancelFunc()

			err := li.WaitN(con, 1)
			if err != nil {
				ctx.String(http.StatusForbidden, "限流了")
				return
			}
			next(ctx)
		}
	}
}
