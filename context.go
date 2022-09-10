package go_rookie

import (
	"fmt"
	"github.com/Jack-ZL/go_rookie/render"
	"html/template"
	"net/http"
	"net/url"
)

type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	engine *Engine
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
	return c.Render(status, &render.HTML{
		Data:       html,
		IsTemplate: false,
	})
}

/**
 * HTMLTemplate
 * @Author：Jack-Z
 * @Description: 渲染html模板文件（多文件名模式）
 * @receiver c
 * @param name
 * @param data
 * @param filenames
 * @return error
 */
func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

/**
 * HTMLTemplate
 * @Author：Jack-Z
 * @Description: 渲染html模板文件（通配符模式）
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

/**
 * Template
 * @Author：Jack-Z
 * @Description: 模板渲染-通用型
 * @receiver c
 * @param name
 * @param data
 * @return error
 */
func (c *Context) Template(name string, data any) error {
	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		IsTemplate: true,
		Template:   c.engine.HTMLRender.Template,
		Name:       name,
	})
}

/**
 * JSON
 * @Author：Jack-Z
 * @Description: json格式的数据渲染
 * @receiver c
 * @param status
 * @param data
 * @return error
 */
func (c *Context) JSON(status int, data any) error {
	return c.Render(status, &render.JSON{
		Data: data,
	})
}

/**
 * XML
 * @Author：Jack-Z
 * @Description: XML数据渲染
 * @receiver c
 * @param status
 * @param data
 * @return error
 */
func (c *Context) XML(status int, data any) error {
	return c.Render(status, &render.XML{
		Data: data,
	})
}

/**
 * File
 * @Author：Jack-Z
 * @Description: 文件下载
 * @receiver c
 * @param filename
 */
func (c *Context) File(filename string) {
	http.ServeFile(c.W, c.R, filename)
}

/**
 * FileAttachment
 * @Author：Jack-Z
 * @Description: 文件下载（可以自定义名称）
 * @receiver c
 * @param filepath
 * @param filename
 */
func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

/**
 * FileFromFS
 * @Author：Jack-Z
 * @Description:从文件系统下载
 * @receiver c
 * @param filepath 相对文件系统的路径
 * @param fs
 */
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)

	c.R.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

/**
 * Redirect
 * @Author：Jack-Z
 * @Description: url重定向
 * @receiver c
 * @param status
 * @param url
 */
func (c *Context) Redirect(status int, url string) {
	// status状态码判断
	if (status < http.StatusMultipleChoices || status > http.StatusPermanentRedirect) && status != http.StatusCreated {
		panic(fmt.Sprintf("Cannot redirect with status code %d", status))
	}

	http.Redirect(c.W, c.R, url, status)
}

/**
 * String
 * @Author：Jack-Z
 * @Description: 字符串渲染和格式化
 * @receiver c
 * @param status
 * @param format
 * @param values
 * @return error
 */
func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{
		Format: format,
		Data:   values,
	})
}

/**
 * Render
 * @Author：Jack-Z
 * @Description: 抽离的公共方法——渲染
 * @receiver c
 * @param statusCode
 * @param w
 * @param r
 * @return error
 */
func (c *Context) Render(statusCode int, r render.Render) error {
	err := r.Render(c.W)
	c.W.WriteHeader(statusCode)
	return err
}
