package binding

import "net/http"

/**
 * Binding
 * @Description: 绑定接口
 */
type Binding interface {
	Name() string                  // 绑定类型
	Bind(*http.Request, any) error // 绑定操作
}

var (
	JSON = jsonBinding{}
	XML  = xmlBinding{}
)
