package main

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "log"
    "strings"
)

func main() {
    // Target backend server
    backendURL, err := url.Parse("http://monitoring-prometheus-server.externaltools")
    if (err != nil) {
        log.Fatal(err)
    }
    backendGrafanaURL, err := url.Parse("http://grafana-rancher.monitoring/")
    if (err != nil) {
        log.Fatal(err)
    }

    

    // Create a reverse proxy
    proxy := httputil.NewSingleHostReverseProxy(backendURL)
    proxyGrafana := httputil.NewSingleHostReverseProxy(backendGrafanaURL)

    // Modify the proxy's Director function
    originalDirector := proxy.Director
    proxy.Director = func(req *http.Request) {
        originalDirector(req)
        // Strip /prom prefix
        req.URL.Path = strings.TrimPrefix(req.URL.Path, "/prom")
        if req.URL.Path == "" {
            req.URL.Path = "/"
        }
    }

    originalDirectorGrafana := proxyGrafana.Director
    proxyGrafana.Director = func(req *http.Request) {
        originalDirectorGrafana(req)
        // Strip /prom prefix
        req.URL.Path = strings.TrimPrefix(req.URL.Path, "/grafana")
        if req.URL.Path == "" {
            req.URL.Path = "/"
        }
    }   


    // Add response modifier to handle redirects
    proxy.ModifyResponse = func(resp *http.Response) error {
        // Check if this is a redirect response (3xx status codes)
        if resp.StatusCode >= 300 && resp.StatusCode < 400 {
            location := resp.Header.Get("Location")
            
            // Only modify relative URLs (no host part) and those without /prom prefix
            if location != "" && !strings.Contains(location, "://") && !strings.HasPrefix(location, "/prom") {
                // Add /prom prefix to the path
                resp.Header.Set("Location", "/prom" + location)
                log.Printf("Modified redirect: %s -> %s", location, "/prom" + location)
            }
        }
        return nil
    }

    proxyGrafana.ModifyResponse = func(resp *http.Response) error {
        // Check if this is a redirect response (3xx status codes)
        if resp.StatusCode >= 300 && resp.StatusCode < 400 {
            location := resp.Header.Get("Location")
            
            // Only modify relative URLs (no host part) and those without /prom prefix
            if location != "" && !strings.Contains(location, "://") && !strings.HasPrefix(location, "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana") {
                // Add /prom prefix to the path
                resp.Header.Set("Location", "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana" + location)
                log.Printf("Modified redirect: %s -> %s", location, "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana" + location)
            }
        }
        return nil
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/prom", prometheusHandler(proxy))
    mux.HandleFunc("/prom/", prometheusHandler(proxy)) 
    mux.HandleFunc("/grafana", grafanaHandler(proxyGrafana))
    mux.HandleFunc("/grafana/", grafanaHandler(proxyGrafana)) 

    log.Println("Starting reverse proxy server on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}

func prometheusHandler(proxy *httputil.ReverseProxy) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add cache control headers
        w.Header().Set("Cache-Control", "public")
                
        // Handle the request
        proxy.ServeHTTP(w, r)
    }
}

func grafanaHandler(proxy *httputil.ReverseProxy) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add cache control headers
        w.Header().Set("Cache-Control", "public")
                
        // Handle the request
        proxy.ServeHTTP(w, r)
    }
}