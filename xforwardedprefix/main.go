package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// BackendServer represents our simple backend application
func BackendServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Backend Server\n\n")
	fmt.Fprintf(w, "Request URL: %s\n", r.URL.String())
	fmt.Fprintf(w, "Request Path: %s\n", r.URL.Path)
	
	// Display X-Forwarded-Prefix header if present
	prefix := r.Header.Get("X-Forwarded-Prefix")
	if prefix != "" {
		fmt.Fprintf(w, "X-Forwarded-Prefix: %s\n", prefix)
		fmt.Fprintf(w, "Full Path (with prefix): %s%s\n", prefix, r.URL.Path)
	} else {
		fmt.Fprintf(w, "X-Forwarded-Prefix: Not set\n")
	}
	
	// Display all request headers
	fmt.Fprintf(w, "\nAll Request Headers:\n")
	for name, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(w, "%s: %s\n", name, value)
		}
	}
}

// CustomReverseProxy creates a reverse proxy with X-Forwarded-Prefix handling
func CustomReverseProxy(targetHost string, pathPrefix string) http.Handler {
	target, err := url.Parse(targetHost)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Store the original director function
	originalDirector := proxy.Director
	
	// Create a custom director that modifies the request
	proxy.Director = func(req *http.Request) {
		// Call the original director
		originalDirector(req)
		
		// Add X-Forwarded-Prefix header
		req.Header.Add("X-Forwarded-Prefix", pathPrefix)
		
		// Trim the prefix from the request path
		if strings.HasPrefix(req.URL.Path, pathPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefix)
		}
		
		// If path is empty after trimming, set it to "/"
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		
		log.Printf("Proxied request: %s -> %s%s", req.RequestURI, targetHost, req.URL.Path)
	}
	
	return proxy
}

func main() {
	// Start the backend server on port 8080
	http.HandleFunc("/", BackendServer)
	go func() {
		log.Println("Starting backend server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Configure multiple path-based proxies on port 9090
	mux := http.NewServeMux()
	
	// Route /api/* requests to the backend with /api prefix
	mux.Handle("/api/", http.StripPrefix("/api", CustomReverseProxy("http://localhost:8080", "/api")))
	
	// Route /app/* requests to the backend with /app prefix
	mux.Handle("/app/", http.StripPrefix("/app", CustomReverseProxy("http://localhost:8080", "/app")))
	
	// Route /v1/* requests to the backend with /v1 prefix
	mux.Handle("/v1/", http.StripPrefix("/v1", CustomReverseProxy("http://localhost:8080", "/v1")))
	
	// Direct requests to the backend without prefix handling
	mux.HandleFunc("/direct", func(w http.ResponseWriter, r *http.Request) {
		proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost:8080"})
		proxy.ServeHTTP(w, r)
	})
	
	// Root handler with instructions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		fmt.Fprintf(w, "X-Forwarded-Prefix Demo\n\n")
		fmt.Fprintf(w, "Try the following routes:\n")
		fmt.Fprintf(w, "1. /api/test - Proxied with X-Forwarded-Prefix: /api\n")
		fmt.Fprintf(w, "2. /app/users - Proxied with X-Forwarded-Prefix: /app\n")
		fmt.Fprintf(w, "3. /v1/data - Proxied with X-Forwarded-Prefix: /v1\n")
		fmt.Fprintf(w, "4. /direct - Proxied without X-Forwarded-Prefix\n")
	})
	
	// Start the proxy server on port 9090
	log.Println("Starting reverse proxy server on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}