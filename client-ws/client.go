package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Replace with your WebSocket server URL
	serverURL := "ws://10.185.28.113:80/ws"
	
	fmt.Printf("Connecting to %s\n", serverURL)
	
	// Establish WebSocket connection
	c, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer c.Close()
	
	// Set up interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	
	done := make(chan struct{})
	
	// Goroutine to handle incoming messages
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}
			log.Printf("Received: %s", message)
		}
	}()
	
	// Send initial message
	err = c.WriteMessage(websocket.TextMessage, []byte("Hello, WebSocket server!"))
	if err != nil {
		log.Println("write error:", err)
		return
	}
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	// Main loop
	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Ping at %v", t)))
			if err != nil {
				log.Println("write error:", err)
				return
			}
		case <-interrupt:
			log.Println("Shutting down...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("close error:", err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}