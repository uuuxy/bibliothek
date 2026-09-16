package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Nicht jeder Fehler beim endgültigen Löschen ist ein Konflikt.
//
// Fund (OFFEN.md 5.6): Der Handler beantwortete JEDEN Fehler des Purge mit 409. Ein
// Verbindungsabbruch, ein kaputter Constraint, ein Tippfehler in einer Abfrage — alles las
// sich wie „da ist noch etwas offen". Die Bibliothek sucht dann einen offenen Vorgang, den
// es nicht gibt, und der echte Fehler bleibt unsichtbar. Das ist die Bugklasse
// „Fehler-Kollaps" (docs/sweeps.md), nur in der anderen Richtung: nicht 500 für alles,
// sondern 409 für alles.
//
// Geprüft wird an der Quelle der Unterscheidung — den Sentinels. Was der Handler daraus
// macht (409 / 404 / 500), steht in einem einzigen switch daneben.
func TestPurge_UnterscheidetKonfliktVonServerfehler(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewAuditRepository(pool)

	var schueler string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ('S-PURGE-ART', 'Pur', 'Ge', '9a', 2029, 'schueler') RETURNING id`).Scan(&schueler); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE id = $1`, schueler); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	// (1) Nicht im Papierkorb → Blockade, also Konflikt.
	err := repo.PurgeStudent(ctx, schueler, "")
	if !errors.Is(err, repository.ErrLoeschenBlockiert) {
		t.Errorf("ein Datensatz außerhalb des Papierkorbs muss als Blockade kenntlich sein, bekam: %v", err)
	}

	// (2) Unbekannte Kennung → nicht gefunden, kein Konflikt. Die Bibliothek soll nicht
	// nach einer offenen Ausleihe suchen, wenn es die Person gar nicht gibt.
	err = repo.PurgeStudent(ctx, "00000000-0000-0000-0000-000000000000", "")
	if !errors.Is(err, repository.ErrLeserNichtGefunden) {
		t.Errorf("eine unbekannte Kennung muss als „nicht gefunden\" kenntlich sein, bekam: %v", err)
	}
	if errors.Is(err, repository.ErrLoeschenBlockiert) {
		t.Error("eine unbekannte Kennung wird als Blockade gemeldet — dann sucht jemand einen offenen Vorgang, den es nicht gibt")
	}

	// (3) Offene Ausleihe → Blockade. Die Gegenprobe zu (2): Hier IST etwas offen.
	if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = NOW() WHERE id = $1`, schueler); err != nil {
		t.Fatalf("in den Papierkorb legen: %v", err)
	}
	var titel, exemplar string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('Purge-Titel', 'X', 'Buch')
		RETURNING id`).Scan(&titel); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, 'B-PURGE-ART', false) RETURNING id`, titel).Scan(&exemplar); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, NOW() + INTERVAL '7 days')`, exemplar, schueler); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM ausleihen WHERE schueler_id = $1`, schueler); err != nil {
			t.Errorf("Aufräumen Ausleihe: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE id = $1`, exemplar); err != nil {
			t.Errorf("Aufräumen Exemplar: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titel); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
	})

	err = repo.PurgeStudent(ctx, schueler, "")
	if !errors.Is(err, repository.ErrLoeschenBlockiert) {
		t.Errorf("eine offene Ausleihe muss als Blockade gemeldet werden, bekam: %v", err)
	}
}
