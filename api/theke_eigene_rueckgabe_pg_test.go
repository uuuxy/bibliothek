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

// Der Zwilling dazu (Raster-Durchgang 12.09.2026 über die Änderungen vom 11.09.): Die
// Theke scannt in offener Sitzung ein Buch, das einem ANDEREN Kind gehört. Das ist eine
// Fremdrückgabe — handleForeignReturn gibt nur zurück, für das Kind in der Sitzung
// entsteht KEINE Ausleihe. Die Schranken der Ausleihe (Sperre, Ausleihlimit) liefen
// trotzdem, weil sie an `!isReturningThis` hingen statt an „es entsteht eine Ausleihe":
// Wer gesperrt oder am Limit war, konnte das Buch eines Mitschülers nicht abgeben,
// während dieselbe Rückgabe ohne offene Sitzung (HandleSimpleReturn) durchging.
func TestTheke_FremdrueckgabeUnabhaengigVonSperreUndLimit(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := adminFuerAudit(t, pool)
	bookRepo := repository.NewBookRepository(pool)
	loanSvc := service.NewLoanService(pool, repository.NewStudentRepository(pool), bookRepo,
		repository.NewLoanRepository(pool), repository.NewAuditRepository(pool))

	titel := titelMitMeldebestand(t, pool, "Titel Fremdrueckgabe", 3)
	for _, bc := range []string{"B-FREMD-1", "B-FREMD-2", "B-FREMD-3"} {
		exemplar(t, pool, titel, bc, true, "")
	}

	// Das Ausleihlimit auf 1 — sonst bräuchte der zweite Fall fünf Exemplare.
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('max_ausleihen_schueler', '1')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Ausleihlimit setzen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel = 'max_ausleihen_schueler'`); err != nil {
			t.Errorf("Ausleihlimit zurücksetzen: %v", err)
		}
	})

	uebergangen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'OVERRIDE_BLOCK'`).Scan(&n); err != nil {
			t.Fatalf("Protokoll zählen: %v", err)
		}
		return n
	}

	// fremdrueckgabe scannt das Buch des Eigentümers, während `inSitzung` an der Theke steht.
	fremdrueckgabe := func(t *testing.T, barcode, inSitzung, eigentuemer string) {
		t.Helper()
		ex, err := bookRepo.GetCopyByBarcode(ctx, barcode)
		if err != nil {
			t.Fatalf("Exemplar %s laden: %v", barcode, err)
		}
		vorher := uebergangen()

		lr, err := loanSvc.HandleUnifiedCheckout(ctx, ex, &inSitzung, nil, bearbeiter, false)
		if err != nil {
			t.Fatalf("Fremdrückgabe von %s abgewiesen: %v", barcode, err)
		}
		if !lr.Fremdrueckgabe || lr.Type != "rueckgabe" {
			t.Errorf("erwartet Fremdrückgabe, bekam Type=%q Fremdrueckgabe=%v", lr.Type, lr.Fremdrueckgabe)
		}
		var offen int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, ex.ID).Scan(&offen); err != nil {
			t.Fatal(err)
		}
		if offen != 0 {
			t.Errorf("nach der Fremdrückgabe steht die Ausleihe des Eigentümers noch offen (%d)", offen)
		}
		// Das Buch darf dabei nicht an das Kind an der Theke gewandert sein.
		var uebernommen int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND schueler_id = $2`,
			ex.ID, inSitzung).Scan(&uebernommen); err != nil {
			t.Fatal(err)
		}
		if uebernommen != 0 {
			t.Errorf("die Fremdrückgabe hat das Buch dem Kind in der Sitzung angehängt (%d Ausleihen)", uebernommen)
		}
		if n := uebergangen(); n != vorher {
			t.Errorf("die Fremdrückgabe steht als übergangene Sperre im Protokoll (%d neue Einträge)", n-vorher)
		}
	}

	// 1. Das Kind an der Theke ist gesperrt — das Buch gehört einem anderen.
	eigentuemer1 := schuelerAnlegen(t, pool, "Eigentuemer", "9H2", "S-FREMD-EIG1")
	ausleiheUeberDenDienst(t, pool, "B-FREMD-1", eigentuemer1, bearbeiter)
	gesperrt := schuelerAnlegen(t, pool, "Gesperrt", "9H2", "S-FREMD-SPERRE")
	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test: Sperre' WHERE id = $1`, gesperrt); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}
	fremdrueckgabe(t, "B-FREMD-1", gesperrt, eigentuemer1)

	// 2. Das Kind an der Theke ist am Ausleihlimit — das Buch gehört einem anderen.
	eigentuemer2 := schuelerAnlegen(t, pool, "Eigentuemer2", "9H2", "S-FREMD-EIG2")
	ausleiheUeberDenDienst(t, pool, "B-FREMD-2", eigentuemer2, bearbeiter)
	amLimit := schuelerAnlegen(t, pool, "AmLimit", "9H2", "S-FREMD-LIMIT")
	ausleiheUeberDenDienst(t, pool, "B-FREMD-3", amLimit, bearbeiter)
	fremdrueckgabe(t, "B-FREMD-2", amLimit, eigentuemer2)
}
