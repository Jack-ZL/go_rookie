package go_rookie

import (
	"fmt"
	"log"
	"net/http"
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
	name             string
	handlerFuncMap   map[string]map[string]HandlerFunc
	handlerMethodMap map[string][]string
	treeNode         *treeNode
	preMiddlewares   []MiddlewareFunc // 请求处理前的中间件
	postMiddlewares  []MiddlewareFunc // 请求处理后的中间件
}

func (r *routerGroup) PreMiddlewares(middlewareFunc ...MiddlewareFunc) {
	r.preMiddlewares = append(r.preMiddlewares, middlewareFunc...)
}

func (r *routerGroup) PostMiddlewares(middlewareFunc ...MiddlewareFunc) {
	r.postMiddlewares = append(r.postMiddlewares, middlewareFunc...)
}

func (r *routerGroup) methodHandler(h HandlerFunc, ctx *Context) {
	// 前置中间件
	if r.preMiddlewares != nil {
		for _, middlewareFunc := range r.preMiddlewares {
			h = middlewareFunc(h)
		}
	}

	h(ctx)
}

// func (r *routerGroup) Add(name string, handlerFunc HandlerFunc) {
// 	r.handlerFuncMap[name] = handlerFunc
// }

func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc) {
	_, ok := r.handlerFuncMap[name]
	if !ok {
		r.handlerFuncMap[name] = make(map[string]HandlerFunc)
	}
	_, ok = r.handlerFuncMap[name][method]
	if ok {
		panic("有重复的路由")
	}
	r.handlerFuncMap[name][method] = handlerFunc
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
func (r *routerGroup) Any(name string, handlerFunc HandlerFunc) {
	r.handle(name, ANY, handlerFunc)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodGet, handlerFunc)
}

func (r *routerGroup) Post(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodPost, handlerFunc)
}

func (r *routerGroup) Delete(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodDelete, handlerFunc)
}
func (r *routerGroup) Put(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodPut, handlerFunc)
}
func (r *routerGroup) Patch(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodPatch, handlerFunc)
}
func (r *routerGroup) Options(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodOptions, handlerFunc)
}
func (r *routerGroup) Head(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodHead, handlerFunc)
}

type router struct {
	routerGroup []*routerGroup
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
		name:             name,
		handlerFuncMap:   make(map[string]map[string]HandlerFunc),
		handlerMethodMap: make(map[string][]string),
		treeNode: &treeNode{
			name:     "/",
			children: make([]*treeNode, 0),
		},
	}
	r.routerGroup = append(r.routerGroup, rg)
	return rg
}

type Engine struct {
	router
}

/**
 * New
 * @Author：Jack-Z
 * @Description: 实例化一个路由
 * @return *Engine
 */
func New() *Engine {
	return &Engine{
		router: router{},
	}
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
	e.httpRequestHandler(w, r)
}

func (e *Engine) httpRequestHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routerGroup {
		routerName := SubStringLast(r.RequestURI, "/"+group.name)
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			// 路由匹配
			ctx := &Context{
				W: w,
				R: r,
			}
			handle, ok := group.handlerFuncMap[node.routerName][ANY]
			if ok {
				group.methodHandler(handle, ctx)
				return
			}

			handle, ok = group.handlerFuncMap[node.routerName][method]
			if ok {
				group.methodHandler(handle, ctx)
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
func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8800", nil)
	if err != nil {
		log.Fatal(err)
	}
}
