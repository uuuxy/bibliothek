package service

// Ein Ausweis oder Etikett von VOR dem 17.09.2026 wird weiter gelesen (OFFEN.md 9.2).
//
// Bis dahin druckte die Anwendung Code 39 MIT Prüfzeichen. Das Zeichen steht in den
// Strichcode-Daten: Unter der Karte steht „A-10003", das Lesegerät liefert „A-100037".
// Der Server suchte eine Nummer, die es nicht gibt — und weil ein unbekannter Scan nur
// eine leere Trefferliste erzeugt, sah es an der Theke aus, als täte der Scanner nichts.
//
// Gedruckt wird seither Code 128 ohne Prüfzeichen. Die Karten und Etiketten von vorher
// sind aber im Umlauf und sollen weiter funktionieren — das war von Anfang an gefordert.
//
// Geprüft am ECHTEN Service gegen echtes Postgres, nicht an der Zeichenrechnung: Dass
// Mod 43 stimmt, steht in pkg/code39. Hier geht es um die Frage, die nur der Live-Pfad
// beantwortet — kommt am Ende eine Ausleihe zustande, und findet ein alter Ausweis sein
// Kind?

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/code39"
	"bibliothek/repository"
)

func TestAlterAufdruckMitPruefzeichenWirdGelesen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var mitarbeiterID, schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Alt', 'Etikett', $1, 'mitarbeiter', true) RETURNING id
	`, "altetikett-"+suffix+"@schule.invalid").Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}

	// Der Ausweis trägt eine Nummer, wie sie gedruckt wurde — die Vorsilbe A-.
	ausweis := "A-" + suffix[len(suffix)-6:]
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Alte', 'Karte', '08C', 2031) RETURNING id
	`, ausweis).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Altetikett-Testband', 'Prüfer', 'Buch', false) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	buchNummer := "B-" + suffix[len(suffix)-6:]
	var exemplarID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 9.00) RETURNING id`, titelID, buchNummer).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE schueler_id = $1`,
			`DELETE FROM schueler WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, schuelerID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM benutzer WHERE id = $1`, mitarbeiterID); err != nil {
			t.Errorf("Aufräumen Mitarbeiter: %v", err)
		}
	})

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	loanSvc := NewLoanService(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	svc := NewOmniboxService(pool, studentRepo, bookRepo, userRepo, loanRepo, loanSvc, deviceSvc)

	// wieGedruckt hängt das Prüfzeichen an — genau das, was das Lesegerät liefert.
	wieGedruckt := func(t *testing.T, nummer string) string {
		t.Helper()
		zeichen, ok := code39.Pruefzeichen(nummer)
		if !ok {
			t.Fatalf("Pruefzeichen(%q) scheiterte", nummer)
		}
		gescannt := nummer + string(zeichen)
		if gescannt == nummer {
			t.Fatalf("kein Prüfzeichen an %q angehängt — der Test misst dann nichts", nummer)
		}
		return gescannt
	}

	t.Run("ein alter Schülerausweis findet sein Kind", func(t *testing.T) {
		gescannt := wieGedruckt(t, ausweis)

		res, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: gescannt, StaffID: mitarbeiterID})
		if err != nil {
			t.Fatalf("Scan von %q (auf der Karte steht %q): %v", gescannt, ausweis, err)
		}
		if res.Student == nil {
			t.Fatalf("Scan von %q lieferte keinen Leser (Type=%q) — auf der Karte steht %q, "+
				"und die Karte liegt seit Jahren in einer Schultasche",
				gescannt, res.Type, ausweis)
		}
		if res.Student.BarcodeID != ausweis {
			t.Errorf("Leser hat Ausweis %q, want %q", res.Student.BarcodeID, ausweis)
		}
	})

	t.Run("ein altes Buchetikett bucht eine Ausleihe", func(t *testing.T) {
		// Der Live-Pfad, nicht nur die Auflösung: Nach diesem Scan muss eine Ausleihe in
		// der Datenbank stehen. Ein „gefunden, aber nichts gebucht" wäre an der Theke
		// dasselbe Ärgernis wie vorher.
		gescannt := wieGedruckt(t, buchNummer)

		res, err := svc.ProcessQuery(ctx, OmniboxQuery{
			Query: gescannt, ActiveLeserID: &schuelerID, StaffID: mitarbeiterID,
		})
		if err != nil {
			t.Fatalf("Scan von %q (auf dem Etikett steht %q): %v", gescannt, buchNummer, err)
		}
		if res.Type != "ausleihe" {
			t.Fatalf("Type = %q, want „ausleihe\" — Scan %q, Etikett %q", res.Type, gescannt, buchNummer)
		}

		var offen int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`,
			exemplarID).Scan(&offen); err != nil {
			t.Fatalf("Ausleihen zählen: %v", err)
		}
		if offen != 1 {
			t.Errorf("%d offene Ausleihen, want 1 — der Scan hat nicht gebucht", offen)
		}
	})

	t.Run("eine Nummer, die es gar nicht gibt, bleibt unbekannt", func(t *testing.T) {
		// Die Nachsicht darf nicht dazu führen, dass irgendetwas irgendwie passt. Diese
		// Nummer trägt ein gültiges Prüfzeichen — und auch gekürzt gibt es sie nicht.
		gescannt := wieGedruckt(t, "B-999999999")

		res, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: gescannt, StaffID: mitarbeiterID})
		if err == nil && res.Book != nil {
			t.Errorf("Scan von %q lieferte ein Buch (%v) — es gibt weder die Nummer noch "+
				"ihre gekürzte Form", gescannt, res.Book.BarcodeID)
		}
	})

	t.Run("ein Scan OHNE Prüfzeichen bleibt unverändert", func(t *testing.T) {
		// Der Normalfall seit dem 17.09.2026: Code 128, kein Prüfzeichen. Er muss auf dem
		// direkten Weg gefunden werden — ohne dass die Nachsicht ihn anfasst.
		res, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: ausweis, StaffID: mitarbeiterID})
		if err != nil {
			t.Fatalf("Scan von %q: %v", ausweis, err)
		}
		if res.Student == nil || res.Student.BarcodeID != ausweis {
			t.Errorf("Scan von %q fand den Leser nicht (Type=%q)", ausweis, res.Type)
		}
	})
}
