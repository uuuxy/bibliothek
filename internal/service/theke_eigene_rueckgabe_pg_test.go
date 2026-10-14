package service_test

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// In offener Sitzung scannt die Theke ein Buch, das das Kind selbst ausgeliehen hat — das ist
// eine Rückgabe. Liefe die Sperrprüfung vor der Rückgabe-Erkennung, würde ein Kind, das wegen
// überfälliger Bücher gesperrt ist, mit genau diesen Büchern abgewiesen; weiter ginge es nur
// mit „Sperre übergehen", und das Protokoll hielte eine übergangene Sperre fest statt einer
// Rückgabe.
func TestTheke_EigeneRueckgabeTrotzSperre(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	kind := testdaten.LegeSchuelerAn(t, pool, "S-SPERRE-1", "Gesperrt", "9H1")
	testdaten.LegeExemplarAn(t, pool, testdaten.LegeTitelAn(t, pool, "Titel Sperre", 1), "B-SPERRE-1", true, "")
	testdaten.BucheAusleiheUeberDienst(t, pool, "B-SPERRE-1", kind, bearbeiter)

	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test: Sperre' WHERE id = $1`, kind); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}

	bookRepo := repository.NewKatalogAbfragen(pool)
	loanSvc := service.NewAusleihDienst(pool, repository.NewLeserAbfragen(pool), bookRepo,
		repository.NewAusleihAbfragen(pool), repository.NewProtokollAbfragen(pool))
	ex, err := bookRepo.FindeExemplar(ctx, "B-SPERRE-1")
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
	if _, err := loanSvc.BucheAmLeser(ctx, ex, &kind, bearbeiter, false); err != nil {
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
	ex, err = bookRepo.FindeExemplar(ctx, "B-SPERRE-1")
	if err != nil {
		t.Fatalf("Exemplar neu laden: %v", err)
	}
	if _, err := loanSvc.BucheAmLeser(ctx, ex, &kind, bearbeiter, false); !errors.Is(err, service.ErrGesperrt) {
		t.Errorf("neue Ausleihe an ein gesperrtes Kind: erwartet ErrGesperrt, bekam %v", err)
	}
}

// Der Zwilling dazu: Die Theke scannt in offener Sitzung ein Buch, das einem anderen Kind
// gehört. Das ist eine Fremdrückgabe — verarbeiteFremdrueckgabe gibt nur zurück, für das Kind
// in der Sitzung entsteht keine Ausleihe. Die Schranken der Ausleihe (Sperre, Ausleihlimit)
// hängen deshalb daran, dass eine Ausleihe entsteht: Sonst könnte, wer gesperrt oder am Limit
// ist, das Buch eines Mitschülers nicht abgeben, während dieselbe Rückgabe ohne offene Sitzung
// (NimmZurueck) durchgeht.
func TestTheke_FremdrueckgabeUnabhaengigVonSperreUndLimit(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	bookRepo := repository.NewKatalogAbfragen(pool)
	loanSvc := service.NewAusleihDienst(pool, repository.NewLeserAbfragen(pool), bookRepo,
		repository.NewAusleihAbfragen(pool), repository.NewProtokollAbfragen(pool))

	titel := testdaten.LegeTitelAn(t, pool, "Titel Fremdrueckgabe", 3)
	for _, bc := range []string{"B-FREMD-1", "B-FREMD-2", "B-FREMD-3"} {
		testdaten.LegeExemplarAn(t, pool, titel, bc, true, "")
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
		ex, err := bookRepo.FindeExemplar(ctx, barcode)
		if err != nil {
			t.Fatalf("Exemplar %s laden: %v", barcode, err)
		}
		vorher := uebergangen()

		lr, err := loanSvc.BucheAmLeser(ctx, ex, &inSitzung, bearbeiter, false)
		if err != nil {
			t.Fatalf("Fremdrückgabe von %s abgewiesen: %v", barcode, err)
		}
		if !lr.Fremdrueckgabe || lr.Art != "rueckgabe" {
			t.Errorf("erwartet Fremdrückgabe, bekam Type=%q Fremdrueckgabe=%v", lr.Art, lr.Fremdrueckgabe)
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
	eigentuemer1 := testdaten.LegeSchuelerAn(t, pool, "S-FREMD-EIG1", "Eigentuemer", "9H2")
	testdaten.BucheAusleiheUeberDienst(t, pool, "B-FREMD-1", eigentuemer1, bearbeiter)
	gesperrt := testdaten.LegeSchuelerAn(t, pool, "S-FREMD-SPERRE", "Gesperrt", "9H2")
	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test: Sperre' WHERE id = $1`, gesperrt); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}
	fremdrueckgabe(t, "B-FREMD-1", gesperrt, eigentuemer1)

	// 2. Das Kind an der Theke ist am Ausleihlimit — das Buch gehört einem anderen.
	eigentuemer2 := testdaten.LegeSchuelerAn(t, pool, "S-FREMD-EIG2", "Eigentuemer2", "9H2")
	testdaten.BucheAusleiheUeberDienst(t, pool, "B-FREMD-2", eigentuemer2, bearbeiter)
	amLimit := testdaten.LegeSchuelerAn(t, pool, "S-FREMD-LIMIT", "AmLimit", "9H2")
	testdaten.BucheAusleiheUeberDienst(t, pool, "B-FREMD-3", amLimit, bearbeiter)
	fremdrueckgabe(t, "B-FREMD-2", amLimit, eigentuemer2)
}
