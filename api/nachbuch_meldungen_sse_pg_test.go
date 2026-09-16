package api

// Das Live-Signal der Meldungsliste (Schritt C des Offline-Baus, OFFEN.md Abschnitt 2).
//
// Die Liste hängt am Band jedes Arbeitsplatzes. Ohne Signal zeigte der zweite Arbeitsplatz
// seine Zahl bis zur nächsten Anmeldung weiter — bei einer Theke mit zwei Rechnern heißt
// das: Einer quittiert, der andere sieht die erledigte Meldung den ganzen Tag.
//
// Gemeldet wird an der WIRKUNG, nicht an einer Liste der Ergebnisse, die eine Meldung
// erzeugen: Welche das sind, weiß der Dienst; eine zweite Liste im Handler stimmte genau
// bis zur nächsten Meldung, die jemand hinzufügt. Deshalb zählt der Handler vorher und
// nachher — und genau das prüft dieser Test: Ein Nachbuchen OHNE Abweichung schweigt, eines
// MIT Abweichung meldet sich.

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/google/uuid"
)

func TestNachbuchMeldungen_SSESignalBeiAenderung(t *testing.T) {
	w := nbTuerAufbau(t)

	broker := sse.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	w.srv.Broker = broker

	stream := httptest.NewServer(broker.Handler())
	resp, err := http.Get(stream.URL)
	if err != nil {
		cancel()
		stream.Close()
		t.Fatalf("SSE-Strom nicht erreichbar: %v", err)
	}
	// Reihenfolge wie in settings_sse_test.go: erst den Client-Body schließen, dann den
	// Broker beenden — sonst wartet httptest.Server.Close auf eine Verbindung, die noch
	// auf Ereignisse horcht, und der Test hängt bis zum Paket-Timeout.
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
	naechstes := func(was string) string {
		t.Helper()
		select {
		case e := <-ereignisse:
			return e
		case <-time.After(3 * time.Second):
			t.Fatalf("kein SSE-Ereignis innerhalb von 3 s (%s)", was)
			return ""
		}
	}
	// Auf der Leitung liegt mehr als diese eine Sorte: Der Online-Scan meldet `action`.
	// Gewartet wird deshalb AUF das Ereignis, nicht auf das nächste.
	warteAuf := func(was string) {
		t.Helper()
		for i := 0; i < 8; i++ {
			if naechstes(was) == "nachbuch-meldungen" {
				return
			}
		}
		t.Fatalf("kein Ereignis nachbuch-meldungen (%s)", was)
	}
	// Kein Ereignis dieser Sorte in der kurzen Frist. Ein langsamer Broker macht das hier
	// grün, nie rot — die Zusage „wir melden nur bei einer Änderung" ist damit nicht
	// bewiesen, aber ein Dauersender fiele auf.
	stillFuer := func(dauer time.Duration, was string) {
		t.Helper()
		ende := time.After(dauer)
		for {
			select {
			case e := <-ereignisse:
				if e == "nachbuch-meldungen" {
					t.Fatalf("Ereignis nachbuch-meldungen, obwohl sich nichts geändert hat (%s)", was)
				}
			case <-ende:
				return
			}
		}
	}
	if e := naechstes("Handshake"); e != "connected" {
		t.Fatalf("Handshake fehlt: erwartet connected, war %q", e)
	}

	// Anna hat das Buch (online gebucht). Der Offline-Scan schreibt es Ben zu: Das ist die
	// Umbuchung — beim Vorbesitzer zurückgenommen, neu ausgeliehen — und genau die legt
	// eine Meldung an.
	if code, _ := w.onlineScan(t, uuid.NewString(), w.anna); code != http.StatusOK {
		t.Fatalf("Online-Ausleihe an Anna: Status %d", code)
	}
	schluessel := uuid.NewString()
	w.nachbuchenMit(t, w.nachbuch, w.eintrag(schluessel, "ausleihe", &w.ben, time.Now()))

	warteAuf("Umbuchung")

	// Quittieren ist die zweite Änderung: eine Meldung weniger.
	liste, err := repository.ListeNachbuchMeldungen(context.Background(), w.pool, true)
	if err != nil || len(liste) == 0 {
		t.Fatalf("Meldung nicht gefunden (err=%v, %d Zeilen)", err, len(liste))
	}
	rec := httptest.NewRecorder()
	req := w.sitzung(httptest.NewRequest(http.MethodPost,
		"/api/action/nachbuch-meldungen/"+liste[0].ID+"/quittieren", nil))
	req.SetPathValue("id", liste[0].ID)
	w.srv.NachbuchMeldungQuittierenHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Quittieren: Status %d, %s", rec.Code, rec.Body.String())
	}

	warteAuf("Quittieren")

	// Und die Gegenprobe: Ein Nachbuchen, das nichts ändert (derselbe Schlüssel noch
	// einmal), meldet sich nicht. Sonst holte jeder Arbeitsplatz bei jedem Sync eine Zahl,
	// die dieselbe ist.
	w.nachbuchenMit(t, w.nachbuch, w.eintrag(schluessel, "ausleihe", &w.ben, time.Now()))
	stillFuer(500*time.Millisecond, "zweites Nachbuchen mit demselben Schlüssel")
}
