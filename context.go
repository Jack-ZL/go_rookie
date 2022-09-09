package go_rookie

import (
	"html/template"
	"net/http"
)

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

/**
 * HTML
 * @Author：Jack-Z
 * @Description: 渲染html代码
 * @receiver c
 * @param status
 * @param html
 * @return error
 */
func (c *Context) HTML(status int, html string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(status) // 状态值200（即使不设置默认也是200）
	_, err := c.W.Write([]byte(html))
	return err
}

/**
 * HTMLTemplate
 * @Author：Jack-Z
 * @Description: 渲染html模板文件
 * @receiver c
 * @param name
 * @param data
 * @param filenames
 * @return error
 */
func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}
