package main

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
    "regexp"

    "github.com/gorilla/websocket"
)

func main() {
    // Target Grafana server
    grafanaURL, err := url.Parse("http://grafana-rancher.monitoring")
    if err != nil {
        log.Fatal("Invalid Grafana URL:", err)
    }

    // Base path for our proxy
    basePath := "/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana"

    // Create reverse proxy
    proxy := httputil.NewSingleHostReverseProxy(grafanaURL)
    
    // Customize the director function
    originalDirector := proxy.Director
    proxy.Director = func(req *http.Request) {
        // Store original path for debugging
        originalPath := req.URL.Path
        
        // Get the path relative to the basePath
        path := strings.TrimPrefix(req.URL.Path, basePath)
        if path == "" {
            path = "/"
        }
        
        // Set the URL path to the backend path
        req.URL.Path = path
        req.URL.Host = grafanaURL.Host
        req.URL.Scheme = grafanaURL.Scheme
        req.Host = grafanaURL.Host
        
        // Add debug headers
        req.Header.Set("X-Debug-Original-Path", originalPath)
        req.Header.Set("X-Debug-Modified-Path", req.URL.Path)
        
        log.Printf("Transforming request path: %s → %s", originalPath, req.URL.Path)
        
        originalDirector(req)
    }

    // Modify responses to fix Grafana URLs
    proxy.ModifyResponse = func(resp *http.Response) error {
        contentType := resp.Header.Get("Content-Type")
        
        // Add debug headers
        resp.Header.Set("X-Debug-Response-Status", resp.Status)
        resp.Header.Set("X-Debug-Content-Type", contentType)
        
        // Handle cookies - modify paths to match our proxy path
        if cookies := resp.Header["Set-Cookie"]; len(cookies) > 0 {
            for i, cookie := range cookies {
                if strings.Contains(cookie, "Path=/grafana") {
                    cookies[i] = strings.Replace(cookie, "Path=/grafana", "Path="+basePath, 1)
                } else if strings.Contains(cookie, "Path=/") {
                    cookies[i] = strings.Replace(cookie, "Path=/", "Path="+basePath, 1)
                }
            }
            resp.Header["Set-Cookie"] = cookies
        }
        
        // Create a simple check for content types
        isHtmlOrJsOrJson := false
        if strings.Contains(contentType, "text/html") {
            isHtmlOrJsOrJson = true
        }
        if strings.Contains(contentType, "application/javascript") {
            isHtmlOrJsOrJson = true
        }
        if strings.Contains(contentType, "application/json") {
            isHtmlOrJsOrJson = true
        }
        
        if isHtmlOrJsOrJson {
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                return err
            }
            resp.Body.Close()

            // Replace appSubUrl in JSON responses
            body = bytes.ReplaceAll(
                body,
                []byte(`"appSubUrl":""`),
                []byte(`"appSubUrl":"`+basePath+`"`)
            )
            
            // Fix any duplicate paths
            body = bytes.ReplaceAll(
                body,
                []byte(basePath+"/grafana/"),
                []byte(basePath+"/")
            )
            
            // Fix duplicate paths with regex pattern
            doubleGrafanaPattern := regexp.MustCompile(`(` + regexp.QuoteMeta(basePath) + `)/grafana`)
            body = doubleGrafanaPattern.ReplaceAll(body, []byte("$1"))
            
            // Fix appSubUrl issue for embedded Grafana
            body = bytes.ReplaceAll(
                body,
                []byte(`"appSubUrl":"/grafana"`),
                []byte(`"appSubUrl":"`+basePath+`"`)
            )
            
            // Replace avatar URLs
            body = bytes.ReplaceAll(
                body, 
                []byte(`:"/avatar/`),
                []byte(`:"avatar/`)
            )
            
            // Update content length
            resp.Body = io.NopCloser(bytes.NewBuffer(body))
            resp.Header.Set("Content-Length", fmt.Sprint(len(body)))
        }
        
        return nil
    }

    // WebSocket upgrader
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }

    // Main handler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Add debug headers
        w.Header().Set("X-Debug-Original-Url", r.URL.String())
        w.Header().Set("X-Debug-Original-Path", r.URL.Path)
        w.Header().Set("X-Debug-Original-Host", r.Host)
        
        if !strings.HasPrefix(r.URL.Path, basePath) {
            http.NotFound(w, r)
            return
        }
        
        // Check WebSocket request separately to avoid line breaks in conditionals
        pathHasLiveAPI := strings.Contains(r.URL.Path, "/api/live/")
        upgradeIsWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
        isWebSocketRequest := pathHasLiveAPI && upgradeIsWebSocket
        
        if isWebSocketRequest {
            // Strip the base path from the URL
            r.URL.Path = strings.TrimPrefix(r.URL.Path, basePath)
            
            // Create a dialer to the backend
            backendURL := url.URL{
                Scheme: "ws",
                Host:   grafanaURL.Host,
                Path:   r.URL.Path,
            }
            
            // Connect to the backend WebSocket
            backendConn, _, err := websocket.DefaultDialer.Dial(backendURL.String(), nil)
            if err != nil {
                http.Error(w, "Could not connect to backend", http.StatusInternalServerError)
                return
            }
            defer backendConn.Close()
            
            // Upgrade the client connection
            clientConn, err := upgrader.Upgrade(w, r, nil)
            if err != nil {
                return
            }
            defer clientConn.Close()
            
            // Proxy the WebSocket connection
            go proxyWebsocket(clientConn, backendConn)
            proxyWebsocket(backendConn, clientConn)
            
        } else {
            // Standard HTTP proxy
            w.Header().Set("Cache-Control", "public")
            proxy.ServeHTTP(w, r)
        }
    })

    // Start the server
    log.Println("Starting Grafana proxy on :8080")
    log.Println("Grafana available at: " + basePath)
    err = http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("Server error:", err)
    }
}

// Function to proxy WebSocket connections bidirectionally
func proxyWebsocket(dst, src *websocket.Conn) {
    for {
        messageType, message, err := src.ReadMessage()
        if err != nil {
            break
        }
        err = dst.WriteMessage(messageType, message)
        if err != nil {
            break
        }
    }
}