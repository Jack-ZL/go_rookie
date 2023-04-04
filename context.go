package go_rookie

import (
	"errors"
	"github.com/Jack-ZL/go_rookie/binding"
	grLog "github.com/Jack-ZL/go_rookie/log"
	"github.com/Jack-ZL/go_rookie/render"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const defaultMultipartMemory = 32 << 20 // 默认是分配32M的内存

type Context struct {
	W                     http.ResponseWriter
	R                     *http.Request
	engine                *Engine
	queryCache            url.Values
	formCache             url.Values
	DisallowUnknownFields bool
	IsValidate            bool
	StatusCode            int
	Logger                *grLog.Logger
	Keys                  map[string]any
	mu                    sync.RWMutex
	sameSite              http.SameSite // 降低跨域信息泄露的风险，并为跨站点请求伪造攻击提供一些保护
}

func (c *Context) SetSameSite(s http.SameSite) {
	c.sameSite = s
}

func (c *Context) Set(key string, value any) {
	c.mu.Lock()

	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}
	c.Keys[key] = value
	c.mu.Unlock()
}

func (c *Context) Get(key string) (any, bool) {
	c.mu.Lock()
	value, ok := c.Keys[key]
	c.mu.Unlock()
	return value, ok
}

func (c *Context) SetBasicAuth(username, password string) {
	c.R.Header.Set("Authorization", "GOROOKIE"+BasicAuth(username, password))
}

// 处理query参数，比如：http://xxx.com/user/add?id=1&age=20&username=张三

/**
 * initQueryCache
 * @Author：Jack-Z
 * @Description: 初始化query参数
 * @receiver c
 */
func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
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

// map类型参数获取，比如：http://localhost:8080/queryMap?user[id]=1&user[name]=张三

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
 * initPostFormCache
 * @Author：Jack-Z
 * @Description: 初始化form表单到内存中
 * @receiver c
 */
func (c *Context) initPostFormCache() {
	if c.R != nil {
		if err := c.R.ParseMultipartForm(defaultMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.R.PostForm
	} else {
		c.formCache = url.Values{}
	}
}

/**
 * GetPostFormArray
 * @Author：Jack-Z
 * @Description: 获取form表单参数（多个的array形式的）
 * @receiver c
 * @param key
 * @return []string
 * @return bool
 */
func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.initPostFormCache()
	values, ok := c.formCache[key]
	return values, ok
}

/**
 * PostFormArray
 * @Author：Jack-Z
 * @Description: 获取form表单参数（多个的array形式的），不返回判断结果
 * @receiver c
 * @param key
 * @return []string
 */
func (c *Context) PostFormArray(key string) []string {
	values, _ := c.GetPostFormArray(key)
	return values
}

/**
 * GetPostForm
 * @Author：Jack-Z
 * @Description: 获取form表单参数（单个的）
 * @receiver c
 * @param key
 * @return string
 * @return bool
 */
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

/**
 * GetPostFormMap
 * @Author：Jack-Z
 * @Description: 获取form表单（map形式的）
 * @receiver c
 * @param key
 * @return map[string]string
 * @return bool
 */
func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initPostFormCache()
	return c.get(c.formCache, key)
}

/**
 * PostFormMap
 * @Author：Jack-Z
 * @Description: 获取form表单（map形式的），不返回判断结果
 * @receiver c
 * @param key
 * @return map[string]string
 */
func (c *Context) PostFormMap(key string) map[string]string {
	dicts, _ := c.GetPostFormMap(key)
	return dicts
}

/**
 * FormFile
 * @Author：Jack-Z
 * @Description: form文件获取（单个）
 * @receiver c
 * @param name
 * @return *multipart.FileHeader
 */
func (c *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := c.R.FormFile(name)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	return header
}

/**
 * FormFiles
 * @Author：Jack-Z
 * @Description: form文件获取（同一参数名中包含多个文件的情况）
 * @receiver c
 * @param name
 * @return []*multipart.FileHeader
 */
func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

/**
 * SaveUploadedFile
 * @Author：Jack-Z
 * @Description: 将上传的文件保存
 * @receiver c
 * @param file
 * @param dst
 * @return error
 */
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

/**
 * MultipartForm
 * @Author：Jack-Z
 * @Description: 获取整体的form表单内容（包含普通的key-value和文件类型）
 * @receiver c
 * @return *multipart.Form
 * @return error
 */
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMultipartMemory)
	return c.R.MultipartForm, err
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
	err := r.Render(c.W, statusCode)
	c.StatusCode = statusCode
	// 多次调用WriteHeader 产生：superfluous response.WriteHeader问题
	// if statusCode != http.StatusOK {
	// 	// 如果状态码不是200，才写入状态码，
	// 	// 针对 `http: superfluous response.WriteHeader` 问题
	// 	c.W.WriteHeader(statusCode)
	// }
	return err
}

/**
 * DealJson
 * @Author：Jack-Z
 * @Description: json传参支持
 * @receiver c
 * @param obj
 * @return error
 */
func (c *Context) BindJson(obj any) error {
	json := binding.JSON
	json.DisallowUnknownFields = true
	json.IsValidate = true
	return c.MustBindWith(obj, json)
}

/**
 * MustBindWith
 * @Author：Jack-Z
 * @Description: 通过json绑定器实现参数校验
 * @receiver c
 * @param obj
 * @param bind
 * @return error
 */
func (c *Context) MustBindWith(obj any, bind binding.Binding) error {
	if err := c.ShouldBind(obj, bind); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

/**
 * ShouldBind
 * @Author：Jack-Z
 * @Description: json绑定器实现参数校验
 * @receiver c
 * @param obj
 * @param bind
 * @return error
 */
func (c *Context) ShouldBind(obj any, bind binding.Binding) error {
	return bind.Bind(c.R, obj)
}

/**
 * BindXML
 * @Author：Jack-Z
 * @Description: 通过xml绑定器实现参数校验
 * @receiver c
 * @param obj
 * @return error
 */
func (c *Context) BindXML(obj any) error {
	json := binding.XML
	return c.MustBindWith(obj, json)
}

func (c *Context) Fail(code int, msg string) {
	c.String(code, msg)
}

func (c *Context) HandlerWithError(statusCode int, obj any, err error) {
	if err != nil {
		code, data := c.engine.errorHandler(err)
		c.JSON(code, data)
		return
	}
	c.JSON(statusCode, obj)
}

/**
 * SetCookie
 * @Author：Jack-Z
 * @Description: 设置cookie
 * @receiver c
 * @param name
 * @param value
 * @param maxAge
 * @param path
 * @param domain
 * @param secure
 * @param httpOnly
 */
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.W, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		SameSite: c.sameSite,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

/**
 * GetHeader
 * @Author：Jack-Z
 * @Description: 获取header中的内容
 * @receiver c
 * @param key
 * @return string
 */
func (c *Context) GetHeader(key string) string {
	return c.R.Header.Get(key)
}
