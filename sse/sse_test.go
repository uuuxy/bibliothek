package sse

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/pkg/closeutil"
)

// Fährt der Server herunter, während ein Arbeitsplatz einen Strom offen hat, endet erst
// FuehreAus, danach räumt httpServer.Shutdown die offenen Verbindungen ab, und jede Tür meldet
// ihr Fenster ab. Blockiert das Abmelden, wartet Shutdown bis zu seiner Frist, und main endet
// mit os.Exit(1). Arbeitsplätze sind den ganzen Tag verbunden; es träfe jedes Update.
func TestMeldeAbNachDemHerunterfahrenBlockiertNicht(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)

	fenster := broker.meldeAn()
	if fenster == nil {
		t.Fatal("Anmeldung am laufenden Broker muss gelingen")
	}

	// Der Server fährt herunter.
	cancel()
	warteAufHerunterfahren(t, broker)

	// Jetzt läuft die Tür in ihr defer. Das darf nicht hängen.
	fertig := make(chan struct{})
	go func() {
		broker.meldeAb(fenster)
		close(fertig)
	}()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("meldeAb nach dem Herunterfahren blockiert — genau hier hängt die " +
			"Tür, und httpServer.Shutdown läuft in seine Frist")
	}
}

// Trifft nach dem Signal noch eine Verbindung ein, wird sie abgewiesen. Eine wartende Tür hielte
// das Herunterfahren genauso auf.
func TestAnmeldungNachDemHerunterfahrenWirdAbgelehnt(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	cancel()
	warteAufHerunterfahren(t, broker)

	ergebnis := make(chan chan string, 1)
	go func() { ergebnis <- broker.meldeAn() }()

	select {
	case ch := <-ergebnis:
		if ch != nil {
			t.Error("nach dem Herunterfahren darf sich kein Client mehr anmelden")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("meldeAn nach dem Herunterfahren blockiert statt abzulehnen")
	}
}

// Die Türen warten an ihrem Kanal. Bleiben die Kanäle beim Herunterfahren offen, wartet jede
// Tür bis zu 15 Sekunden auf das nächste Lebenszeichen, länger als die zehn Sekunden, die
// main.go dem Herunterfahren gibt.
func TestHerunterfahrenSchliesstOffeneStroeme(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)

	fenster := broker.meldeAn()
	if fenster == nil {
		t.Fatal("Anmeldung muss gelingen")
	}

	cancel()

	select {
	case _, ok := <-fenster:
		if ok {
			t.Error("erwartet war ein geschlossener Kanal als Abbruchsignal")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("der Client-Kanal wurde beim Herunterfahren nicht geschlossen — " +
			"der Handler bliebe bis zum nächsten Heartbeat stehen")
	}
}

// warteAufHerunterfahren wartet, bis FuehreAus das Herunterfahren verarbeitet hat.
func warteAufHerunterfahren(t *testing.T, b *Broker) {
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

// ── Gewöhnlicher Betrieb ───────────────────────────────────────────────────────

// Der Zweck des Pakets: Mehrere Arbeitsplätze sind verbunden, eine Buchung an einem kommt an
// allen anderen an.
func TestVerteileErreichtAlleFenster(t *testing.T) {
	broker := NewBroker()

	fenster := make([]chan string, 3)
	for i := range fenster {
		fenster[i] = broker.meldeAn()
		if fenster[i] == nil {
			t.Fatalf("Client %d konnte sich nicht anmelden", i)
		}
	}

	broker.Verteile("ausleihe", `{"barcode":"B-00042"}`)

	const erwartet = "event: ausleihe\ndata: {\"barcode\":\"B-00042\"}\n\n"
	for i, c := range fenster {
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

// Ein Senden auf einen geschlossenen Kanal beendet den Prozess. meldeAb schließt den Kanal;
// Verteile darf ihn danach nicht mehr kennen.
func TestAbgemeldetesFensterBekommtNichtsMehr(t *testing.T) {
	broker := NewBroker()
	bleibt := broker.meldeAn()
	geht := broker.meldeAn()

	broker.meldeAb(geht)
	broker.Verteile("test", "{}") // sendete sonst auf einem geschlossenen Kanal

	select {
	case <-bleibt:
	case <-time.After(time.Second):
		t.Error("der verbliebene Client hätte die Nachricht bekommen müssen")
	}
	if _, ok := <-geht; ok {
		t.Error("der abgemeldete Kanal muss geschlossen sein")
	}
}

// Ein Arbeitsplatz, dessen Browser hängt, bremst die Theke nicht mit aus: Verteile sendet, ohne
// zu warten, und überspringt volle Puffer.
func TestLangsamesFensterBremstDieAnderenNicht(t *testing.T) {
	broker := NewBroker()
	langsam := broker.meldeAn()
	schnell := broker.meldeAn()

	// Den Puffer des langsamen Fensters randvoll laufen lassen.
	for i := 0; i < pufferJeFenster+5; i++ {
		broker.Verteile("fuellen", "{}")
		<-schnell // das schnelle Fenster liest mit
	}

	fertig := make(chan struct{})
	go func() { broker.Verteile("wichtig", "{}"); close(fertig) }()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("Verteile blockiert an einem vollen Client-Puffer")
	}

	select {
	case got := <-schnell:
		if got != "event: wichtig\ndata: {}\n\n" {
			t.Errorf("der schnelle Client bekam %q", got)
		}
	case <-time.After(time.Second):
		t.Error("der schnelle Client wurde vom langsamen ausgebremst")
	}
	if len(langsam) != pufferJeFenster {
		t.Errorf("Puffer des langsamen Clients: %d, erwartet %d (voll)", len(langsam), pufferJeFenster)
	}
}

// Unter -race: Arbeitsplätze kommen und gehen, während Ereignisse verteilt werden, wie im
// Betrieb.
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
					broker.Verteile("tick", "{}")
				}
			}
		}()
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if c := broker.meldeAn(); c != nil {
					broker.meldeAb(c)
				}
			}
		}()
	}

	// Die Sender laufen, bis alle An- und Abmeldungen durch sind.
	go func() { time.Sleep(300 * time.Millisecond); close(stop) }()
	wg.Wait()
}

// ── Die Tür über echtes HTTP ───────────────────────────────────────────────────

// Geprüft wird der Weg über HTTP statt nur der Broker: die Kopfzeilen, die Begrüßung, und ein
// verteiltes Ereignis kommt beim Browser an.
func TestTuerLiefertStromUndEreignisse(t *testing.T) {
	broker := NewBroker()
	srv := httptest.NewServer(broker.Tuer())
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
	defer closeutil.Schliesse(resp.Body, "sse-test response body")

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, erwartet text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, erwartet no-cache", got)
	}
	// HTTP/2 verbietet das Kopffeld Connection; es darf hier nicht auftauchen.
	if got := resp.Header.Get("Connection"); got == "keep-alive" {
		t.Error("Connection: keep-alive gesetzt — bricht SSE über HTTP/2")
	}

	leser := bufio.NewReader(resp.Body)
	if zeile := leseZeile(t, leser); zeile != "event: connected" {
		t.Fatalf("Handshake erwartet, bekam %q", zeile)
	}

	// Warten, bis die Tür ihr Fenster angemeldet hat, sonst geht das Ereignis ins Leere.
	warteAufFensterzahl(t, broker, 1)
	broker.Verteile("rueckgabe", `{"barcode":"B-1"}`)

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

// Kommt nach dem Signal noch eine Verbindung, antwortet die Tür mit 503, statt das
// Herunterfahren aufzuhalten.
func TestTuerWeistNachDemHerunterfahrenAb(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	cancel()
	warteAufHerunterfahren(t, broker)

	srv := httptest.NewServer(broker.Tuer())
	defer srv.Close()

	resp, err := http.Get(srv.URL) //nolint:noctx // Testaufruf gegen den lokalen Testserver
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	defer closeutil.Schliesse(resp.Body, "sse-test response body")

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

func warteAufFensterzahl(t *testing.T, b *Broker, n int) {
	t.Helper()
	frist := time.Now().Add(2 * time.Second)
	for time.Now().Before(frist) {
		b.mu.RLock()
		da := len(b.fenster)
		b.mu.RUnlock()
		if da == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("es waren nicht %d Clients angemeldet", n)
}

// istGeschlossen meldet, ob der Broker heruntergefahren ist. Nur die Tests fragen danach;
// deshalb steht es hier und nicht im Programm.
func (b *Broker) istGeschlossen() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.geschlossen
}
