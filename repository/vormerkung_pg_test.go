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

// Die Prüfung in LegeAn erkennt eine laufende eigene Ausleihe am selben Titel: ausleihen hängt
// am Exemplar, nicht am Titel, und nur an der Datenbank zeigt sich, dass der Join und der
// Filter auf titel_id greifen.
func TestVormerkungLegeAn_SperrtSelbstAusgeliehenenTitel_PG(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "Vorm", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	schueler := legeSchuelerAn(t, pool, "V-1", "Mia", "7a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, ex[0], schueler, bearbeiter)

	repo := NewVormerkungsAbfragen(pool)

	ex2 := legeSignaturMitExemplarenAn(t, pool, "Vorm2", 1)
	titel2 := titelIDVonExemplar(t, pool, ex2[0])
	frisch := legeSchuelerAn(t, pool, "V-3", "Zoe", "7a")

	// (1) Solange die Eigen-Ausleihe offen ist, darf keine Vormerkung auf denselben Titel entstehen.
	if _, err := repo.LegeAn(ctx, titelID, "", schueler); !errors.Is(err, ErrTitelBereitsAusgeliehen) {
		t.Fatalf("aktive Eigen-Ausleihe = %v, erwartet ErrTitelBereitsAusgeliehen", err)
	}

	// (2) Derselbe Schüler merkt einen anderen Titel vor (prüft den Filter auf titel_id).
	if _, err := repo.LegeAn(ctx, titel2, "", schueler); err != nil {
		t.Fatalf("Eigen-Ausleihe auf Titel 1 darf Vormerkung auf Titel 2 nicht blockieren: %v", err)
	}

	// (3) Ein anderer Schüler merkt denselben Titel vor (prüft den Filter auf schueler_id).
	if _, err := repo.LegeAn(ctx, titelID, "", frisch); err != nil {
		t.Fatalf("Eigen-Ausleihe von Schüler A darf Vormerkung von Schüler B nicht blockieren: %v", err)
	}

	// (4) Nach der Rückgabe ist die Vormerkung auf den eigenen Titel wieder erlaubt.
	bucheRueckgabe(t, pool, loan)
	if _, err := repo.LegeAn(ctx, titelID, "", schueler); err != nil {
		t.Fatalf("nach Rückgabe muss Vormerkung möglich sein, bekam: %v", err)
	}
}

// Eine abholbereite Vormerkung, deren Abholfrist abgelaufen ist, wird gelöscht (wer nicht
// abholt, verliert den Platz), und das Exemplar geht an den nächsten Wartenden, sofern es noch
// frei ist.
func TestLasseAbgelaufeneVerfallen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewVormerkungsAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "Verfall", 1)
	exID := ex[0]
	titelID := titelIDVonExemplar(t, pool, exID)
	noShow := legeSchuelerAn(t, pool, "VF-NOSHOW", "Nora", "7a")
	naechster := legeSchuelerAn(t, pool, "VF-NEXT", "Nils", "7b")

	// Nicht abgeholt: abholbereit, Frist gestern abgelaufen, hält exID.
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

	verfallen, neuBereit, err := repo.LasseAbgelaufeneVerfallen(ctx)
	if err != nil {
		t.Fatalf("Verfall: %v", err)
	}
	if verfallen != 1 || neuBereit != 1 {
		t.Fatalf("repo.LasseAbgelaufeneVerfallen(ctx) = %d/%d, erwartet 1 verfallen + 1 neu bereit", verfallen, neuBereit)
	}

	// Die nicht abgeholte Vormerkung ist weg.
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

// Ist das frei gewordene Exemplar inzwischen ausgeliehen, wird die nicht abgeholte Vormerkung
// trotzdem gelöscht, der Nächste aber nicht bedient; das erledigt die nächste Rückgabe.
func TestLasseAbgelaufeneVerfallen_ExemplarWeg(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewVormerkungsAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "VerfallWeg", 1)
	exID := ex[0]
	titelID := titelIDVonExemplar(t, pool, exID)
	noShow := legeSchuelerAn(t, pool, "VFW-NOSHOW", "Nora", "7a")
	naechster := legeSchuelerAn(t, pool, "VFW-NEXT", "Nils", "7b")
	borger := legeSchuelerAn(t, pool, "VFW-BORG", "Ben", "8c")
	bearbeiter := legeBearbeiterAn(t, pool)

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
	legeAusleiheAn(t, pool, exID, borger, bearbeiter)

	verfallen, neuBereit, err := repo.LasseAbgelaufeneVerfallen(ctx)
	if err != nil {
		t.Fatalf("Verfall: %v", err)
	}
	if verfallen != 1 || neuBereit != 0 {
		t.Fatalf("repo.LasseAbgelaufeneVerfallen(ctx) = %d/%d, erwartet 1 verfallen + 0 neu bereit (Exemplar weg)", verfallen, neuBereit)
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

// Dieselbe Vormerkung zweimal: Die Tabelle hält UNIQUE(titel_id, schueler_id). Der Fehler des
// Constraints ist kein Serverfehler, sondern die Antwort „steht schon auf der Liste"; als 500
// stünde an der Theke eine Störungsmeldung, und niemand wüsste, ob die erste Vormerkung noch
// gilt.
//
// Vormerkungen ohne Schüler bleiben davon unberührt: NULL kollidiert in UNIQUE nicht mit NULL.
func TestVormerkungLegeAn_ZweiteIstKonflikt_PG(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "VormDup", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	schueler := legeSchuelerAn(t, pool, "V-DUP", "Ida", "7a")
	repo := NewVormerkungsAbfragen(pool)

	if _, err := repo.LegeAn(ctx, titelID, "", schueler); err != nil {
		t.Fatalf("erste Vormerkung: %v", err)
	}
	_, err := repo.LegeAn(ctx, titelID, "", schueler)
	if !errors.Is(err, ErrVormerkungBereitsVorhanden) {
		t.Fatalf("zweite Vormerkung = %v, erwartet ErrVormerkungBereitsVorhanden", err)
	}

	// Gegenprobe: Die Vormerkung ohne Schüler entsteht mehrfach; die Theke merkt damit „ein
	// Exemplar für die Klasse 7b" mehrmals vor.
	if _, err := repo.LegeAn(ctx, titelID, "anonym 1", ""); err != nil {
		t.Fatalf("erste anonyme Vormerkung: %v", err)
	}
	if _, err := repo.LegeAn(ctx, titelID, "anonym 2", ""); err != nil {
		t.Fatalf("zweite anonyme Vormerkung darf nicht kollidieren: %v", err)
	}
}
