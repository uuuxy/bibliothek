package theke

// Idempotenz hält auch nach dem Commit. Würde die Antwort erst nach der Arbeit gespeichert und
// reservierte nichts den Schlüssel vorher, fände eine zweite Anfrage mit demselben Schlüssel,
// die nach dem Commit der ersten und vor dem Speichern ihrer Antwort eintrifft, nichts in der
// Ablage und liefe neu: Das Buch läge schon beim Kind und würde zurückgenommen. Zwei
// Theken-Tabs, ein Scanner-Doppelklick oder ein Nachbuchen, das den abgebrochenen
// Online-Versand wiederholt, und die Ausleihe wäre weg. Dieser Test prüft die Form „nach dem
// Commit"; gleichzeitige Anfragen, bei denen die späteren auf die Reservierung der ersten
// warten, prüft frontend/e2e/theke-gleicher-schluessel.spec.js am Stack.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
	"bibliothek/sse"
)

// zwischenzeitlich lässt nach dem Commit der ersten Anfrage die zweite starten, bevor die
// erste ihre Antwort speichert.
type zwischenzeitlich struct {
	echt    *service.OmniboxDienst
	einmal  sync.Once
	zweite  func()
	fertig  chan struct{}
	gewesen bool
}

func (z *zwischenzeitlich) VerarbeiteEingabe(ctx context.Context, q service.OmniboxEingabe) (*service.OmniboxErgebnis, error) {
	res, err := z.echt.VerarbeiteEingabe(ctx, q)
	z.einmal.Do(func() {
		z.gewesen = true
		go func() {
			z.zweite()
			close(z.fertig)
		}()
		time.Sleep(300 * time.Millisecond) // die zweite Anfrage ist jetzt unterwegs
	})
	return res, err
}

func TestIdempotenz_ZweiteAnfrageNachCommitVorSpeichern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var mitarbeiterID, schuelerID, titelID string
	mitarbeiterID = testdaten.LegeKontoAn(t, pool, "Idem", "Potenz", "idempotenz-"+suffix+"@schule.invalid", "mitarbeiter", true)
	schuelerID = testdaten.LegeSchuelerMitNamenAn(t, pool, "S-"+suffix, "Anna", "Doppelt", "07B", 2031)
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Idempotenz-Testband', 'Prüfer', 'Buch', false) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	barcode := "B-IDEM-" + suffix
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 20.00)`, titelID, barcode); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	studentRepo := repository.NewLeserAbfragen(pool)
	bookRepo := repository.NewKatalogAbfragen(pool)
	loanRepo := repository.NewAusleihAbfragen(pool)
	auditRepo := repository.NewProtokollAbfragen(pool)
	loanSvc := service.NewAusleihDienst(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewGeraeteDienst(pool, studentRepo, loanRepo, auditRepo)
	echt := service.NewOmniboxDienst(pool, studentRepo, bookRepo, loanRepo, loanSvc, deviceSvc)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: sse.NewBroker()}}
	schluessel := "3f2504e0-4f89-11d3-9a0c-0305e82c" + suffix[len(suffix)-4:]
	anfrage := func() *http.Request {
		body := fmt.Sprintf(`{"query":%q,"active_leser_id":%q,"idempotency_key":%q}`, barcode, schuelerID, schluessel)
		req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
		return req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{Rolle: auth.Rolle("mitarbeiter"), KontoID: mitarbeiterID}))
	}

	zweite := httptest.NewRecorder()
	dazwischen := &zwischenzeitlich{echt: echt, fertig: make(chan struct{})}
	dazwischen.zweite = func() { srv.EingabeVerarbeiten(echt)(zweite, anfrage()) }

	erste := httptest.NewRecorder()
	srv.EingabeVerarbeiten(dazwischen)(erste, anfrage())
	select {
	case <-dazwischen.fertig:
	case <-time.After(10 * time.Second):
		t.Fatal("die zweite Anfrage ist nicht zurückgekommen")
	}
	if !dazwischen.gewesen {
		t.Fatal("die zweite Anfrage wurde nie gestartet — der Test misst nichts")
	}

	typ := func(w *httptest.ResponseRecorder) string {
		var resp eingabeAntwort
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			return "kein JSON: " + w.Body.String()
		}
		return resp.Art
	}
	if erste.Code != http.StatusOK || typ(erste) != "ausleihe" {
		t.Fatalf("erste Anfrage: Status %d, Typ %q, Body %s", erste.Code, typ(erste), erste.Body.String())
	}
	if zweite.Code != http.StatusOK || typ(zweite) != "ausleihe" {
		t.Errorf("zweite Anfrage mit demselben Schlüssel: Status %d, Typ %q — erwartet dieselbe Ausleihe, Body %s",
			zweite.Code, typ(zweite), zweite.Body.String())
	}
	var offen int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE e.barcode_id = $1 AND a.rueckgabe_am IS NULL`, barcode).Scan(&offen); err != nil {
		t.Fatalf("offene Ausleihen zählen: %v", err)
	}
	if offen != 1 {
		t.Errorf("das Buch soll beim Kind liegen (1 offene Ausleihe), sind %d", offen)
	}
}
