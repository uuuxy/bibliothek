package leser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Nach einer verlorenen Antwort fragt die Oberfläche mit der Vorschau nach, ob der Wechsel lief
// (Schuljahreswechsel.svelte). Läuft er da noch, wartet die Vorschau an seiner Sperre und sieht
// danach seinen Eintrag: 409 mit dem Lauf statt einer Vorschau unter „es wurde noch nichts
// geändert".
func TestVersetzung_VorschauWartetAufLaufendenWechsel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('VW', 'Admin', 'vw-admin@test.invalid', 'admin', true)
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE aktion = 'SCHULJAHRESWECHSEL'`)
	})

	// Der laufende Wechsel: Er hält die Sperre und hat seinen Eintrag geschrieben, committet aber
	// erst, wenn die Vorschau an der Sperre wartet.
	lauf, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Lauf beginnen: %v", err)
	}
	defer db.Verwerfe(ctx, lauf)
	if err := repository.SperreSchuljahreswechsel(ctx, lauf); err != nil {
		t.Fatalf("Sperre: %v", err)
	}
	if err := repository.SchreibeAdminProtokoll(ctx, lauf, adminID, repository.AktionSchuljahreswechsel, "",
		`{"versetzt": 0, "abgaenger": 0}`); err != nil {
		t.Fatalf("Eintrag des Laufs: %v", err)
	}

	codes := make(chan int, 1)
	go func() {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
		req := httptest.NewRequest(http.MethodPost, "/api/students/promote", strings.NewReader(`{"dry_run": true}`))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin}))
		w := httptest.NewRecorder()
		srv.Schuljahreswechsel()(w, req)
		codes <- w.Code
	}()

	frist := time.Now().Add(10 * time.Second)
	for wartet := false; !wartet; {
		select {
		case code := <-codes:
			t.Fatalf("Die Vorschau wartete nicht auf den laufenden Wechsel: Sie antwortete %d, bevor er committet hatte", code)
		default:
		}
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_locks l JOIN pg_stat_activity a USING (pid)
				WHERE l.locktype = 'advisory' AND NOT l.granted
				  AND a.datname = current_database() AND a.query LIKE '%pg_advisory_xact_lock%')`).Scan(&wartet); err != nil {
			t.Fatalf("Sperren lesen: %v", err)
		}
		if !wartet && time.Now().After(frist) {
			t.Fatal("Die Vorschau kam in 10 s weder an die Sperre noch zu einer Antwort")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := lauf.Commit(ctx); err != nil {
		t.Fatalf("Lauf committen: %v", err)
	}
	if code := <-codes; code != http.StatusConflict {
		t.Errorf("Vorschau nach dem Lauf = %d, erwartet 409", code)
	}
}
