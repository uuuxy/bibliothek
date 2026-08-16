package sse

import (
	"bufio"
	"context"

	"bibliothek/pkg/closeutil"

	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestUnregisterNachShutdownBlockiertNicht hält den Fehler fest, der bei JEDEM Deploy
// zuschlägt, sobald ein Arbeitsplatz einen Live-Stream offen hat.
//
// Ablauf in main.go: SIGTERM beendet den ctx → broker.Start kehrt zurück → niemand liest
// mehr aus register/unregister. Danach räumt httpServer.Shutdown die offenen Verbindungen
// ab; jeder SSE-Handler läuft in sein `defer b.unregister <- clientChan` und bleibt dort
// für immer stehen, weil der Kanal ungepuffert und der Leser weg ist. Shutdown wartet auf
// diese Handler, läuft in seinen 10-Sekunden-Timeout und main beendet sich mit os.Exit(1).
//
// In einer Schule mit mehreren Arbeitsplätzen, die den ganzen Tag verbunden sind, ist das
// nicht der Ausnahme-, sondern der Regelfall.
func TestUnregisterNachShutdownBlockiertNicht(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)

	client := broker.subscribe()
	if client == nil {
		t.Fatal("Anmeldung am laufenden Broker muss gelingen")
	}

	// Der Server fährt herunter.
	cancel()
	wartAufAbmeldungAllerClients(t, broker)

	// Jetzt läuft der Handler in sein defer. Das darf nicht hängen.
	fertig := make(chan struct{})
	go func() {
		broker.unsubscribe(client)
		close(fertig)
	}()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("unsubscribe nach dem Herunterfahren blockiert — genau hier hängt der " +
			"SSE-Handler, und httpServer.Shutdown läuft in seinen Timeout")
	}
}

// TestAnmeldungNachShutdownWirdAbgelehnt: Trifft nach dem Signal noch eine Verbindung
// ein, muss sie abgewiesen werden. Ein blockierender Handler verhindert sonst das
// Herunterfahren genauso.
func TestAnmeldungNachShutdownWirdAbgelehnt(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	cancel()
	wartAufAbmeldungAllerClients(t, broker)

	ergebnis := make(chan chan string, 1)
	go func() { ergebnis <- broker.subscribe() }()

	select {
	case ch := <-ergebnis:
		if ch != nil {
			t.Error("nach dem Herunterfahren darf sich kein Client mehr anmelden")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe nach dem Herunterfahren blockiert statt abzulehnen")
	}
}

// TestShutdownSchliesstOffeneStreams: Die Handler hängen an ihrem Kanal. Werden die
// Kanäle beim Herunterfahren nicht geschlossen, wartet jeder Handler bis zu 15 Sekunden
// auf den nächsten Heartbeat — länger als das Shutdown-Zeitfenster von 10 Sekunden.
func TestShutdownSchliesstOffeneStreams(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)

	client := broker.subscribe()
	if client == nil {
		t.Fatal("Anmeldung muss gelingen")
	}

	cancel()

	select {
	case _, ok := <-client:
		if ok {
			t.Error("erwartet war ein geschlossener Kanal als Abbruchsignal")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("der Client-Kanal wurde beim Herunterfahren nicht geschlossen — " +
			"der Handler bliebe bis zum nächsten Heartbeat stehen")
	}
}

// wartAufAbmeldungAllerClients wartet, bis Start das Herunterfahren verarbeitet hat.
func wartAufAbmeldungAllerClients(t *testing.T, b *Broker) {
	t.Helper()
	frist := time.Now().Add(2 * time.Second)
	for time.Now().Before(frist) {
		if b.istGeschlossen() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("Broker hat das Herunterfahren nicht verarbeitet")
}

// ── Normalbetrieb ──────────────────────────────────────────────────────────────

// TestBroadcastErreichtAlleClients: Der Zweck des Pakets. Mehrere Arbeitsplätze sind
// verbunden, eine Buchung an einem muss an allen anderen ankommen.
func TestBroadcastErreichtAlleClients(t *testing.T) {
	broker := NewBroker()

	clients := make([]chan string, 3)
	for i := range clients {
		clients[i] = broker.subscribe()
		if clients[i] == nil {
			t.Fatalf("Client %d konnte sich nicht anmelden", i)
		}
	}

	broker.Broadcast("ausleihe", `{"barcode":"B-00042"}`)

	const erwartet = "event: ausleihe\ndata: {\"barcode\":\"B-00042\"}\n\n"
	for i, c := range clients {
		select {
		case got := <-c:
			if got != erwartet {
				t.Errorf("Client %d bekam %q, erwartet %q", i, got, erwartet)
			}
		case <-time.After(time.Second):
			t.Errorf("Client %d hat die Nachricht nicht bekommen", i)
		}
	}
}

// TestAbgemeldeterClientBekommtNichtsMehr verhindert die Rückkehr eines Absturzes:
// Senden auf einen geschlossenen Kanal beendet den Prozess. unsubscribe schließt den
// Kanal — Broadcast darf ihn danach nicht mehr kennen.
func TestAbgemeldeterClientBekommtNichtsMehr(t *testing.T) {
	broker := NewBroker()
	bleibt := broker.subscribe()
	geht := broker.subscribe()

	broker.unsubscribe(geht)
	broker.Broadcast("test", "{}") // dürfte sonst auf einem geschlossenen Kanal senden

	select {
	case <-bleibt:
	case <-time.After(time.Second):
		t.Error("der verbliebene Client hätte die Nachricht bekommen müssen")
	}
	if _, ok := <-geht; ok {
		t.Error("der abgemeldete Kanal muss geschlossen sein")
	}
}

// TestLangsamerClientBlockiertDieAnderenNicht: Ein Arbeitsplatz, dessen Browser hängt,
// darf den Tresen nicht mit ausbremsen. Broadcast sendet nicht-blockierend und
// überspringt volle Puffer.
func TestLangsamerClientBlockiertDieAnderenNicht(t *testing.T) {
	broker := NewBroker()
	langsam := broker.subscribe()
	schnell := broker.subscribe()

	// Puffer des langsamen Clients randvoll laufen lassen.
	for i := 0; i < clientPuffer+5; i++ {
		broker.Broadcast("fuellen", "{}")
		<-schnell // der schnelle Client liest brav mit
	}

	fertig := make(chan struct{})
	go func() { broker.Broadcast("wichtig", "{}"); close(fertig) }()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcast blockiert an einem vollen Client-Puffer")
	}

	select {
	case got := <-schnell:
		if got != "event: wichtig\ndata: {}\n\n" {
			t.Errorf("der schnelle Client bekam %q", got)
		}
	case <-time.After(time.Second):
		t.Error("der schnelle Client wurde vom langsamen ausgebremst")
	}
	if len(langsam) != clientPuffer {
		t.Errorf("Puffer des langsamen Clients: %d, erwartet %d (voll)", len(langsam), clientPuffer)
	}
}

// TestNebenlaeufigkeit läuft unter -race und deckt das ab, was im Betrieb wirklich
// passiert: Arbeitsplätze kommen und gehen, während Ereignisse verteilt werden.
func TestNebenlaeufigkeit(t *testing.T) {
	broker := NewBroker()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					broker.Broadcast("tick", "{}")
				}
			}
		}()
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if c := broker.subscribe(); c != nil {
					broker.unsubscribe(c)
				}
			}
		}()
	}

	// Die Sender laufen, bis alle An-/Abmelder durch sind.
	go func() { time.Sleep(300 * time.Millisecond); close(stop) }()
	wg.Wait()
}

// ── Handler über echtes HTTP ───────────────────────────────────────────────────

// TestHandlerLiefertStromUndEreignisse prüft den Live-Pfad statt nur den Broker:
// richtige Kopfzeilen, Handshake, und ein Broadcast kommt beim Browser an.
func TestHandlerLiefertStromUndEreignisse(t *testing.T) {
	broker := NewBroker()
	srv := httptest.NewServer(broker.Handler())
	defer srv.Close()

	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Verbindung: %v", err)
	}
	defer closeutil.LogClose(resp.Body, "sse-test response body")

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, erwartet text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, erwartet no-cache", got)
	}
	// HTTP/2 verbietet Connection-Header; er darf hier nicht auftauchen.
	if got := resp.Header.Get("Connection"); got == "keep-alive" {
		t.Error("Connection: keep-alive gesetzt — bricht SSE über HTTP/2")
	}

	leser := bufio.NewReader(resp.Body)
	if zeile := leseZeile(t, leser); zeile != "event: connected" {
		t.Fatalf("Handshake erwartet, bekam %q", zeile)
	}

	// Warten, bis der Handler tatsächlich angemeldet ist, sonst geht der Broadcast ins Leere.
	wartAufClientZahl(t, broker, 1)
	broker.Broadcast("rueckgabe", `{"barcode":"B-1"}`)

	var sah string
	for i := 0; i < 6; i++ {
		if zeile := leseZeile(t, leser); zeile == "event: rueckgabe" {
			sah = zeile
			break
		}
	}
	if sah == "" {
		t.Error("das gesendete Ereignis kam nicht im Strom an")
	}
}

// TestHandlerWeistNachShutdownAb: Kommt nach dem Signal noch eine Verbindung, muss der
// Handler mit 503 antworten statt das Herunterfahren zu blockieren.
func TestHandlerWeistNachShutdownAb(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	cancel()
	wartAufAbmeldungAllerClients(t, broker)

	srv := httptest.NewServer(broker.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL) //nolint:noctx // Testaufruf gegen den lokalen Testserver
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	defer closeutil.LogClose(resp.Body, "sse-test response body")

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("HTTP %d, erwartet 503 — sonst hängt der Handler beim Herunterfahren",
			resp.StatusCode)
	}
}

func leseZeile(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	zeile, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("Strom lesen: %v", err)
	}
	return strings.TrimRight(zeile, "\r\n")
}

func wartAufClientZahl(t *testing.T, b *Broker, n int) {
	t.Helper()
	frist := time.Now().Add(2 * time.Second)
	for time.Now().Before(frist) {
		b.mu.RLock()
		da := len(b.clients)
		b.mu.RUnlock()
		if da == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("es waren nicht %d Clients angemeldet", n)
}

// istGeschlossen meldet, ob der Broker heruntergefahren ist — reine
// Test-Beobachtung, deshalb lebt sie hier und nicht in der Produktionsdatei.
func (b *Broker) istGeschlossen() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.geschlosen
}
