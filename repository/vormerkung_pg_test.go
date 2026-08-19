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

// TestVerfalleAbgelaufeneVormerkungen belegt die Betreiber-Entscheidung (19.08.2026):
// Eine „abholbereit"-Reservierung, deren Abholfrist abgelaufen ist, wird gelöscht (der
// No-Show verliert den Platz) und das Exemplar dem nächsten Wartenden zugeteilt — sofern
// es noch frei ist. Ohne diesen Lauf blieb der No-Show für immer stecken.
func TestVerfalleAbgelaufeneVormerkungen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewVormerkungRepository(pool)

	ex := seedSignaturMitExemplaren(t, pool, "Verfall", 1)
	exID := ex[0]
	titelID := titelIDVonExemplar(t, pool, exID)
	noShow := seedSchueler(t, pool, "VF-NOSHOW", "Nora", "7a")
	naechster := seedSchueler(t, pool, "VF-NEXT", "Nils", "7b")

	// No-Show: abholbereit, Frist GESTERN abgelaufen, hält exID.
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
		VALUES ($1,$2,'abholbereit',$3, CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '5 days')`,
		titelID, noShow, exID); err != nil {
		t.Fatalf("No-Show-Vormerkung: %v", err)
	}
	// Nächster: wartend am selben Titel, älter als ein etwaiger dritter.
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
		VALUES ($1,$2,'wartend', CURRENT_TIMESTAMP - INTERVAL '2 days')`, titelID, naechster); err != nil {
		t.Fatalf("Warte-Vormerkung: %v", err)
	}

	verfallen, neuBereit, err := repo.VerfalleAbgelaufeneVormerkungen(ctx)
	if err != nil {
		t.Fatalf("Verfall: %v", err)
	}
	if verfallen != 1 || neuBereit != 1 {
		t.Fatalf("erwartet 1 verfallen + 1 neu bereit, war %d/%d", verfallen, neuBereit)
	}

	// No-Show ist weg.
	var noShowDa int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE schueler_id=$1`, noShow).Scan(&noShowDa); err != nil {
		t.Fatal(err)
	}
	if noShowDa != 0 {
		t.Error("abgelaufene No-Show-Vormerkung wurde nicht gelöscht")
	}
	// Nächster ist jetzt abholbereit mit exID.
	var status, exemplar string
	if err := pool.QueryRow(ctx,
		`SELECT status, COALESCE(bereitgestellt_exemplar_id::text,'') FROM vormerkungen WHERE schueler_id=$1`,
		naechster).Scan(&status, &exemplar); err != nil {
		t.Fatal(err)
	}
	if status != "abholbereit" || exemplar != exID {
		t.Errorf("nächster Wartender nicht bedient: status=%q exemplar=%q (erwartet abholbereit/%s)", status, exemplar, exID)
	}
}

// TestVerfalleAbgelaufeneVormerkungen_ExemplarWeg: Ist das freigewordene Exemplar
// zwischenzeitlich ausgeliehen, wird der No-Show trotzdem abgeräumt, der nächste aber
// NICHT bedient (das Exemplar ist ja weg) — der reguläre Rückgabe-Pfad erledigt das später.
func TestVerfalleAbgelaufeneVormerkungen_ExemplarWeg(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewVormerkungRepository(pool)

	ex := seedSignaturMitExemplaren(t, pool, "VerfallWeg", 1)
	exID := ex[0]
	titelID := titelIDVonExemplar(t, pool, exID)
	noShow := seedSchueler(t, pool, "VFW-NOSHOW", "Nora", "7a")
	naechster := seedSchueler(t, pool, "VFW-NEXT", "Nils", "7b")
	borger := seedSchueler(t, pool, "VFW-BORG", "Ben", "8c")
	bearbeiter := seedBearbeiter(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
		VALUES ($1,$2,'abholbereit',$3, CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '5 days')`,
		titelID, noShow, exID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
		VALUES ($1,$2,'wartend', CURRENT_TIMESTAMP - INTERVAL '2 days')`, titelID, naechster); err != nil {
		t.Fatal(err)
	}
	// Das Exemplar ist inzwischen an einen Dritten ausgeliehen.
	seedAusleihe(t, pool, exID, borger, bearbeiter)

	verfallen, neuBereit, err := repo.VerfalleAbgelaufeneVormerkungen(ctx)
	if err != nil {
		t.Fatalf("Verfall: %v", err)
	}
	if verfallen != 1 || neuBereit != 0 {
		t.Fatalf("erwartet 1 verfallen + 0 neu bereit (Exemplar weg), war %d/%d", verfallen, neuBereit)
	}
	// Nächster bleibt wartend.
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM vormerkungen WHERE schueler_id=$1`, naechster).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "wartend" {
		t.Errorf("nächster Wartender darf ohne freies Exemplar nicht bedient werden, war %q", status)
	}
}
