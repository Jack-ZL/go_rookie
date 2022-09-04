package go_rookie

import (
	"fmt"
	"log"
	"net/http"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request)

type routerGroup struct {
	name             string
	handlerFuncMap   map[string]HandlerFunc
	handlerMethodMap map[string][]string
}

func (r *routerGroup) Add(name string, handlerFunc HandlerFunc) {
	r.handlerFuncMap[name] = handlerFunc
}

func (r *routerGroup) Any(name string, handlerFunc HandlerFunc) {
	r.handlerFuncMap[name] = handlerFunc
	r.handlerMethodMap["ANY"] = append(r.handlerMethodMap["ANY"], name)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc) {
	r.handlerFuncMap[name] = handlerFunc
	r.handlerMethodMap[http.MethodGet] = append(r.handlerMethodMap[http.MethodGet], name)
}

func (r *routerGroup) Post(name string, handlerFunc HandlerFunc) {
	r.handlerFuncMap[name] = handlerFunc
	r.handlerMethodMap[http.MethodPost] = append(r.handlerMethodMap[http.MethodPost], name)
}

type router struct {
	routerGroup []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	rg := &routerGroup{
		name:             name,
		handlerFuncMap:   make(map[string]HandlerFunc),
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
				routers, ok := group.handlerMethodMap["ANY"]
				if ok {
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
				}
				// method进行匹配
				routers, ok = group.handlerMethodMap[method]
				if ok {
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
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
