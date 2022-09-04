package go_rookie

import (
	"fmt"
	"log"
	"net/http"
)

const ANY = "ANY"

type HandlerFunc func(ctx *Context)

type routerGroup struct {
	name             string
	handlerFuncMap   map[string]map[string]HandlerFunc
	handlerMethodMap map[string][]string
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
}

func (r *routerGroup) Any(name string, handlerFunc HandlerFunc) {
	r.handle(name, ANY, handlerFunc)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodGet, handlerFunc)
}

func (r *routerGroup) Post(name string, handlerFunc HandlerFunc) {
	r.handle(name, http.MethodPost, handlerFunc)
}

type router struct {
	routerGroup []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	rg := &routerGroup{
		name:             name,
		handlerFuncMap:   make(map[string]map[string]HandlerFunc),
		handlerMethodMap: make(map[string][]string),
	}
	r.routerGroup = append(r.routerGroup, rg)
	return rg
}

type Engine struct {
	router
}

func New() *Engine {
	return &Engine{
		router: router{},
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routerGroup {
		for name, methodHandle := range group.handlerFuncMap {
			url := "/" + group.name + name
			if r.RequestURI == url {
				ctx := &Context{
					W: w,
					R: r,
				}
				handle, ok := methodHandle[ANY]
				if ok {
					handle(ctx)
					return
				}

				handle, ok = methodHandle[method]
				if ok {
					handle(ctx)
					return
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "%s %s not allowed \n", r.RequestURI, method)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s %s not found \n", r.RequestURI)
}

func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8800", nil)
	if err != nil {
		log.Fatal(err)
	}
}
