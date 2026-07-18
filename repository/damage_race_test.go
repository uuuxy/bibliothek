package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestReportDamageRace sichert den Race-Schutz ab: Bleibt das Schadensformular offen,
// während das Buch zurückgegeben und neu ausgeliehen wird, darf der "Melden"-Klick das
// jetzt aktiv verliehene Exemplar nicht aussondern.
func TestReportDamageRace(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewDamageRepository(pool)

	_, ex := seedSignaturMitExemplaren(t, pool, "RaceTest", 1)
	copyID := ex[0]
	schuelerA := seedSchueler(t, pool, "RACE-A", "Anna", "7a")
	schuelerB := seedSchueler(t, pool, "RACE-B", "Ben", "7b")
	bearbeiter := seedBearbeiter(t, pool)

	// Ausleihe 1 an Schüler A (die im Schadensformular referenzierte).
	alteLoan := seedAusleihe(t, pool, copyID, schuelerA, bearbeiter)
	// A gibt zurück, B leiht neu aus (die Ausleihe, die es zu schützen gilt).
	returnLoan(t, pool, alteLoan)
	seedAusleihe(t, pool, copyID, schuelerB, bearbeiter)

	// Jetzt kommt der verspätete "Schaden melden"-Klick mit der ALTEN loanID.
	_, err := repo.ReportDamage(ctx, copyID, alteLoan, schuelerA, bearbeiter, "Kaffeefleck", 5.0)
	if !errors.Is(err, ErrExemplarNeuVerliehen) {
		t.Fatalf("erwartet ErrExemplarNeuVerliehen, war: %v", err)
	}

	// Das Exemplar darf NICHT ausgesondert sein — B's Ausleihe bleibt intakt.
	if ausgesondert := ausgesonderteZahl(t, pool, []string{copyID}); ausgesondert != 0 {
		t.Error("Exemplar wurde trotz neuer Ausleihe ausgesondert")
	}
	var aktiveLoans int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, copyID).Scan(&aktiveLoans); err != nil {
		t.Fatal(err)
	}
	if aktiveLoans != 1 {
		t.Errorf("B's aktive Ausleihe: erwartet 1, war %d", aktiveLoans)
	}
}

// TestReportDamageNormalfall stellt sicher, dass der Guard den regulären Fall nicht
// blockiert: Ein Schaden an der eigenen, noch aktiven Ausleihe geht durch.
func TestReportDamageNormalfall(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewDamageRepository(pool)

	_, ex := seedSignaturMitExemplaren(t, pool, "NormalTest", 1)
	copyID := ex[0]
	schueler := seedSchueler(t, pool, "NORM-A", "Cora", "8a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)

	schadensID, err := repo.ReportDamage(ctx, copyID, loan, schueler, bearbeiter, "Riss im Einband", 3.0)
	if err != nil {
		t.Fatalf("regulärer Schaden abgelehnt: %v", err)
	}
	if schadensID == "" {
		t.Error("keine Schadens-ID zurückgegeben")
	}
	if ausgesondert := ausgesonderteZahl(t, pool, []string{copyID}); ausgesondert != 1 {
		t.Error("Exemplar wurde nicht ausgesondert")
	}
}

// TestReportDamageIdempotent sichert #4 (Doppelte Rechnungen) ab: Ein doppelt abgeschickter
// "Schaden melden"-Klick mit derselben ausleihe_id darf den Schüler nicht zweimal belasten.
// Der zweite Aufruf muss den bereits angelegten Schadensfall idempotent zurückgeben; es darf
// genau EIN Schadensfall entstehen. Der FOR-UPDATE-Lock auf der Ausleihe-Zeile serialisiert
// dabei auch die echt parallele Variante (zwei Transaktionen gleichzeitig).
func TestReportDamageIdempotent(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewDamageRepository(pool)

	_, ex := seedSignaturMitExemplaren(t, pool, "IdemTest", 1)
	copyID := ex[0]
	schueler := seedSchueler(t, pool, "IDEM-A", "Dora", "9a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)

	id1, err := repo.ReportDamage(ctx, copyID, loan, schueler, bearbeiter, "Wasserschaden", 7.5)
	if err != nil {
		t.Fatalf("erster Report abgelehnt: %v", err)
	}
	id2, err := repo.ReportDamage(ctx, copyID, loan, schueler, bearbeiter, "Wasserschaden", 7.5)
	if err != nil {
		t.Fatalf("zweiter Report (Doppelklick) abgelehnt: %v", err)
	}
	if id1 != id2 {
		t.Errorf("Doppelklick erzeugte einen zweiten Schadensfall: %q vs %q", id1, id2)
	}

	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schadensfaelle WHERE ausleihe_id = $1`, loan).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("erwartet genau 1 Schadensfall für die Ausleihe, waren %d (Doppelbelastung des Schülers)", anzahl)
	}
}

func seedSchueler(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, 'Test', $3, 2030) RETURNING id`, barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Schüler %q anlegen: %v", barcode, err)
	}
	return id
}

func seedBearbeiter(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		 VALUES ('DMG-B', 'Bibliotheks', 'Kraft', 'dmg@example.org', 'mitarbeiter', true) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	return id
}

func seedAusleihe(t *testing.T, pool *pgxpool.Pool, copyID, schuelerID, bearbeiterID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, rueckgabe_frist)
		 VALUES ($1, $2, $3, CURRENT_DATE + 14) RETURNING id`, copyID, schuelerID, bearbeiterID).Scan(&id); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	return id
}

func returnLoan(t *testing.T, pool *pgxpool.Pool, loanID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP WHERE id = $1`, loanID); err != nil {
		t.Fatalf("Rückgabe buchen: %v", err)
	}
}
