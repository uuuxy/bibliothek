package api

// Idempotenz hält (OFFEN.md 2.2, Commit 7). Bis zum 15.09.2026 wurde die Antwort erst NACH der
// Arbeit gespeichert, mit dem Request-Context; nichts reservierte den Schlüssel vorher. Eine
// zweite Anfrage mit demselben Schlüssel, die NACH dem Commit der ersten und VOR dem Speichern
// ihrer Antwort eintraf, fand keinen Cache und lief neu: Das Buch lag schon beim Kind, also
// wurde es zurückgenommen. Zwei Theken-Tabs, ein Scanner-Doppelklick oder ein Nachbuchen, das
// den abgebrochenen Online-Versand wiederholt — und die Ausleihe war weg. (Die gleichzeitige
// Form fängt der Unique-Index; nur die Form „nach dem Commit" ist offen.)

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

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"
	"bibliothek/sse"
)

// zwischenzeitlich lässt nach dem Commit der ersten Anfrage die zweite starten, bevor die
// erste ihre Antwort speichert.
type zwischenzeitlich struct {
	echt    service.OmniboxService
	einmal  sync.Once
	zweite  func()
	fertig  chan struct{}
	gewesen bool
}

func (z *zwischenzeitlich) ProcessQuery(ctx context.Context, q service.OmniboxQuery) (*service.OmniboxResult, error) {
	res, err := z.echt.ProcessQuery(ctx, q)
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var mitarbeiterID, schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Idem', 'Potenz', $2, 'mitarbeiter', true) RETURNING id
	`, "MA-"+suffix, "idempotenz-"+suffix+"@schule.invalid").Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Anna', 'Doppelt', '07B', 2031) RETURNING id
	`, "S-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
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

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	loanSvc := service.NewLoanService(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	echt := service.NewOmniboxService(pool, studentRepo, bookRepo, repository.NewUserRepository(pool), loanRepo, loanSvc, deviceSvc)

	srv := &Server{DB: &db.Database{Pool: pool}, Broker: sse.NewBroker()}
	schluessel := "3f2504e0-4f89-11d3-9a0c-0305e82c" + suffix[len(suffix)-4:]
	anfrage := func() *http.Request {
		body := fmt.Sprintf(`{"query":%q,"active_student_id":%q,"idempotency_key":%q}`, barcode, schuelerID, schluessel)
		req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
		return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{Rolle: auth.Role("mitarbeiter"), UserID: mitarbeiterID}))
	}

	zweite := httptest.NewRecorder()
	dazwischen := &zwischenzeitlich{echt: echt, fertig: make(chan struct{})}
	dazwischen.zweite = func() { srv.ActionHandler(echt)(zweite, anfrage()) }

	erste := httptest.NewRecorder()
	srv.ActionHandler(dazwischen)(erste, anfrage())
	select {
	case <-dazwischen.fertig:
	case <-time.After(10 * time.Second):
		t.Fatal("die zweite Anfrage ist nicht zurückgekommen")
	}
	if !dazwischen.gewesen {
		t.Fatal("die zweite Anfrage wurde nie gestartet — der Test misst nichts")
	}

	typ := func(w *httptest.ResponseRecorder) string {
		var resp ActionResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			return "kein JSON: " + w.Body.String()
		}
		return resp.Type
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
