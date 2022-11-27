package go_rookie

import (
	"fmt"
	"github.com/Jack-ZL/go_rookie/config"
	"github.com/Jack-ZL/go_rookie/gateway"
	grLog "github.com/Jack-ZL/go_rookie/log"
	"github.com/Jack-ZL/go_rookie/register"
	"github.com/Jack-ZL/go_rookie/render"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

const ANY = "ANY"

type HandlerFunc func(ctx *Context)

// 中间件
type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

/**
 * routerGroup
 *  @Description: 路由分组
 */
type routerGroup struct {
	name               string
	handlerFuncMap     map[string]map[string]HandlerFunc
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc
	handlerMethodMap   map[string][]string
	treeNode           *treeNode
	middlewares        []MiddlewareFunc // 请求处理前的中间件
}

func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

func (r *routerGroup) methodHandler(name string, method string, h HandlerFunc, ctx *Context) {
	// 组通用的中间件
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			h = middlewareFunc(h)
		}
	}
	// 路由级别中间件
	middlewareFuncs := r.middlewaresFuncMap[name][method]
	if middlewareFuncs != nil {
		for _, middlewareFunc := range middlewareFuncs {
			h = middlewareFunc(h)
		}
	}
	h(ctx)
}

/**
 * handle
 * @Author：Jack-Z
 * @Description: 路由处理，如“/user/order/12”
 * @receiver r
 * @param name
 * @param method
 * @param handlerFunc
 */
func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	_, ok := r.handlerFuncMap[name]
	if !ok {
		r.handlerFuncMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handlerFuncMap[name][method]
	if ok {
		panic("有重复的路由")
	}
	r.handlerFuncMap[name][method] = handlerFunc
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	r.treeNode.Put(name)
}

/**
 * Any
 * @Author：Jack-Z
 * @Description: 实现各种http的请求方式
 * @receiver r
 * @param name
 * @param handlerFunc
 */
func (r *routerGroup) Any(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, ANY, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Post(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Delete(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Put(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Patch(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Options(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Head(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handlerFunc, middlewareFunc...)
}

type router struct {
	routerGroup []*routerGroup
	engine      *Engine
}

/**
 * Group
 * @Author：Jack-Z
 * @Description: 分组中添加路由
 * @receiver r
 * @param name
 * @return *routerGroup
 */
func (r *router) Group(name string) *routerGroup {
	rg := &routerGroup{
		name:               name,
		handlerFuncMap:     make(map[string]map[string]HandlerFunc),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		handlerMethodMap:   make(map[string][]string),
		treeNode: &treeNode{
			name:     "/",
			children: make([]*treeNode, 0),
		},
	}
	rg.Use(r.engine.middles...)
	r.routerGroup = append(r.routerGroup, rg)
	return rg
}

type ErrorHandler func(err error) (int, any)

type Engine struct {
	router
	funcMap          template.FuncMap
	HTMLRender       render.HTMLRender
	pool             sync.Pool
	Logger           *grLog.Logger
	middles          []MiddlewareFunc
	errorHandler     ErrorHandler
	OpenGateway      bool
	gatewayConfigs   []gateway.GWConfig
	gatewayTreeNode  *gateway.TreeNode
	gatewayConfigMap map[string]gateway.GWConfig
	RegisterType     string              //注册类型
	RegisterOption   register.Option     //注册的配置项
	RegisterCli      register.GrRegister //注册的客户端
}

/**
 * New
 * @Author：Jack-Z
 * @Description: 实例化一个路由
 * @return *Engine
 */
func New() *Engine {
	engine := &Engine{
		router: router{},
		gatewayTreeNode: &gateway.TreeNode{
			Name:     "/",
			Children: make([]*gateway.TreeNode, 0),
		},
		gatewayConfigMap: make(map[string]gateway.GWConfig),
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

func Default() *Engine {
	engine := New()
	engine.Logger = grLog.Default()
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
	}
	engine.Use(Logging, Recovery)
	engine.router.engine = engine
	return engine
}

func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

func (e *Engine) SetGatewayConfig(configs []gateway.GWConfig) {
	e.gatewayConfigs = configs
	// 把这个路径存储起来，访问的时候去匹配里面的路由，匹配到就获取相应的结果
	for _, v := range e.gatewayConfigs {
		e.gatewayTreeNode.Put(v.Path, v.Name)
		e.gatewayConfigMap[v.Name] = v
	}
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

/**
 * LoadTemplate
 * @Author：Jack-Z
 * @Description: 加载模板
 * @receiver e
 * @param pattern
 */
func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHtmlTemplate(t)
}

/**
 * LoadTemplateConf
 * @Author：Jack-Z
 * @Description: 加载模板（按配置文件指定的目录）
 * @receiver e
 */
func (e *Engine) LoadTemplateConf() {
	pattern, ok := config.Conf.Template["pattern"]
	if ok {
		t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
		e.SetHtmlTemplate(t)
	}
}

/**
 * SetHtmlTemplate
 * @Author：Jack-Z
 * @Description: 设置模板
 * @receiver e
 * @param t
 */
func (e *Engine) SetHtmlTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

/**
 * ServeHTTP
 * @Author：Jack-Z
 * @Description: 路由匹配
 * @receiver e
 * @param w
 * @param r
 */
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	ctx.Logger = e.Logger
	e.httpRequestHandler(ctx, w, r)
	e.pool.Put(ctx)
}

/**
 * httpRequestHandler
 * @Author：Jack-Z
 * @Description: 请求处理：网关处理、路由匹配、参数获取等
 * @receiver e
 * @param w
 * @param r
 */
func (e *Engine) httpRequestHandler(ctx *Context, w http.ResponseWriter, r *http.Request) {
	if e.OpenGateway {
		path := r.URL.Path
		node := e.gatewayTreeNode.Get(path)
		if node == nil {
			ctx.W.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(ctx.W, ctx.R.RequestURI+"not found")
			return
		}
		gwConfig := e.gatewayConfigMap[node.GwName]
		gwConfig.Header(ctx.R)
		addr, err := e.RegisterCli.GetValue(gwConfig.ServiceName)
		if err != nil {
			ctx.W.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctx.W, err.Error())
			return
		}
		target, err := url.Parse(fmt.Sprintf("http://%s%s", addr, path))
		if err != nil {
			ctx.W.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctx.W, err.Error())
			return
		}

		//网关处理逻辑
		director := func(req *http.Request) {
			req.Host = target.Host
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.Scheme = target.Scheme
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		}
		//相应处理
		response := func(response *http.Response) error {
			return nil
		}
		//错误处理
		handler := func(writer http.ResponseWriter, request *http.Request, err error) {

		}
		proxy := httputil.ReverseProxy{
			Director:       director,
			ModifyResponse: response,
			ErrorHandler:   handler,
		}
		proxy.ServeHTTP(w, r)
		return
	}

	method := r.Method
	for _, group := range e.routerGroup {
		routerName := SubStringLast(r.URL.Path, "/"+group.name)
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			// 路由匹配
			handle, ok := group.handlerFuncMap[node.routerName][ANY]
			if ok {
				group.methodHandler(node.routerName, ANY, handle, ctx)
				return
			}

			handle, ok = group.handlerFuncMap[node.routerName][method]
			if ok {
				group.methodHandler(node.routerName, method, handle, ctx)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s not allowed \n", r.RequestURI, method)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s not found \n", r.RequestURI)
}

/**
 * Run
 * @Author：Jack-Z
 * @Description: 启动并建监听一个端口
 * @receiver e
 */
func (e *Engine) Run(addr string) {
	if e.RegisterType == "nacos" {
		r := register.GrNacosRegister{}
		err := r.CreateCli(e.RegisterOption)
		if err != nil {
			panic(err)
		}
		e.RegisterCli = &r
	}
	if e.RegisterType == "etcd" {
		r := register.GrEtcdRegister{}
		err := r.CreateCli(e.RegisterOption)
		if err != nil {
			panic(err)
		}
		e.RegisterCli = &r
	}

	http.Handle("/", e)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

/**
 * RunTLS
 * @Author：Jack-Z
 * @Description: 支持https安全
 * @receiver e
 * @param addr
 * @param certFile
 * @param keyFile
 */
func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) Use(middles ...MiddlewareFunc) {
	e.middles = append(e.middles, middles...)
}

/**
 * RegisterErrorHandler
 * @Author：Jack-Z
 * @Description: 注册错误处理器
 * @receiver e
 * @param handler
 */
func (e *Engine) RegisterErrorHandler(handler ErrorHandler) {
	e.errorHandler = handler
}

func (e *Engine) Handler() http.Handler {
	return e
}
