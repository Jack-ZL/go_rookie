package go_rookie

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(status) // 状态值200（即使不设置默认也是200）
	_, err := c.W.Write([]byte(html))
	return err
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
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := c.engine.HTMLRender.Template.ExecuteTemplate(c.W, name, data)
	return err
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
	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.W.WriteHeader(status)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.W.Write(jsonData)
	return err
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
	c.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.W.WriteHeader(status)
	err := xml.NewEncoder(c.W).Encode(data)
	return err
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
