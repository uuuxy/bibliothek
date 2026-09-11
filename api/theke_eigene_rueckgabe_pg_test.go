package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Register 10.09.2026 (Theke, B): In offener Sitzung scannt die Theke ein Buch, das das
// Kind selbst ausgeliehen hat — das ist eine Rückgabe. Die Sperrprüfung lief aber vor der
// Rückgabe-Erkennung: Ein Kind, das wegen überfälliger Bücher gesperrt ist, kam mit genau
// diesen Büchern und wurde abgewiesen. Weiter ging es nur mit „Sperre übergehen", und das
// Protokoll hielt eine übergangene Sperre fest statt einer Rückgabe.
func TestTheke_EigeneRueckgabeTrotzSperre(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := adminFuerAudit(t, pool)
	kind := schuelerAnlegen(t, pool, "Gesperrt", "9H1", "S-SPERRE-1")
	exemplar(t, pool, titelMitMeldebestand(t, pool, "Titel Sperre", 1), "B-SPERRE-1", true, "")
	ausleiheUeberDenDienst(t, pool, "B-SPERRE-1", kind, bearbeiter)

	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test: Sperre' WHERE id = $1`, kind); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}

	bookRepo := repository.NewBookRepository(pool)
	loanSvc := service.NewLoanService(pool, repository.NewStudentRepository(pool), bookRepo,
		repository.NewLoanRepository(pool), repository.NewAuditRepository(pool))
	ex, err := bookRepo.GetCopyByBarcode(ctx, "B-SPERRE-1")
	if err != nil {
		t.Fatalf("Exemplar laden: %v", err)
	}
	uebergangen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'OVERRIDE_BLOCK'`).Scan(&n); err != nil {
			t.Fatalf("Protokoll zählen: %v", err)
		}
		return n
	}
	vorher := uebergangen()

	// Die eigene Rückgabe geht durch — ohne Übergehen.
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, ex, &kind, nil, bearbeiter, false); err != nil {
		t.Fatalf("eigene Rückgabe eines gesperrten Kindes abgewiesen: %v", err)
	}
	var offen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, ex.ID).Scan(&offen); err != nil {
		t.Fatal(err)
	}
	if offen != 0 {
		t.Errorf("nach der Rückgabe steht die Ausleihe noch offen (%d)", offen)
	}
	if n := uebergangen(); n != vorher {
		t.Errorf("die Rückgabe steht als übergangene Sperre im Protokoll (%d neue Einträge)", n-vorher)
	}

	// Gegenprobe: Eine neue Ausleihe an dasselbe Kind bleibt gesperrt.
	ex, err = bookRepo.GetCopyByBarcode(ctx, "B-SPERRE-1")
	if err != nil {
		t.Fatalf("Exemplar neu laden: %v", err)
	}
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, ex, &kind, nil, bearbeiter, false); !errors.Is(err, service.ErrBlocked) {
		t.Errorf("neue Ausleihe an ein gesperrtes Kind: erwartet ErrBlocked, bekam %v", err)
	}
}
