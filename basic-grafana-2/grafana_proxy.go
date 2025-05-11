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
        path := strings.TrimPrefix(req.URL.Path, basePath)
        if path == "" {
            path = "/"
        }
        
        req.URL.Path = path
        req.URL.Host = grafanaURL.Host
        req.URL.Scheme = grafanaURL.Scheme
        req.Host = grafanaURL.Host
        
        originalDirector(req)
    }

    // Modify responses to fix Grafana URLs
    proxy.ModifyResponse = modifyGrafanaResponse(basePath)

    // WebSocket upgrader
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }

    // Main handler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if !strings.HasPrefix(r.URL.Path, basePath) {
            http.NotFound(w, r)
            return
        }
        
        // Handle WebSocket
        if isWebSocketRequest(r) {
            handleWebSocket(w, r, basePath, grafanaURL.Host, upgrader)
        } else {
            // Standard HTTP proxy
            proxy.ServeHTTP(w, r)
        }
    })

    // Start the server
    log.Println("Starting Grafana proxy on :8080")
    log.Println("Grafana available at: " + basePath)
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal("Server error:", err)
    }
}

// Check if request is a WebSocket request
func isWebSocketRequest(r *http.Request) bool {
    return strings.Contains(r.URL.Path, "/api/live/") && 
        strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// Handle WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request, basePath, host string, upgrader websocket.Upgrader) {
    // Strip the base path from the URL
    r.URL.Path = strings.TrimPrefix(r.URL.Path, basePath)
    
    // Create a dialer to the backend
    backendURL := url.URL{
        Scheme: "ws",
        Host:   host,
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
}

// Returns a ModifyResponse function
func modifyGrafanaResponse(basePath string) func(*http.Response) error {
    return func(resp *http.Response) error {
        contentType := resp.Header.Get("Content-Type")
        
        isHTML := strings.Contains(contentType, "text/html")
        isJS := strings.Contains(contentType, "application/javascript") 
        isJSON := strings.Contains(contentType, "application/json")
        
        if isHTML || isJS || isJSON {
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