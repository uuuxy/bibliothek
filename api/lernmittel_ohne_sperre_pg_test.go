package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Antwort der Schule vom 22.09.2026 (docs/OFFEN.md 9.3 c): Die Lernmittelfreiheit in Hessen
// lässt keine automatische Sperre zu — auch keine, die jemand übergehen kann. Ein Kind mit
// unbezahlter Forderung bekommt sein Schulbuch ohne Override; dasselbe Kind wird beim Buch
// der Schülerbücherei weiter abgewiesen. Beide Fälle laufen über den Live-Pfad der Theke
// (HandleUnifiedCheckout), nicht über die Prüffunktion allein: Nur so ist belegt, dass das
// Exemplar an der Weiche ankommt. Rot gesehen am Rückbau der Weiche (22.09.2026).
func TestTheke_LernmittelTrotzOffenerForderung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := adminFuerAudit(t, pool)
	kind := schuelerAnlegen(t, pool, "Forderung", "9H1", "S-LMF-SPERRE-1")

	// Die Forderung hängt an einem verlorenen Buch der Schülerbücherei.
	buecherei := titelMitMeldebestand(t, pool, "Roman ohne Kennung", 1)
	verloren := exemplar(t, pool, buecherei, "B-LMF-SPERRE-0", true, "")
	ausleiheUeberDenDienst(t, pool, "B-LMF-SPERRE-0", kind, bearbeiter)
	if _, err := pool.Exec(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Verloren', 12.00, 'nicht_zurueckgegeben')`, verloren, kind); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}

	var lernmittel string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('Mathe 9', true) RETURNING id`).Scan(&lernmittel); err != nil {
		t.Fatalf("Lernmittel-Titel anlegen: %v", err)
	}
	exemplar(t, pool, lernmittel, "B-LMF-SPERRE-1", true, "")
	exemplar(t, pool, buecherei, "B-LMF-SPERRE-2", true, "")

	bookRepo := repository.NewBookRepository(pool)
	loanSvc := service.NewLoanService(pool, repository.NewStudentRepository(pool), bookRepo,
		repository.NewLoanRepository(pool), repository.NewAuditRepository(pool))
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

	// Das Lernmittel geht raus — ohne Override, ohne Protokolleintrag.
	schulbuch, err := bookRepo.GetCopyByBarcode(ctx, "B-LMF-SPERRE-1")
	if err != nil {
		t.Fatalf("Lernmittel laden: %v", err)
	}
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, schulbuch, &kind, bearbeiter, false); err != nil {
		t.Fatalf("Lernmittel trotz offener Forderung abgewiesen: %v", err)
	}
	var offen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, schulbuch.ID).Scan(&offen); err != nil {
		t.Fatal(err)
	}
	if offen != 1 {
		t.Errorf("das Lernmittel steht nicht als Ausleihe da (%d)", offen)
	}
	if n := uebergangen(); n != vorher {
		t.Errorf("die Ausgabe des Lernmittels steht als übergangene Sperre im Protokoll (%d neue Einträge)", n-vorher)
	}

	// Gegenprobe: Das Buch der Schülerbücherei bleibt hinter der Forderung.
	roman, err := bookRepo.GetCopyByBarcode(ctx, "B-LMF-SPERRE-2")
	if err != nil {
		t.Fatalf("Roman laden: %v", err)
	}
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, roman, &kind, bearbeiter, false); !errors.Is(err, service.ErrBlocked) {
		t.Errorf("Schülerbücherei bei offener Forderung: erwartet ErrBlocked, bekam %v", err)
	}
}
