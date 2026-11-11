package tuertest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/sse"
)

// BeobachteLeitung öffnet an einem neuen Broker einen Strom und liefert den Broker und eine
// Zählung: Sie schickt eine Marke über die Leitung und nennt, wie oft ein Ereignis davor ankam.
// Was vor der Marke ankommt, hat der Schritt davor gesendet; so lässt sich auch prüfen, dass ein
// Schritt nichts sendet, ohne auf Stille zu warten.
func BeobachteLeitung(t *testing.T) (broker *sse.Broker, zaehleBisMarke func(ereignis string) int) {
	t.Helper()
	broker = sse.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	// Der Strom gilt als vertraut: Er sieht die Fassung eines Ereignisses mit Kennungen.
	immer := func(*http.Request) bool { return true }
	strom := httptest.NewServer(broker.Tuer(immer, func(*http.Request) string { return "" }, immer))
	t.Cleanup(func() {
		cancel()
		strom.CloseClientConnections()
		strom.Close()
	})
	// Dieser Aufräumer läuft zuerst: Den Strom des Clients schließen, bevor der Broker endet, denn
	// strom.Close wartet auf die offene Verbindung.
	antwort := oeffneStrom(ctx, t, strom.URL)
	t.Cleanup(func() {
		if err := antwort.Body.Close(); err != nil {
			t.Logf("Leitung schließen: %v", err)
		}
	})

	ereignisse := leseEreignisse(antwort.Body)
	if e := naechstesEreignis(t, ereignisse); e != "connected" {
		t.Fatalf("erstes Ereignis = %q, erwartet connected", e)
	}

	marken := 0
	return broker, func(ereignis string) int {
		t.Helper()
		marken++
		marke := fmt.Sprintf("marke-%d", marken)
		broker.Verteile(marke, "{}")
		n := 0
		for {
			switch naechstesEreignis(t, ereignisse) {
			case marke:
				return n
			case ereignis:
				n++
			}
		}
	}
}

// oeffneStrom öffnet den Strom der Leitung; er bleibt offen, bis ctx endet oder jemand ihn schließt.
func oeffneStrom(ctx context.Context, t *testing.T, adresse string) *http.Response {
	t.Helper()
	anfrage, err := http.NewRequestWithContext(ctx, http.MethodGet, adresse, http.NoBody)
	if err != nil {
		t.Fatalf("Anfrage an die Leitung bauen: %v", err)
	}
	antwort, err := http.DefaultClient.Do(anfrage)
	if err != nil {
		t.Fatalf("Leitung nicht erreichbar: %v", err)
	}
	return antwort
}

// leseEreignisse liefert die Namen der Ereignisse auf dem Strom, bis er endet.
func leseEreignisse(strom io.Reader) <-chan string {
	ereignisse := make(chan string, 32)
	go func() {
		zeilen := bufio.NewScanner(strom)
		for zeilen.Scan() {
			if z := zeilen.Text(); strings.HasPrefix(z, "event: ") {
				ereignisse <- strings.TrimPrefix(z, "event: ")
			}
		}
	}()
	return ereignisse
}

// naechstesEreignis wartet höchstens drei Sekunden auf das nächste Ereignis.
func naechstesEreignis(t *testing.T, ereignisse <-chan string) string {
	t.Helper()
	select {
	case e := <-ereignisse:
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("kein Ereignis auf der Leitung innerhalb von 3 s")
		return ""
	}
}
