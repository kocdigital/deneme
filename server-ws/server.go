package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

// Configuration
var addr = flag.String("addr", ":8080", "http service address")

// Upgrader for WebSocket connections
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all connections (customize in production)
    },
}

// Client represents a connected websocket client
type Client struct {
    conn     *websocket.Conn
    send     chan []byte
    clientID string
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.Mutex
}

// Create new hub
func newHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

// Run the hub
func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            log.Printf("Client connected: %s (total: %d)", client.clientID, len(h.clients))
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
                log.Printf("Client disconnected: %s (total: %d)", client.clientID, len(h.clients))
            }
            h.mu.Unlock()
        case message := <-h.broadcast:
            h.mu.Lock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.Unlock()
        }
    }
}

// Handle WebSocket connections
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Error upgrading connection:", err)
        return
    }

    clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
    client := &Client{
        conn:     conn,
        send:     make(chan []byte, 256),
        clientID: clientID,
    }

    hub.register <- client

    // Send welcome message to new client
    client.send <- []byte(fmt.Sprintf("Welcome! You are connected as %s", clientID))

    // Start goroutines for reading and writing
    go client.writePump(hub)
    go client.readPump(hub)
}

// ReadPump pumps messages from the websocket to the hub
func (c *Client) readPump(hub *Hub) {
    defer func() {
        hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(512 * 1024) // 512KB max message size
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("Error reading from client %s: %v", c.clientID, err)
            }
            break
        }

        // Process the message
        log.Printf("Received from %s: %s", c.clientID, message)
        
        // Echo back to the sender
        c.send <- []byte(fmt.Sprintf("Server received: %s", message))
        
        // Broadcast to all clients
        hub.broadcast <- []byte(fmt.Sprintf("Message from %s: %s", c.clientID, message))
    }
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) writePump(hub *Hub) {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                // Hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued messages
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func main() {
    flag.Parse()
    
    // Create a new hub
    hub := newHub()
    go hub.run()

    // HTTP handler for WebSocket endpoint
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        serveWs(hub, w, r)
    })

    // HTTP handler for health check
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Create server with graceful shutdown
    server := &http.Server{
        Addr:    *addr,
        Handler: nil, // Use default ServeMux
    }

    // Start server in a goroutine
    go func() {
        log.Printf("WebSocket server starting on %s", *addr)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Wait for interrupt signal to gracefully shut down the server
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit

    log.Println("Server is shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exited properly")
}