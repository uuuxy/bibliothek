package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/sse"
)

// Das Live-Signal des LMF-Plans folgt dem Schreiben: Es kommt nach jedem Speichern,
// Veröffentlichen und Verwerfen und bleibt aus, wo nichts geschrieben wurde (Vorschau,
// Ablehnung, ein zweites Veröffentlichen). Geprüft am Strom: Nach jedem Schritt geht eine
// Marke über die Leitung; was vor ihr ankommt, hat der Schritt gesendet.
func TestLmfPlan_SignalNurNachDemSchreiben(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	raeumePlaene := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Pläne aufräumen: %v", err)
		}
	}
	raeumePlaene()

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
	// Erst den Strom des Clients schließen, dann den Broker beenden: stream.Close wartet
	// auf die offene Verbindung.
	t.Cleanup(func() {
		raeumePlaene()
		if cerr := resp.Body.Close(); cerr != nil {
			t.Logf("SSE-Strom schließen: %v", cerr)
		}
		cancel()
		stream.CloseClientConnections()
		stream.Close()
	})

	ereignisse := make(chan string, 32)
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
	// signaleSeitDerLetztenMarke zählt die Plan-Signale, die vor der nächsten Marke ankommen.
	marken := 0
	signaleSeitDerLetztenMarke := func() int {
		t.Helper()
		marken++
		marke := fmt.Sprintf("marke-%d", marken)
		broker.Broadcast(marke, "{}")
		n := 0
		for {
			switch naechstes() {
			case marke:
				return n
			case "lmf-plan":
				n++
			}
		}
	}

	srv := &Server{DB: &db.Database{Pool: pool}, Broker: broker}
	seedSchueler(t, pool, "SIG-1", "Anna", "9H1")
	rahmen := `"letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]`
	plan := `{"letzter_tag":"2027-06-28",` + rahmen + `}`
	korrektur := `{"letzter_tag":"2027-06-30",` + rahmen + `}`
	vorschau := `{"letzter_tag":"2027-06-28",` + rahmen + `,"vorschau":true}`

	for _, s := range []struct {
		name, methode, art, body string
		status, signale          int
	}{
		{"unbekannte Art", http.MethodPut, "quatsch", plan, http.StatusBadRequest, 0},
		{"Vorschau", http.MethodPut, "rueckgabe", vorschau, http.StatusOK, 0},
		{"Verwerfen ohne Plan", http.MethodDelete, "rueckgabe", "", http.StatusNotFound, 0},
		{"Veröffentlichen ohne Plan", http.MethodPost, "rueckgabe", "", http.StatusNotFound, 0},
		{"Entwurf speichern", http.MethodPut, "rueckgabe", plan, http.StatusOK, 1},
		{"Veröffentlichen", http.MethodPost, "rueckgabe", "", http.StatusOK, 1},
		{"schon veröffentlicht", http.MethodPost, "rueckgabe", "", http.StatusOK, 0},
		{"veröffentlichten Plan korrigieren", http.MethodPut, "rueckgabe", korrektur, http.StatusOK, 1},
		{"Verwerfen", http.MethodDelete, "rueckgabe", "", http.StatusOK, 1},
	} {
		rec := lmfPlanAufruf(t, srv, s.methode, s.art, s.body)
		if rec.Code != s.status {
			t.Fatalf("%s: Status %d, erwartet %d: %s", s.name, rec.Code, s.status, rec.Body.String())
		}
		if n := signaleSeitDerLetztenMarke(); n != s.signale {
			t.Errorf("%s: %d Signale, erwartet %d", s.name, n, s.signale)
		}
	}
}
