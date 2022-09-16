package go_rookie

import (
	"errors"
	"fmt"
	"github.com/Jack-ZL/go_rookie/grerror"
	"net/http"
	"runtime"
	"strings"
)

/**
 * detailMsg
 * @Author：Jack-Z
 * @Description: 输出错误所在的文件和行号
 * @param err
 * @return string
 */
func detailMsg(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}

	return sb.String()
}

func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				if err2 != nil {
					var grError *grerror.GrError
					if errors.As(err2, &grError) {
						grError.ExecuteResult()
						return
					}
				}

				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		next(ctx)
	}
}
