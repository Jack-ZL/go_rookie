package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type GrHttpClient struct {
	client     http.Client
	serviceMap map[string]GrService // 服务集合
}

/**
 * NewHttpClient
 * @Author：Jack-Z
 * @Description:
 * @return *GrHttpClient
 */
func NewHttpClient() *GrHttpClient {
	client := http.Client{
		Timeout: time.Duration(3) * time.Second, // 超时时间（3秒）
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &GrHttpClient{
		client:     client,
		serviceMap: make(map[string]GrService),
	}
}

/**
 * GetRequest
 * @Author：Jack-Z
 * @Description: 进阶版——http的get请求
 * @receiver c
 * @param method 请求方法
 * @param url 请求地址
 * @param args 参数集合
 * @return *http.Request
 * @return error
 */
func (c *GrHttpClient) GetRequest(method string, url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

/**
 * FormRequest
 * @Author：Jack-Z
 * @Description: 进阶版——http的post请求（from表单参数）
 * @receiver c
 * @param method
 * @param url
 * @param args
 * @return *http.Request
 * @return error
 */
func (c *GrHttpClient) FormRequest(method string, url string, args map[string]any) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

/**
 * JsonRequest
 * @Author：Jack-Z
 * @Description: 进阶版——http的post请求（json格式的参数）
 * @receiver c
 * @param method
 * @param url
 * @param args
 * @return *http.Request
 * @return error
 */
func (c *GrHttpClient) JsonRequest(method string, url string, args map[string]any) (*http.Request, error) {
	jsonStr, _ := json.Marshal(args)
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *GrHttpClientSession) Response(req *http.Request) ([]byte, error) {
	return c.responseHandler(req)
}

/**
 * Get
 * @Author：Jack-Z
 * @Description: http的get请求
 * @receiver c
 * @param url  请求地址
 * @param args 请求参数
 * @return []byte
 * @return error
 */
func (c *GrHttpClientSession) Get(url string, args map[string]any) ([]byte, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

/**
 * PostForm
 * @Author：Jack-Z
 * @Description: http的post请求（form表单的参数）
 * @receiver c
 * @param url
 * @param args
 * @return []byte
 * @return error
 */
func (c *GrHttpClientSession) PostForm(url string, args map[string]any) ([]byte, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

/**
 * PostJson
 * @Author：Jack-Z
 * @Description: http的post请求（json格式的请求参数）
 * @receiver c
 * @param url
 * @param args
 * @return []byte
 * @return error
 */
func (c *GrHttpClientSession) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", url, bytes.NewReader(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

/**
 * responseHandler
 * @Author：Jack-Z
 * @Description: 响应的处理
 * @receiver c
 * @param request
 * @return []byte
 * @return error
 */
func (c *GrHttpClientSession) responseHandler(request *http.Request) ([]byte, error) {
	c.ReqHandler(request)
	do, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if do.StatusCode != http.StatusOK {
		errinfo := fmt.Sprintf("response status id %d", do.StatusCode)
		return nil, errors.New(errinfo)
	}
	reader := bufio.NewReader(do.Body)
	defer do.Body.Close()
	var buf = make([]byte, 127)
	var body []byte
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil
}

/**
 * toValues
 * @Author：Jack-Z
 * @Description: 请求参数组装处理
 * @receiver c
 * @param args
 * @return string
 */
func (c *GrHttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""
}

/**
 * HttpConfig
 *  @Description: 配置项
 */
type HttpConfig struct {
	Protocol string // 协议类型
	Host     string // 域名
	Port     int    // 端口号
}

// 不同的协议和方法常量
const (
	HTTP     = "http"
	HTTPS    = "https"
	GET      = "GET"
	POSTFORM = "POST_FORM"
	POSTJSON = "POST_JSON"
)

/**
 * GrService
 * @Description: 配置项读取
 */
type GrService interface {
	Env() HttpConfig
}

/**
 * RegisterHttpService
 * @Author：Jack-Z
 * @Description: 注册服务
 * @receiver c
 * @param name 服务名称
 * @param service 提供者
 */
func (c *GrHttpClient) RegisterHttpService(name string, service GrService) {
	c.serviceMap[name] = service
}

type GrHttpClientSession struct {
	*GrHttpClient
	ReqHandler func(req *http.Request)
}

func (c *GrHttpClient) Session() *GrHttpClientSession {
	return &GrHttpClientSession{
		c, nil,
	}
}

/**
 * Do
 * @Author：Jack-Z
 * @Description: 服务执行
 * @receiver c
 * @param service 服务名称
 * @param method  方法
 * @return GrService
 */
func (c *GrHttpClientSession) Do(service string, method string) GrService {
	grService, ok := c.serviceMap[service]
	if !ok {
		panic(errors.New("service not found"))
	}

	t := reflect.TypeOf(grService)
	v := reflect.ValueOf(grService)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("service type not pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	fieldIndex := -1
	for i := 0; i < tVar.NumField(); i++ {
		name := tVar.Field(i).Name
		if name == method {
			fieldIndex = i
			break
		}
	}

	if fieldIndex == -1 {
		panic(errors.New("method not found"))
	}
	tag := tVar.Field(fieldIndex).Tag
	rpcInfo := tag.Get("grpc")
	if rpcInfo == "" {
		panic(errors.New("not rpc tag "))
	}
	split := strings.Split(rpcInfo, ",")
	if len(split) != 2 {
		panic(errors.New("tag grpc not valid"))
	}
	methodType := split[0]
	path := split[1]
	httpConfig := grService.Env()
	f := func(args map[string]any) ([]byte, error) {
		if methodType == GET {
			return c.Get(httpConfig.Prefix()+path, args)
		}

		if methodType == POSTFORM {
			return c.PostForm(httpConfig.Prefix()+path, args)
		}

		if methodType == POSTJSON {
			return c.PostJson(httpConfig.Prefix()+path, args)
		}

		return nil, errors.New("no match method type")
	}
	fValue := reflect.ValueOf(f)
	vVar.Field(fieldIndex).Set(fValue)
	return grService
}

/**
 * Prefix
 * @Author：Jack-Z
 * @Description: 请求的地址前缀
 * @receiver h
 * @return string
 */
func (h HttpConfig) Prefix() string {
	if h.Protocol == "" {
		h.Protocol = HTTP
	}
	switch h.Protocol {
	case HTTP:
		return fmt.Sprintf("http://%s:%d", h.Host, h.Port)

	case HTTPS:
		return fmt.Sprintf("https://%s:%d", h.Host, h.Port)
	}
	return ""
}
