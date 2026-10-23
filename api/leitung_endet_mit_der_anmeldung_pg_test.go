package api

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stromEnde meldet leitungsStrom, wenn der Server den Strom beendet hat.
const stromEnde = "Ende des Stroms"

// Ein Mithörer ohne Browser öffnet die Live-Leitung mit einem kopierten Token. Vor jedem
// Lebenszeichen prüft sie die Anmeldung und endet, sobald diese abgemeldet oder gesperrt ist; ein
// Aussetzer der Datenbank beendet sie nicht. Geprüft am ganzen Router mit dem Takt des Betriebs:
// Die Ströme laufen nebeneinander, der Test wartet einen Takt von 15 Sekunden.
func TestLeitung_EndetMitDerAnmeldung(t *testing.T) {
	pool := pgtest.Pool(t)
	kontoID := testdaten.LegeAdminKontoAn(t, pool, "leitung-anmeldung@example.org", "Lea", "Leitung")
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM sitzungen WHERE benutzer_id = $1`, kontoID)
	})

	aussetzer := &leitungsAussetzer{Pool: pool}
	ungestoert := baueLeitungsServer(t, pool, pool)
	gestoert := baueLeitungsServer(t, pool, aussetzer)

	bleibt := ungestoert.oeffne(t, "gültige Anmeldung", kontoID)
	abgemeldet := ungestoert.oeffne(t, "abgemeldet", kontoID)
	gesperrt := ungestoert.oeffne(t, "gesperrt", kontoID)
	ohneDatenbank := gestoert.oeffne(t, "Aussetzer der Datenbank", kontoID)

	// Abgemeldet wird über die Tür der Oberfläche, mit dem Token, das der Mithörer kopiert hat.
	abmelden := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	abmelden.AddCookie(&http.Cookie{Name: "session_token", Value: abgemeldet.token})
	rec := httptest.NewRecorder()
	ungestoert.router.ServeHTTP(rec, abmelden)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/logout: Status %d: %s", rec.Code, rec.Body.String())
	}
	// Gesperrt nach Inaktivität in einem anderen Fenster: dieselbe Zeile, die POST /api/auth/sperren
	// schreibt.
	if _, err := repository.SperreSitzung(t.Context(), pool, gesperrt.sitzungID); err != nil {
		t.Fatalf("Anmeldung sperren: %v", err)
	}
	aussetzer.an.Store(true)

	bleibt.erwarte(t, "event: ping")
	abgemeldet.erwarte(t, stromEnde)
	gesperrt.erwarte(t, stromEnde)
	ohneDatenbank.erwarte(t, "event: ping")
	if aussetzer.getroffen.Load() == 0 {
		t.Error("die Prüfung vor dem Lebenszeichen las die Anmeldung nicht; der Aussetzer blieb ungeprüft")
	}
}

// leitungsAussetzer lässt das Lesen der Anmeldung scheitern, solange an gesetzt ist, wie bei
// einem Aussetzer der Datenbank.
type leitungsAussetzer struct {
	*pgxpool.Pool
	an        atomic.Bool
	getroffen atomic.Int32
}

func (p *leitungsAussetzer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if p.an.Load() && strings.Contains(sql, "FROM sitzungen") {
		p.getroffen.Add(1)
		return aussetzerZeile{}
	}
	return p.Pool.QueryRow(ctx, sql, args...)
}

type aussetzerZeile struct{}

func (aussetzerZeile) Scan(...any) error {
	return errors.New("testfehler: verbindung zur datenbank abgerissen")
}

// leitungsServer ist der ganze Router mit eigener Anmeldung und eigenem Broker.
type leitungsServer struct {
	pool      *pgxpool.Pool
	anmeldung *auth.Anmeldung
	router    http.Handler
	server    *httptest.Server
}

// baueLeitungsServer baut den Router; die Anmeldung liest über anmeldungsPool.
func baueLeitungsServer(t *testing.T, pool *pgxpool.Pool, anmeldungsPool repository.Datenbank) *leitungsServer {
	t.Helper()
	anmeldung, err := auth.NewAnmeldung("pg-test-sitzungsgeheimnis-32-zeichen!!", anmeldungsPool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	router := NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), ServerKonfig{}).Routes()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return &leitungsServer{pool: pool, anmeldung: anmeldung, router: router, server: server}
}

// leitungsStrom ist eine offene Live-Leitung. Eine eigene Goroutine liest sie und meldet jedes
// Ereignis und das Ende.
type leitungsStrom struct {
	name, token, sitzungID string
	meldungen              <-chan string
}

// oeffne legt eine Anmeldung an und öffnet mit ihrem Token die Live-Leitung, wie es ein Mithörer
// ohne Browser täte.
func (l *leitungsServer) oeffne(t *testing.T, name, kontoID string) *leitungsStrom {
	t.Helper()
	sitzungID, err := repository.LegeSitzungAn(t.Context(), l.pool, kontoID, "", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("%s: Anmeldung anlegen: %v", name, err)
	}
	token, err := l.anmeldung.ErzeugeToken(kontoID, auth.RolleAdmin, sitzungID)
	if err != nil {
		t.Fatalf("%s: Token: %v", name, err)
	}

	ctx, abbrechen := context.WithCancel(context.Background())
	t.Cleanup(abbrechen)
	anfrage, err := http.NewRequestWithContext(ctx, http.MethodGet, l.server.URL+"/events", nil)
	if err != nil {
		t.Fatalf("%s: Anfrage bauen: %v", name, err)
	}
	anfrage.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	antwort, err := l.server.Client().Do(anfrage)
	if err != nil {
		t.Fatalf("%s: GET /events: %v", name, err)
	}
	t.Cleanup(func() {
		if err := antwort.Body.Close(); err != nil {
			t.Logf("%s: Strom schließen: %v", name, err)
		}
	})
	if antwort.StatusCode != http.StatusOK {
		t.Fatalf("%s: GET /events: Status %d", name, antwort.StatusCode)
	}

	meldungen := make(chan string, 8)
	go func() {
		defer close(meldungen)
		leser := bufio.NewReader(antwort.Body)
		for {
			zeile, err := leser.ReadString('\n')
			if errors.Is(err, io.EOF) {
				meldungen <- stromEnde
				return
			}
			if err != nil {
				meldungen <- "Lesefehler: " + err.Error()
				return
			}
			if ereignis := strings.TrimSpace(zeile); strings.HasPrefix(ereignis, "event: ") {
				meldungen <- ereignis
			}
		}
	}()

	strom := &leitungsStrom{name: name, token: token, sitzungID: sitzungID, meldungen: meldungen}
	strom.erwarte(t, "event: connected")
	return strom
}

// erwarte liest die nächste Meldung des Stroms. Ein Takt dauert 15 Sekunden, 20 lassen der
// Prüfung davor Zeit.
func (s *leitungsStrom) erwarte(t *testing.T, erwartet string) {
	t.Helper()
	select {
	case meldung := <-s.meldungen:
		if meldung != erwartet {
			t.Errorf("%s: nächste Meldung %q, erwartet %q", s.name, meldung, erwartet)
		}
	case <-time.After(20 * time.Second):
		t.Errorf("%s: nach 20 Sekunden keine Meldung, erwartet %q", s.name, erwartet)
	}
}
