package go_rookie

import "net/http"

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

func (c *Context) HTML(status int, html string) error {
	// 状态值200（即使不设置默认也是200）
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(status)
	_, err := c.W.Write([]byte(html))
	return err
}
