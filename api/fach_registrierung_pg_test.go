package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
)

// TestFachAutoRegistrierung belegt die Betriebsregel hinter Migration 078: subject ist
// FK auf systematik_kategorien(bezeichnung), und die Schreibpfade registrieren
// unbekannte Fächer SELBST (inventur.StelleFaecherSicher) — die Lehre aus Migration
// 021→060, wo eine FK-Spalte starb, weil nur die Migration sie je befüllte. Ein Import
// mit neuem Fach darf nicht scheitern, eine Schreibvariante ("deutsch") muss auf die
// kanonische Registrierung ("Deutsch") laufen statt eine Dublette anzulegen.
func TestFachAutoRegistrierung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := inventur.NewBookRepository(pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	fach := "Astronomie" + suffix
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM buecher_titel WHERE titel LIKE $1`, "%"+suffix)
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE bezeichnung ILIKE $1`, "%"+suffix)
	})

	// 1) Unbekanntes Fach: CreateBook registriert es und der Titel trägt es.
	id, err := repo.CreateBook(ctx, inventur.Book{Title: "Sternbuch-" + suffix, ISBN: "ISBN-A-" + suffix, Subject: fach})
	if err != nil {
		t.Fatalf("CreateBook mit unbekanntem Fach: %v", err)
	}
	var registriert int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM systematik_kategorien WHERE bezeichnung = $1`, fach).Scan(&registriert); err != nil {
		t.Fatalf("Registrierung prüfen: %v", err)
	}
	if registriert != 1 {
		t.Fatalf("Fach wurde nicht auto-registriert (count=%d)", registriert)
	}

	// 2) Schreibvariante: läuft auf die kanonische Form, KEINE zweite Sachgruppe.
	id2, err := repo.CreateBook(ctx, inventur.Book{Title: "Sternbuch2-" + suffix, ISBN: "ISBN-B-" + suffix, Subject: strings.ToLower(fach)})
	if err != nil {
		t.Fatalf("CreateBook mit Schreibvariante: %v", err)
	}
	var subject2 string
	if err := pool.QueryRow(ctx,
		`SELECT subject FROM buecher_titel WHERE id = $1::uuid`, id2).Scan(&subject2); err != nil {
		t.Fatalf("subject lesen: %v", err)
	}
	if subject2 != fach {
		t.Errorf("Schreibvariante nicht kanonisiert: subject = %q, erwartet %q", subject2, fach)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM systematik_kategorien WHERE lower(bezeichnung) = lower($1)`, fach).Scan(&registriert); err != nil {
		t.Fatalf("Dubletten prüfen: %v", err)
	}
	if registriert != 1 {
		t.Errorf("Schreibvariante hat eine Sachgruppen-Dublette erzeugt (count=%d)", registriert)
	}

	// 3) Leeres Fach bleibt NULL (der FK gilt nur für Nicht-NULL-Werte).
	if err := repo.UpdateBookCategory(ctx, id, "", 5); err != nil {
		t.Fatalf("UpdateBookCategory mit leerem Fach: %v", err)
	}
	var istNull bool
	if err := pool.QueryRow(ctx,
		`SELECT subject IS NULL FROM buecher_titel WHERE id = $1::uuid`, id).Scan(&istNull); err != nil {
		t.Fatalf("NULL prüfen: %v", err)
	}
	if !istNull {
		t.Error("leeres Fach muss als NULL gespeichert werden")
	}

	// 4) Kürzel-Kollision: Ein Fach, dessen Leerzeichen-freier Kandidat schon als
	// Kürzel vergeben ist, weicht auf ein Hash-Suffix aus, statt zu scheitern.
	besetzt := "Chemie" + suffix
	if _, err := pool.Exec(ctx,
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ($1, $2)`,
		besetzt, "Anderes Fach "+suffix); err != nil {
		t.Fatalf("Kürzel besetzen: %v", err)
	}
	if _, err := repo.CreateBook(ctx, inventur.Book{Title: "Chemiebuch-" + suffix, ISBN: "ISBN-C-" + suffix, Subject: besetzt}); err != nil {
		t.Fatalf("CreateBook trotz Kürzel-Kollision: %v", err)
	}
	var hashKuerzel string
	if err := pool.QueryRow(ctx,
		`SELECT kuerzel FROM systematik_kategorien WHERE bezeichnung = $1`, besetzt).Scan(&hashKuerzel); err != nil {
		t.Fatalf("Hash-Kürzel lesen: %v", err)
	}
	if !strings.Contains(hashKuerzel, "~") {
		t.Errorf("erwartet Hash-Suffix im Ausweich-Kürzel, war %q", hashKuerzel)
	}
}

// TestSystematikRenameZiehtOffeneInventurMit sichert den Anschluss der Inventur an die
// Umbenennung: scope_subject offener Sessions ist Text, kein FK — bliebe er auf dem
// alten Namen, zählte die Session null Exemplare und jeder Scan liefe auf 409.
// Abgeschlossene Sessions sind Historie und behalten den alten Namen.
func TestSystematikRenameZiehtOffeneInventurMit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	alt, neu := "Physik"+suffix, "Astrophysik"+suffix
	_, id := systematikAnlegen(t, srv, "Phy"+suffix, alt)
	if id == "" {
		t.Fatal("Sachgruppe konnte nicht angelegt werden")
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM inventur_sessions WHERE scope_label LIKE $1`, "%"+suffix)
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
	})

	var offenID, zuID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type, scope_subject, scope_label)
		VALUES ('filter', $1, 'Fach offen ' || $2) RETURNING id`, alt, suffix).Scan(&offenID); err != nil {
		t.Fatalf("offene Session anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type, scope_subject, scope_label, abgeschlossen_am)
		VALUES ('filter', $1, 'Fach zu ' || $2, now()) RETURNING id`, alt, suffix).Scan(&zuID); err != nil {
		t.Fatalf("abgeschlossene Session anlegen: %v", err)
	}

	koerper, err := json.Marshal(map[string]string{"kuerzel": "Phy" + suffix, "bezeichnung": neu})
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/systematics/"+id, bytes.NewReader(koerper))
	req.SetPathValue("id", id)
	srv.UpdateSystematikHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Umbenennen: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}

	var offenScope, zuScope string
	if err := pool.QueryRow(ctx,
		`SELECT scope_subject FROM inventur_sessions WHERE id = $1`, offenID).Scan(&offenScope); err != nil {
		t.Fatalf("offene Session lesen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT scope_subject FROM inventur_sessions WHERE id = $1`, zuID).Scan(&zuScope); err != nil {
		t.Fatalf("abgeschlossene Session lesen: %v", err)
	}
	if offenScope != neu {
		t.Errorf("offene Session muss dem neuen Fachnamen folgen, war %q", offenScope)
	}
	if zuScope != alt {
		t.Errorf("abgeschlossene Session ist Historie und behält den alten Namen, war %q", zuScope)
	}
}
