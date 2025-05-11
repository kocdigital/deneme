package router

import (
    "net/http"
    "sync"
)

type Route struct {
    Path    string
    Handler http.HandlerFunc
}

type Router struct {
    routes []Route
    mu     sync.RWMutex
}

func NewRouter() *Router {
    return &Router{
        routes: []Route{},
    }
}

func (r *Router) HandleFunc(path string, handler http.HandlerFunc) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.routes = append(r.routes, Route{Path: path, Handler: handler})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, route := range r.routes {
        if req.URL.Path == route.Path {
            route.Handler(w, req)
            return
        }
    }
    http.NotFound(w, req)
}