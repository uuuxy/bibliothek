package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// titelIDVonExemplar liefert den titel_id eines Exemplars.
func titelIDVonExemplar(t *testing.T, pool *pgxpool.Pool, exID string) string {
	t.Helper()
	var titelID string
	if err := pool.QueryRow(context.Background(),
		`SELECT titel_id FROM buecher_exemplare WHERE id = $1`, exID).Scan(&titelID); err != nil {
		t.Fatalf("titel_id lesen: %v", err)
	}
	return titelID
}

// Bug 4 (Vormerkungs-Monopolisierung), DB-Seite: Der EXISTS-Join ausleihen→buecher_exemplare
// in Create muss eine aktive Eigen-Ausleihe am selben TITEL korrekt erkennen (ausleihen hängt
// am Exemplar, nicht am Titel). Nur ein echter DB-Test prüft, dass Join und titel_id-Filter
// tatsächlich greifen — pgxmock spielt nur nachgestellte Antworten zurück.
func TestVormerkungCreate_BlocksSelfBorrowedTitle_PG(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "Vorm", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	schueler := seedSchueler(t, pool, "V-1", "Mia", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, ex[0], schueler, bearbeiter)

	repo := NewVormerkungRepository(pool)

	ex2 := seedSignaturMitExemplaren(t, pool, "Vorm2", 1)
	titel2 := titelIDVonExemplar(t, pool, ex2[0])
	frisch := seedSchueler(t, pool, "V-3", "Zoe", "7a")

	// (1) Solange die Eigen-Ausleihe offen ist, darf keine Vormerkung auf denselben Titel entstehen.
	if _, err := repo.Create(ctx, titelID, "", schueler); !errors.Is(err, ErrTitelBereitsAusgeliehen) {
		t.Fatalf("aktive Eigen-Ausleihe: erwartet ErrTitelBereitsAusgeliehen, bekam %v", err)
	}

	// (2) Derselbe Schüler darf aber einen ANDEREN Titel vormerken (Prüft den titel_id Filter).
	if _, err := repo.Create(ctx, titel2, "", schueler); err != nil {
		t.Fatalf("Eigen-Ausleihe auf Titel 1 darf Vormerkung auf Titel 2 nicht blockieren: %v", err)
	}

	// (3) Ein ANDERER Schüler darf denselben Titel vormerken (Prüft den schueler_id Filter).
	if _, err := repo.Create(ctx, titelID, "", frisch); err != nil {
		t.Fatalf("Eigen-Ausleihe von Schüler A darf Vormerkung von Schüler B nicht blockieren: %v", err)
	}

	// (4) Nach der Rückgabe ist die Vormerkung auf den eigenen Titel wieder erlaubt.
	returnLoan(t, pool, loan)
	if _, err := repo.Create(ctx, titelID, "", schueler); err != nil {
		t.Fatalf("nach Rückgabe muss Vormerkung möglich sein, bekam: %v", err)
	}
}
