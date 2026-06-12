package sse

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"bibliothek/apierrors"
)

// Broker manages active client connections and broadcasts messages.
type Broker struct {
	clients    map[chan string]bool
	register   chan chan string
	unregister chan chan string
	mu         sync.RWMutex
}

// NewBroker initializes and returns a new Broker.
func NewBroker() *Broker {
	return &Broker{
		clients:    make(map[chan string]bool),
		register:   make(chan chan string),
		unregister: make(chan chan string),
	}
}

// Start runs the broker's event loop in a background goroutine.
// Handles client registrations, deregistrations, and lifetime events.
func (b *Broker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case clientChan := <-b.register:
			b.mu.Lock()
			b.clients[clientChan] = true
			b.mu.Unlock()
			log.Println("SSE: New client registered")
		case clientChan := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[clientChan]; ok {
				delete(b.clients, clientChan)
				close(clientChan)
			}
			b.mu.Unlock()
			log.Println("SSE: Client disconnected")
		}
	}
}

// Broadcast sends a message to all currently connected SSE clients.
func (b *Broker) Broadcast(event, data string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	formattedMessage := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	for clientChan := range b.clients {
		select {
		case clientChan <- formattedMessage:
		default:
			// Non-blocking send; skip if client is lagging or channel is full
		}
	}
}

// Handler returns an http.HandlerFunc that establishes an SSE connection.
// Sets necessary streaming headers and handles the 1-second ping (heartbeat/dead-man-switch).
func (b *Broker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		clientChan := make(chan string, 10)
		b.register <- clientChan

		// Unregister client on connection closure
		defer func() {
			b.unregister <- clientChan
		}()

		// Send handshake acknowledgment
		_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
		flusher.Flush()

		// Heartbeat ticker for dead-man-switch detection
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				// Write heartbeat ping
				_, _ = fmt.Fprintf(w, "event: ping\ndata: {\"timestamp\":%d}\n\n", time.Now().Unix())
				flusher.Flush()
			case msg, ok := <-clientChan:
				if !ok {
					return
				}
				_, _ = fmt.Fprint(w, msg)
				flusher.Flush()
			}
		}
	}
}
