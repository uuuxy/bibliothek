package service

// Eine Lehrkraft OHNE Ausweisnummer muss ausleihen können: Bis ein Ausweis gedruckt ist,
// steht sie an der Theke über die Namenssuche zur Verfügung, und ihre Kennung geht als
// aktiver Leser in die Buchung.
//
// Bis zum 15.09.2026 machte die Buch-Ausleihe an eine Lehrkraft aus JEDEM Fehler „Aktives
// Lehrerprofil nicht gefunden" (404). Zwei Folgen: Ein Datenbank-Aussetzer sah aus wie ein
// Bedienfehler, und eine Lehrkraft ohne Ausweis scheiterte am Scan der NULL-Spalte in einen
// string — ebenfalls als 404. Seit Migration 125 liest der Ausleihpfad die Leserzeile
// (GetLeserByID) und damit dieselbe Quelle wie die Theke; die NULL-Spalte kommt als
// coalesce, und die Frist entscheidet die Art.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

func TestResolveBorrower_LehrkraftOhneAusweis(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// Seit Migration 136 bekommt ein aktives Konto eine Nummer, seit 145 behält es eine.
	// Ohne Nummer ist eine Lehrkraft seitdem nur ohne aktives Konto — etwa ohne Schul-Adresse,
	// solange die Akte keine nachgetragen hat. Der Ausleihpfad muss den Zustand tragen.
	var leserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (vorname, nachname, art) VALUES ('Ohne', $1, 'lehrkraft') RETURNING id
	`, "Ausweis-"+suffix).Scan(&leserID); err != nil {
		t.Fatalf("Lehrkraft ohne Ausweis anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen (Leser): %v", err)
		}
	})

	svc := &defaultLoanService{pool: pool, studentRepo: repository.NewStudentRepository(pool)}
	chk, err := svc.resolveBorrowerAndDueTime(ctx, &repository.BookCopy{}, &leserID)
	if err != nil {
		t.Fatalf("eine aktive Lehrkraft ohne Ausweis muss ausleihen können, bekam: %v", err)
	}
	if chk.borrowerID != leserID || chk.leser == nil || chk.leser.BarcodeID != "" {
		t.Fatalf("erwartet Leser %s ohne Ausweisnummer, bekam %q/%+v", leserID, chk.borrowerID, chk.leser)
	}
	if chk.istSchueler() {
		t.Errorf("eine Lehrkraft ist kein Schüler, Art gelesen: %q", chk.leser.Art)
	}
	// Ohne Klasse gibt es keine Schuljahresfrist: Nicht-Schüler bekommen ein Jahr.
	if chk.dueTime.Before(time.Now().AddDate(0, 11, 0)) {
		t.Errorf("Frist %s liegt zu früh — erwartet rund ein Jahr", chk.dueTime)
	}
}
