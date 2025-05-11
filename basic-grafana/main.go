package main

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "log"
    "strings"
    "io"
    "bytes"
    "fmt"
    )

func main() {
    backendGrafanaURL, err := url.Parse("http://grafana-rancher.monitoring/")
    if (err != nil) {
        log.Fatal(err)
    }

    // Create a reverse proxy
    proxyGrafana := httputil.NewSingleHostReverseProxy(backendGrafanaURL)

    // Modify the proxy's Director function
    originalDirectorGrafana := proxyGrafana.Director
    proxyGrafana.Director = func(req *http.Request) {
        originalDirectorGrafana(req)
        // Strip /grafana prefix
        req.URL.Path = strings.TrimPrefix(req.URL.Path, "/grafana")
        if req.URL.Path == "" {
            req.URL.Path = "/"
        }
    }   

    // Add ModifyResponse to log headers and body
    proxyGrafana.ModifyResponse = func(resp *http.Response) error {
        // Log response status and headers
        log.Printf("Response Status: %s", resp.Status)
        log.Println("Response Headers:")
        for key, values := range resp.Header {
            for _, value := range values {
                log.Printf("\t%s: %s", key, value)
            }
        }
        
        // Handle cookies - modify paths to match our proxy path
        targetPath := "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana"
        if cookies := resp.Header["Set-Cookie"]; len(cookies) > 0 {
            for i, cookie := range cookies {
                // Update cookie path from /grafana to our target path
                if strings.Contains(cookie, "Path=/grafana") {
                    cookies[i] = strings.Replace(cookie, "Path=/grafana", "Path="+targetPath, 1)
                } else if strings.Contains(cookie, "Path=/") {
                    cookies[i] = strings.Replace(cookie, "Path=/", "Path="+targetPath, 1)
                }
            }
            resp.Header["Set-Cookie"] = cookies
        }
        
        // Read and log the body only if content type is HTML
        contentType := resp.Header.Get("Content-Type")
        if resp.Body != nil && strings.Contains(strings.ToLower(contentType), "text/html") {
            bodyBytes, err := io.ReadAll(resp.Body)
            if err != nil {
                log.Printf("Error reading body: %v", err)
                return err
            }
            
            // Close original body
            resp.Body.Close()
            
            // Replace appSubUrl with the k8s path
            targetPath := "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana"
            bodyBytes = bytes.Replace(
                bodyBytes,
                []byte(`"appSubUrl":"/grafana"`),
                []byte(`"appSubUrl":"`+targetPath+`"`),
                -1,
            )
            
            // Log the modified HTML body 
            log.Printf("HTML Response Body: %s", bodyBytes)
            
            // Restore the body for the client
            resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
            
            // Update Content-Length if needed
            resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
        } else if resp.Body != nil {
            // For non-HTML responses, just ensure the body is preserved
            bodyBytes, err := io.ReadAll(resp.Body)
            if err != nil {
                log.Printf("Error reading non-HTML body: %v", err)
                return err
            }
            resp.Body.Close()
            resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
        }
        
        return nil
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/grafana", grafanaHandler(proxyGrafana))
    mux.HandleFunc("/grafana/", grafanaHandler(proxyGrafana)) 

    log.Println("Starting reverse proxy server on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}


func grafanaHandler(proxy *httputil.ReverseProxy) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add cache control headers
        w.Header().Set("Cache-Control", "public")
                
        // Handle the request
        proxy.ServeHTTP(w, r)
    }
}