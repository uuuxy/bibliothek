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

	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
	"bibliothek/sse"
)

type festeOmnibox struct{ ergebnis *service.OmniboxResult }

func (o *festeOmnibox) ProcessQuery(context.Context, service.OmniboxQuery) (*service.OmniboxResult, error) {
	return o.ergebnis, nil
}

// Nach einer Ausleihe und nach einer Rückgabe an der Theke geht ein Rundruf an die
// Arbeitsplätze; ein gescannter Ausweis und ein Hinweis ohne Buchung lösen keinen aus. Geprüft am Strom: Nach jedem Scan
// geht eine Marke über die Leitung, und was vor ihr ankommt, hat der Scan gesendet.
func TestActionHandler_RundrufBeiAusleiheUndRueckgabe(t *testing.T) {
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
		if cerr := resp.Body.Close(); cerr != nil {
			t.Logf("SSE-Strom schließen: %v", cerr)
		}
		cancel()
		stream.CloseClientConnections()
		stream.Close()
	})

	type ereignis struct{ name, daten string }
	ereignisse := make(chan ereignis, 32)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		name := ""
		for sc.Scan() {
			switch z := sc.Text(); {
			case strings.HasPrefix(z, "event: "):
				name = strings.TrimPrefix(z, "event: ")
			case strings.HasPrefix(z, "data: "):
				ereignisse <- ereignis{name, strings.TrimPrefix(z, "data: ")}
			}
		}
	}()
	naechstes := func() ereignis {
		t.Helper()
		select {
		case e := <-ereignisse:
			return e
		case <-time.After(3 * time.Second):
			t.Fatal("kein SSE-Ereignis innerhalb von 3 s")
			return ereignis{}
		}
	}
	if e := naechstes(); e.name != "connected" {
		t.Fatalf("Handshake fehlt: erwartet connected, war %q", e.name)
	}

	marken := 0
	rundrufeNach := func(ergebnis *service.OmniboxResult) []string {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/action", strings.NewReader(`{"query":"B-1"}`))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{Rolle: auth.RoleAdmin, UserID: "u1"}))
		w := httptest.NewRecorder()
		(&Server{Broker: broker}).ActionHandler(&festeOmnibox{ergebnis})(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Scan mit Ergebnis %q: Status %d: %s", ergebnis.Type, w.Code, w.Body.String())
		}
		marken++
		marke := fmt.Sprintf("marke-%d", marken)
		broker.Broadcast(marke, "{}")
		var rundrufe []string
		for {
			switch e := naechstes(); e.name {
			case marke:
				return rundrufe
			case "action":
				rundrufe = append(rundrufe, e.daten)
			}
		}
	}

	buch := &repository.BookCopy{ID: "c1", BarcodeID: "B-1", Titel: "Rundruf-Band"}
	for _, f := range []struct {
		typ      string
		fremd    bool
		ereignis string
	}{
		{"ausleihe", false, "ausleihe"},
		{"rueckgabe", false, "rueckgabe"},
		{"rueckgabe", true, "fremdrueckgabe"},
	} {
		rundrufe := rundrufeNach(&service.OmniboxResult{Type: f.typ, Book: buch, Fremdrueckgabe: f.fremd})
		if len(rundrufe) != 1 || !strings.Contains(rundrufe[0], `"event":"`+f.ereignis+`"`) || !strings.Contains(rundrufe[0], `"barcode_id":"B-1"`) {
			t.Errorf("nach %s (fremd %v): Rundrufe %q — erwartet genau einen mit event %q und dem Barcode", f.typ, f.fremd, rundrufe, f.ereignis)
		}
	}
	if rundrufe := rundrufeNach(&service.OmniboxResult{Type: "student"}); len(rundrufe) != 0 {
		t.Errorf("nach einem gescannten Ausweis: Rundrufe %q — erwartet keinen", rundrufe)
	}
	// Ein Hinweis zu einem Buch, bei dem nichts gebucht wurde.
	if rundrufe := rundrufeNach(&service.OmniboxResult{Type: "info", Book: buch}); len(rundrufe) != 0 {
		t.Errorf("nach einem Hinweis ohne Buchung: Rundrufe %q — erwartet keinen", rundrufe)
	}
}
