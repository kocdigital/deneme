package router

import (
    "net/http"
)

type Middleware func(http.Handler) http.Handler

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log the incoming request
        // For example: log.Printf("Received request: %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func AuthenticationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Implement authentication logic here
        // For example: check for a valid token in the request headers
        next.ServeHTTP(w, r)
    })
}