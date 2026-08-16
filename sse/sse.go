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

// clientPuffer ist die Anzahl Nachrichten, die ein langsamer Arbeitsplatz zurückhängen
// darf, bevor Broadcast ihn überspringt. Zehn Ereignisse entsprechen bei uns mehreren
// Sekunden Bedienung am Tresen.
const clientPuffer = 10

// Broker verteilt Ereignisse an alle offenen SSE-Ströme.
//
// Der Zustand liegt hinter einem RWMutex, nicht hinter einer Ereignisschleife mit
// register-/unregister-Kanälen. Die frühere Fassung hatte beides — und die Kanäle waren
// nicht nur überflüssig, sondern der Grund, warum kein Herunterfahren mehr sauber
// gelang: Sobald Start durch den abgebrochenen Kontext zurückkehrte, las niemand mehr
// aus ihnen, und jeder SSE-Handler blieb in seinem `defer b.unregister <- …` für immer
// stehen. httpServer.Shutdown wartet auf genau diese Handler, lief in seinen Timeout,
// und main endete mit os.Exit(1). In einer Schule mit dauerhaft verbundenen
// Arbeitsplätzen war das jeder Deploy.
type Broker struct {
	mu         sync.RWMutex
	clients    map[chan string]struct{}
	geschlosen bool
}

// NewBroker initializes and returns a new Broker.
func NewBroker() *Broker {
	return &Broker{clients: make(map[chan string]struct{})}
}

// Start hält den Broker am Leben, bis ctx abgebrochen wird, und fährt ihn dann herunter.
// Gedacht für `go broker.Start(ctx)`.
//
// Das Herunterfahren schließt alle Client-Kanäle. Das ist der Abbruchsignal für die
// laufenden Ströme: Ohne ihn wartete jeder Handler bis zu 15 Sekunden auf seinen nächsten
// Heartbeat — länger als das Shutdown-Zeitfenster von 10 Sekunden in main.go.
func (b *Broker) Start(ctx context.Context) {
	<-ctx.Done()
	b.shutdown()
}

// shutdown schließt alle offenen Ströme und weist spätere Anmeldungen ab.
func (b *Broker) shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.geschlosen {
		return
	}
	b.geschlosen = true
	for clientChan := range b.clients {
		close(clientChan)
		delete(b.clients, clientChan)
	}
	log.Println("SSE: Broker heruntergefahren, alle Ströme geschlossen")
}

// subscribe meldet einen neuen Strom an. Rückgabe nil bedeutet: Der Broker fährt gerade
// herunter — der Handler muss die Anfrage abweisen, statt zu warten.
func (b *Broker) subscribe() chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.geschlosen {
		return nil
	}
	clientChan := make(chan string, clientPuffer)
	b.clients[clientChan] = struct{}{}
	log.Println("SSE: New client registered")
	return clientChan
}

// unsubscribe meldet einen Strom ab und schließt seinen Kanal. Mehrfach aufrufbar: Nach
// dem Herunterfahren ist der Kanal bereits geschlossen und aus der Liste entfernt, ein
// zweites close() würde sonst das Programm abbrechen.
func (b *Broker) unsubscribe(clientChan chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, offen := b.clients[clientChan]; !offen {
		return
	}
	delete(b.clients, clientChan)
	close(clientChan)
	log.Println("SSE: Client disconnected")
}

// Broadcast sends a message to all currently connected SSE clients.
//
// Die RLock deckt zugleich das Senden ab: unsubscribe und shutdown schließen Kanäle unter
// der Schreibsperre, und ein Senden auf einen geschlossenen Kanal bricht das Programm ab.
// Beides darf sich deshalb nie überschneiden.
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
// Sets necessary streaming headers and handles the heartbeat (dead-man-switch).
func (b *Broker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientChan := b.subscribe()
		if clientChan == nil {
			// Der Broker fährt herunter. 503 statt eines Handlers, der das
			// Herunterfahren blockiert; der Browser verbindet sich später neu.
			http.Error(w, "Server fährt herunter", http.StatusServiceUnavailable)
			return
		}
		defer b.unsubscribe(clientChan)

		rc := http.NewResponseController(w)
		clearStreamDeadlines(rc)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		// Note: "Connection: keep-alive" is forbidden in HTTP/2 (RFC 9113 §8.2.2) and causes
		// ERR_HTTP2_PROTOCOL_ERROR. SSE streams work natively over both HTTP/1.1 and HTTP/2
		// without this header. CORS is handled by the global CORSMiddleware.

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
// Broadcast-Nachrichten, bis der Client die Verbindung schließt, der Broker
// herunterfährt oder ein Schreibfehler (Disconnect) auftritt.
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
				return // Broker heruntergefahren oder Client abgemeldet
			}
			if err := writeAndFlush(msg); err != nil {
				return
			}
		}
	}
}
