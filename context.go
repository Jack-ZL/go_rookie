package go_rookie

import (
	"github.com/Jack-ZL/go_rookie/render"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

type Context struct {
	W          http.ResponseWriter
	R          *http.Request
	engine     *Engine
	queryCache url.Values
}

// 处理query参数，比如：http://xxx.com/user/add?id=1&age=20&username=张三
/**
 * GetDefaultQuery
 * @Author：Jack-Z
 * @Description: 获取参数，没有或为空 就用默认值
 * @receiver c
 * @param key
 * @param defaultValue
 * @return string
 */
func (c *Context) GetDefaultQuery(key, defaultValue string) string {
	values, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

/**
 * GetQuery
 * @Author：Jack-Z
 * @Description: 获取query参数
 * @receiver c
 * @param key
 * @return string
 */
func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

/**
 * QueryArray
 * @Author：Jack-Z
 * @Description: 获取query参数（数组形式的多个参数），返回值不带判断
 * @receiver c
 * @param key
 * @return []string
 */
func (c *Context) QueryArray(key string) []string {
	c.initQueryCache()
	values, _ := c.queryCache[key]
	return values
}

/**
 * GetQueryArray
 * @Author：Jack-Z
 * @Description: 获取query参数（数组形式的多个参数）
 * @receiver c
 * @param key
 * @return []string
 * @return bool
 */
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryCache()
	values, ok := c.queryCache[key]
	return values, ok
}

func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

// map类型参数获取，比如：http://localhost:8080/queryMap?user[id]=1&user[name]=张三

/**
 * QueryMap
 * @Author：Jack-Z
 * @Description: map类型参数获取（不返回判断值）
 * @receiver c
 * @param key
 * @return dicts
 */
func (c *Context) QueryMap(key string) map[string]string {
	dicts, _ := c.GetQueryMap(key)
	return dicts
}

/**
 * GetQueryMap
 * @Author：Jack-Z
 * @Description: map类型参数获取
 * @receiver c
 * @param key
 * @return map[string]string
 * @return bool
 */
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}

/**
 * get
 * @Author：Jack-Z
 * @Description: 通过字符串函数定位左右中括号的位置，来获取键值和参数值，并赋值到map中
 * @receiver c
 * @param cache
 * @param key
 * @return map[string]string
 * @return bool
 */
func (c *Context) get(cache map[string][]string, key string) (map[string]string, bool) {
	// user[id]=1&user[name]=张三
	dicts := make(map[string]string)
	exist := false
	for k, value := range cache {
		// 左中括号 “[” 的位置，并且[不在第一位
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 { // 右中括号 “]” 的位置
				exist = true
				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
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
func (c *Context) Redirect(status int, url string) error {
	return c.Render(status, &render.Redirect{
		Code:     status,
		Request:  c.R,
		Location: url,
	})
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
	if statusCode != http.StatusOK {
		// 如果状态码不是200，才写入状态码，
		// 针对 `http: superfluous response.WriteHeader` 问题
		c.W.WriteHeader(statusCode)
	}
	return err
}
