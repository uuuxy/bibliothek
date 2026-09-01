package service

// Abholfach-Hinweis beim Schüler-Scan (Betreiber-Entscheidung 01.09.2026):
// Schüler scannen nicht selbst — beim Scan des Ausweises durch die Mitarbeiterin
// muss die Antwort die abholbereiten Vormerkungen des Schülers tragen, damit
// das Buch aus dem Abholfach direkt mitgegeben wird, statt dort die 3-Tage-Frist
// abzuwarten. Echtes Postgres über den ECHTEN Service (NewOmniboxService mit
// echten Repos): geprüft wird der Live-Pfad des Scans, nicht ein Stub.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

func TestSchuelerScanTraegtAbholfachHinweis(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Abholtest', 'Fachgreifer', '05A', 2031) RETURNING id
	`, "ABH-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('Abholfach-Testband', 'Prüfer', 'Buch') RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_bis)
		VALUES ($1, $2, 'abholbereit', now() + interval '3 days')
	`, titelID, schuelerID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	// Zweite, noch WARTENDE Vormerkung — sie darf im Hinweis NICHT auftauchen,
	// sonst schickt die Theke jemanden ans Abholfach, in dem nichts liegt.
	var wartenderTitelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('Wartend-Testband', 'Prüfer', 'Buch') RETURNING id
	`).Scan(&wartenderTitelID); err != nil {
		t.Fatalf("zweiten Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')
	`, wartenderTitelID, schuelerID); err != nil {
		t.Fatalf("wartende Vormerkung anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM vormerkungen WHERE schueler_id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Vormerkungen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id IN ($1, $2)`, titelID, wartenderTitelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Schüler: %v", err)
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

	res, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: "ABH-" + suffix})
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "student" || res.Student == nil {
		t.Fatalf("erwartet Schüler-Antwort, war Typ %q", res.Type)
	}
	if len(res.Abholbereit) != 1 {
		t.Fatalf("erwartet genau 1 Abholfach-Hinweis (die wartende Vormerkung zählt nicht), waren %d: %+v",
			len(res.Abholbereit), res.Abholbereit)
	}
	if h := res.Abholbereit[0]; h.Titel != "Abholfach-Testband" || h.BereitgestelltBis == nil {
		t.Errorf("Hinweis unvollständig: %+v", h)
	}
}
