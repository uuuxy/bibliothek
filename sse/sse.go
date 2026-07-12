package sse

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
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
		rc := http.NewResponseController(w)
		clearStreamDeadlines(rc)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		// Note: "Connection: keep-alive" is forbidden in HTTP/2 (RFC 9113 §8.2.2) and causes
		// ERR_HTTP2_PROTOCOL_ERROR. SSE streams work natively over both HTTP/1.1 and HTTP/2
		// without this header. CORS is handled by the global CORSMiddleware.

		clientChan := make(chan string, 10)
		b.register <- clientChan

		// Unregister client on connection closure
		defer func() {
			b.unregister <- clientChan
		}()

		streamEvents(w, r, rc, clientChan)
	}
}

// clearStreamDeadlines hebt Read-/Write-Deadlines für den langlebigen SSE-Stream auf,
// damit er nicht durch die Server-Timeouts beendet wird (best-effort: nicht alle
// Transports unterstützen Deadlines).
func clearStreamDeadlines(rc *http.ResponseController) {
	if err := rc.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("SSE: could not clear read deadline: %v", err)
	}
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		log.Printf("SSE: could not clear write deadline: %v", err)
	}
}

// streamEvents sendet den Handshake, 15s-Heartbeats (Dead-Man-Switch) und die
// Broadcast-Nachrichten, bis der Client die Verbindung schließt oder ein Schreibfehler
// (Disconnect) auftritt.
func streamEvents(w http.ResponseWriter, r *http.Request, rc *http.ResponseController, clientChan <-chan string) {
	// writeAndFlush sends one SSE chunk and flushes it. A non-nil error means the
	// client has disconnected, so the caller must terminate the handler.
	writeAndFlush := func(chunk string) error {
		if _, err := io.WriteString(w, chunk); err != nil {
			return err
		}
		return rc.Flush()
	}

	// Send handshake acknowledgment
	if err := writeAndFlush("event: connected\ndata: {\"status\":\"ok\"}\n\n"); err != nil {
		return
	}

	// Heartbeat ticker for dead-man-switch detection (15s is sufficient for library use)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			ping := fmt.Sprintf("event: ping\ndata: {\"timestamp\":%d}\n\n", time.Now().Unix())
			if err := writeAndFlush(ping); err != nil {
				return
			}
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			if err := writeAndFlush(msg); err != nil {
				return
			}
		}
	}
}
