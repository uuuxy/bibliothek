package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/sse"
)

// Die Sitzungsfristen (Theke leeren, Sperrbildschirm) holt jeder Tab genau einmal beim
// Anmelden (App.svelte). Ohne Signal liefe ein offener Tab am ZWEITEN Arbeitsplatz bis
// zum nächsten F5 mit den alten Fristen weiter — im Zweifel mit einer Sperre, die der
// Admin gerade abgeschaltet hat, oder ohne die, die er gerade eingeschaltet hat.
// Deshalb: Speichern der Sitzungsfelder sendet `sitzungsfristen` über die bestehende
// SSE-Leitung; die Werte selbst holen sich die Tabs per GET (dort sitzt die
// Vorgaben-Logik, hier wäre sie dupliziert).
//
// Geprüft am echten Strom, nicht an einer Attrappe des Brokers: erst ein Patch OHNE
// Sitzungsfelder, dann einer MIT — das nächste Ereignis auf der Leitung muss
// `sitzungsfristen` sein. Hätte der erste Patch gesendet, stünde ein Ereignis davor.
func TestUpdateSettings_SitzungsfristenSendenSSESignal(t *testing.T) {
	broker := sse.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)

	stream := httptest.NewServer(broker.Handler())
	resp, err := http.Get(stream.URL)
	if err != nil {
		cancel()
		stream.Close()
		t.Fatalf("SSE-Strom nicht erreichbar: %v", err)
	}
	// Abbau in DIESER Reihenfolge — und in einem Cleanup statt in defers: Ein
	// `defer stream.Close()` lief vor `cancel()` und vor dem Schließen des Client-Bodys.
	// httptest.Server.Close wartet aber auf alle aktiven Verbindungen, und der
	// Strom-Handler (sse.streamEvents) beendet sich erst, wenn der Client geht
	// (r.Context) oder der Broker herunterfährt (clientChan). Beides stand noch aus —
	// der Test hing bis zum 10-Minuten-Timeout des Pakets und riss den Pre-Push mit.
	t.Cleanup(func() {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Logf("SSE-Strom schließen: %v", cerr)
		}
		cancel()
		stream.CloseClientConnections()
		stream.Close()
	})

	ereignisse := make(chan string, 8)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if z := sc.Text(); strings.HasPrefix(z, "event: ") {
				ereignisse <- strings.TrimPrefix(z, "event: ")
			}
		}
	}()
	naechstes := func() string {
		t.Helper()
		select {
		case e := <-ereignisse:
			return e
		case <-time.After(3 * time.Second):
			t.Fatal("kein SSE-Ereignis innerhalb von 3 s")
			return ""
		}
	}
	if e := naechstes(); e != "connected" {
		t.Fatalf("Handshake fehlt: erwartet connected, war %q", e)
	}

	s := &Server{Broker: broker}
	speichere := func(rumpf string) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/api/einstellungen", strings.NewReader(rumpf))
		s.UpdateSettingsHandler(&attrappeSettingsRepo{}).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("Speichern scheiterte (%d): %s", w.Code, w.Body.String())
		}
	}

	speichere(`{"lesehistorie_tage": 120}`)
	speichere(`{"sperre_minuten": 0}`)
	if e := naechstes(); e != "sitzungsfristen" {
		t.Fatalf("erwartet sitzungsfristen als nächstes Ereignis, war %q", e)
	}
}
