package main

import (
    "log"
    "net/http"
    "custom-http-router/pkg/router"
)

func main() {
    r := router.NewRouter()

    // Define routes
    r.GET("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Welcome to the Custom HTTP Router!"))
    })

    // Start the server
    log.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", r); err != nil {
        log.Fatalf("Could not start server: %s\n", err)
    }
}