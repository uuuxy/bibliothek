package service

// Die Buch-Ausleihe an eine Lehrkraft (resolveTeacherBorrower) machte bis zum 15.09.2026 aus
// JEDEM Fehler „Aktives Lehrerprofil nicht gefunden" (404). Zwei Folgen: Ein Datenbank-
// Aussetzer sah aus wie ein Bedienfehler, und eine Lehrkraft ohne Ausweis (barcode_id NULL,
// im Stapel über active_teacher_id erreichbar) scheiterte am Scan der NULL-Spalte in einen
// string — ebenfalls als 404. Die Geräte-Seite (ladeAktiveLehrkraft, cc9e6c8c) unterscheidet
// seit dem 13.09.2026; seit dem 15.09.2026 lesen beide Wege dieselbe Abfrage.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

func TestResolveTeacherBorrower_LehrkraftOhneAusweis(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ohne', 'Ausweis', $1, 'kollegium', true) RETURNING id
	`, "ohne-ausweis-"+suffix+"@schule.invalid").Scan(&id); err != nil {
		t.Fatalf("Lehrkraft ohne Ausweis anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, id); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	svc := &defaultLoanService{pool: pool}
	chk, err := svc.resolveTeacherBorrower(ctx, id)
	if err != nil {
		t.Fatalf("aktive Lehrkraft ohne Ausweis soll Handapparat bekommen, bekam: %v", err)
	}
	if chk.borrowerType != "teacher" || chk.borrowerID != id || chk.teacher.BarcodeID != "" {
		t.Errorf("erwartete teacher/%s ohne Barcode, bekam %q/%q/%q", id, chk.borrowerType, chk.borrowerID, chk.teacher.BarcodeID)
	}
}
