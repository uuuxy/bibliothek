package tuer

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/sse"
)

// Die Live-Leitung verlangt nur eine Anmeldung, und nach jeder Ausleihe und Rückgabe ging die
// Kennung des Lesers mit der Uhrzeit an jedes angemeldete Konto, auch an eine Lehrkraft, die nur
// „Mein Portal" sieht und sonst an keine Leserdaten kommt. Die Kennung bekommt jetzt nur, wer an
// der Theke bucht (perform_actions) oder Leser einsehen darf (view_students); alle anderen
// erfahren weiter, dass gebucht wurde, aber nicht, bei wem.
func TestSendeThekenEreignis_KennungNurMitThekenOderLeserRecht(t *testing.T) {
	broker := sse.NewBroker()
	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	go broker.FuehreAus(ctx)

	basis := &Basis{Broker: broker}
	for _, r := range []struct {
		rolle                auth.Rolle
		theke, leserEinsicht bool
	}{
		{auth.RolleKollegium, false, false},
		{auth.RolleHelfer, true, false},
		{auth.RolleMitarbeiter, false, true},
	} {
		basis.Rechte.MerkeFuerTest(string(r.rolle), "perform_actions", r.theke)
		basis.Rechte.MerkeFuerTest(string(r.rolle), "view_students", r.leserEinsicht)
	}

	// Die Tür wie im Router, hinter einer Anmeldung, die hier die Rolle aus dem Kopffeld nimmt.
	immer := func(*http.Request) bool { return true }
	leitung := broker.Tuer(immer, func(*http.Request) string { return "" }, basis.SiehtLeserKennung)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.Claims{Rolle: auth.Rolle(r.Header.Get("X-Rolle")), KontoID: r.Header.Get("X-Rolle")}
		leitung(w, r.WithContext(context.WithValue(r.Context(), auth.ClaimsSchluessel, claims)))
	}))
	defer server.Close()

	oeffne := func(rolle auth.Rolle) *bufio.Reader {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("Anfrage: %v", err)
		}
		req.Header.Set("X-Rolle", string(rolle))
		antwort, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Verbindung als %s: %v", rolle, err)
		}
		t.Cleanup(func() {
			if err := antwort.Body.Close(); err != nil {
				t.Logf("Strom schließen: %v", err)
			}
		})
		return bufio.NewReader(antwort.Body)
	}
	// datenVon liest bis zur Meldung „action" und liefert ihre Datenzeile.
	datenVon := func(leser *bufio.Reader) string {
		t.Helper()
		gefunden := make(chan string, 1)
		go func() {
			for {
				zeile, err := leser.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(zeile) == "event: action" {
					daten, _ := leser.ReadString('\n') //nolint:errcheck // ein Abbruch zeigt sich an der leeren Zeile
					gefunden <- strings.TrimSpace(daten)
					return
				}
			}
		}()
		select {
		case daten := <-gefunden:
			return daten
		case <-time.After(3 * time.Second):
			t.Fatal("keine Meldung action innerhalb von 3 s")
			return ""
		}
	}

	stroeme := map[auth.Rolle]*bufio.Reader{}
	for _, rolle := range []auth.Rolle{auth.RolleKollegium, auth.RolleHelfer, auth.RolleMitarbeiter, auth.RolleAdmin} {
		stroeme[rolle] = oeffne(rolle)
	}
	// Die Meldung wiederholen, bis jeder Strom angemeldet und eingestuft ist: Eine frühere
	// erreichte einen Strom noch nicht oder noch in der knappen Fassung.
	ende := time.Now().Add(3 * time.Second)
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		for time.Now().Before(ende) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Millisecond):
				basis.SendeThekenEreignis("ausleihe", "leser-4711")
			}
		}
	}()

	for rolle, mitKennung := range map[auth.Rolle]bool{
		auth.RolleKollegium:   false,
		auth.RolleHelfer:      true,
		auth.RolleMitarbeiter: true,
		auth.RolleAdmin:       true,
	} {
		// Ein vertrauter Strom kann zuerst die knappe Fassung sehen (vor der Einstufung); gelesen
		// wird deshalb, bis die Kennung da ist. Ein Strom ohne Recht darf sie in keiner Meldung sehen.
		var gesehen bool
		versuche := 20
		if !mitKennung {
			versuche = 5
		}
		for versuch := 0; versuch < versuche && !gesehen; versuch++ {
			daten := datenVon(stroeme[rolle])
			if !strings.Contains(daten, `"event":"ausleihe"`) {
				t.Fatalf("%s: Meldung %q ohne die Art des Vorgangs", rolle, daten)
			}
			gesehen = strings.Contains(daten, "leser-4711")
		}
		if gesehen != mitKennung {
			t.Errorf("%s: Kennung des Lesers in der Meldung = %v, erwartet %v", rolle, gesehen, mitKennung)
		}
	}
	abbrechen()
	<-fertig
}
