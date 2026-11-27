package theke

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"
	"bibliothek/internal/sse"
)

type festeOmnibox struct{ ergebnis *service.OmniboxErgebnis }

func (o *festeOmnibox) VerarbeiteEingabe(context.Context, service.OmniboxEingabe) (*service.OmniboxErgebnis, error) {
	return o.ergebnis, nil
}

// hoereRundrufe startet einen Broker und hört an ihm zu wie ein Arbeitsplatz. Die gelieferte
// Funktion führt einen Vorgang aus und sammelt die Rundrufe „action", die er sendet: Danach geht
// eine Marke über die Leitung, und was vor ihr ankommt, hat der Vorgang gesendet.
func hoereRundrufe(t *testing.T) (*sse.Broker, func(vorgang func()) []string) {
	t.Helper()
	broker := sse.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	stream := httptest.NewServer(broker.Tuer(func(*http.Request) bool { return true }, func(*http.Request) string { return "" }, func(*http.Request) bool { return true }))
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
		t.Fatalf("erstes Ereignis = %q, erwartet den Handshake connected", e.name)
	}

	marken := 0
	return broker, func(vorgang func()) []string {
		t.Helper()
		vorgang()
		marken++
		marke := fmt.Sprintf("marke-%d", marken)
		broker.Verteile(marke, "{}")
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
}

// Nach einer Ausleihe und nach einer Rückgabe an der Theke, eines Buchs wie eines Geräts, geht
// ein Rundruf an die Arbeitsplätze; ein gescannter Ausweis, ein Hinweis ohne Buchung und die
// Auskunft zu einem Gerät lösen keinen aus.
func TestEingabeVerarbeiten_RundrufBeiAusleiheUndRueckgabe(t *testing.T) {
	broker, rundrufeNach := hoereRundrufe(t)
	scan := func(ergebnis *service.OmniboxErgebnis) func() {
		return func() {
			req := httptest.NewRequest(http.MethodPost, "/api/action", strings.NewReader(`{"query":"B-1"}`))
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
				&auth.Claims{Rolle: auth.RolleAdmin, KontoID: "u1"}))
			w := httptest.NewRecorder()
			(&Tueren{Basis: &tuer.Basis{Broker: broker}}).EingabeVerarbeiten(&festeOmnibox{ergebnis})(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("Scan mit Ergebnis %q: Status %d: %s", ergebnis.Art, w.Code, w.Body.String())
			}
		}
	}

	buch := &repository.Exemplar{ID: "c1", BarcodeID: "B-1", Titel: "Rundruf-Band"}
	geraet := &repository.Geraet{ID: "g1", BarcodeID: "G-1", Modellname: "Rundruf-Tablet"}
	leser := &repository.Leser{ID: "l1"}
	for _, f := range []struct {
		name     string
		ergebnis *service.OmniboxErgebnis
		ereignis string
	}{
		{"Ausleihe", &service.OmniboxErgebnis{Art: "ausleihe", Exemplar: buch, Leser: leser}, "ausleihe"},
		{"Rückgabe", &service.OmniboxErgebnis{Art: "rueckgabe", Exemplar: buch, Leser: leser}, "rueckgabe"},
		{"fremder Rückgabe", &service.OmniboxErgebnis{Art: "rueckgabe", Exemplar: buch, Leser: leser, Fremdrueckgabe: true}, "fremdrueckgabe"},
		{"Ausleihe eines Geräts", &service.OmniboxErgebnis{Art: "ausleihe", Geraet: geraet, Leser: leser}, "ausleihe"},
		{"Rückgabe eines Geräts", &service.OmniboxErgebnis{Art: "rueckgabe", Geraet: geraet, Leser: leser}, "rueckgabe"},
	} {
		rundrufe := rundrufeNach(scan(f.ergebnis))
		if len(rundrufe) != 1 || !strings.Contains(rundrufe[0], `"event":"`+f.ereignis+`"`) || !strings.Contains(rundrufe[0], `"student_id":"l1"`) {
			t.Errorf("nach %s: Rundrufe %q — erwartet genau einen mit event %q und der Kennung des Lesers", f.name, rundrufe, f.ereignis)
			continue
		}
		// Den Rundruf bekommt jede Sitzung, auch das Kollegium. Was ausgeliehen wurde, steht nicht
		// darin: Mit der Kennung des Lesers ergäbe es eine Lesehistorie neben der befristeten.
		for _, angabe := range []string{"Rundruf-Band", "B-1", `"c1"`, "Rundruf-Tablet", "G-1", `"g1"`} {
			if strings.Contains(rundrufe[0], angabe) {
				t.Errorf("nach %s: Rundruf %s nennt, was ausgeliehen wurde (%s)", f.name, rundrufe[0], angabe)
			}
		}
	}
	for _, f := range []struct {
		name     string
		ergebnis *service.OmniboxErgebnis
	}{
		{"einem gescannten Ausweis", &service.OmniboxErgebnis{Art: "student"}},
		{"einem Hinweis ohne Buchung", &service.OmniboxErgebnis{Art: "info", Exemplar: buch}},
		{"der Auskunft zu einem Gerät", &service.OmniboxErgebnis{Art: "geraet_check", Geraet: geraet}},
	} {
		if rundrufe := rundrufeNach(scan(f.ergebnis)); len(rundrufe) != 0 {
			t.Errorf("nach %s: Rundrufe %q — erwartet keinen", f.name, rundrufe)
		}
	}
}
