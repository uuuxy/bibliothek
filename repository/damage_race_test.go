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

	ex := seedSignaturMitExemplaren(t, pool, "RaceTest", 1)
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
	_, err := repo.ReportDamage(ctx, copyID, alteLoan, bearbeiter, "Kaffeefleck", SchadensArtBeschaedigt, 5.0)
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

	ex := seedSignaturMitExemplaren(t, pool, "NormalTest", 1)
	copyID := ex[0]
	schueler := seedSchueler(t, pool, "NORM-A", "Cora", "8a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)

	schadensID, err := repo.ReportDamage(ctx, copyID, loan, bearbeiter, "Riss im Einband", SchadensArtBeschaedigt, 3.0)
	if err != nil {
		t.Fatalf("regulärer Schaden abgelehnt: %v", err)
	}
	if schadensID == "" {
		t.Error("keine Schadens-ID zurückgegeben")
	}
	if ausgesondert := ausgesonderteZahl(t, pool, []string{copyID}); ausgesondert != 1 {
		t.Error("Exemplar wurde nicht ausgesondert")
	}
	if grund := aussonderungsGrund(t, pool, copyID); grund != "BESCHAEDIGUNG" {
		t.Errorf("aussonderung_grund = %q, want BESCHAEDIGUNG", grund)
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

	ex := seedSignaturMitExemplaren(t, pool, "IdemTest", 1)
	copyID := ex[0]
	schueler := seedSchueler(t, pool, "IDEM-A", "Dora", "9a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)

	id1, err := repo.ReportDamage(ctx, copyID, loan, bearbeiter, "Wasserschaden", SchadensArtBeschaedigt, 7.5)
	if err != nil {
		t.Fatalf("erster Report abgelehnt: %v", err)
	}
	id2, err := repo.ReportDamage(ctx, copyID, loan, bearbeiter, "Wasserschaden", SchadensArtBeschaedigt, 7.5)
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
		`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		 VALUES ('Bibliotheks', 'Kraft', 'dmg@example.org', 'mitarbeiter', true) RETURNING id`).Scan(&id); err != nil {
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

// TestReportDamageSchuldnerAusAusleihe belegt die Objektbindung: Der Schadensfall wird dem
// Schuldner der Ausleihe zugeschrieben. Eine Kennung des Schuldners nimmt ReportDamage nicht
// entgegen; dass die mitgeschickte schueler_id der Anfrage folgenlos bleibt, prüft
// api/schaden_schuldner_pg_test.go an der Route.
func TestReportDamageSchuldnerAusAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewDamageRepository(pool)

	ex := seedSignaturMitExemplaren(t, pool, "BindTest", 1)
	copyID := ex[0]
	echterSchuldner := seedSchueler(t, pool, "BIND-A", "Anna", "7a")
	fremder := seedSchueler(t, pool, "BIND-B", "Ben", "7b")
	bearbeiter := seedBearbeiter(t, pool)

	loan := seedAusleihe(t, pool, copyID, echterSchuldner, bearbeiter)

	schadensID, err := repo.ReportDamage(ctx, copyID, loan, bearbeiter, "Riss", SchadensArtBeschaedigt, 4.0)
	if err != nil {
		t.Fatalf("ReportDamage: %v", err)
	}

	var gebucht string
	if err := pool.QueryRow(ctx,
		`SELECT schueler_id::text FROM schadensfaelle WHERE id = $1`, schadensID).Scan(&gebucht); err != nil {
		t.Fatalf("Schadensfall lesen: %v", err)
	}
	if gebucht != echterSchuldner {
		t.Errorf("Schaden muss dem Ausleiher (%s) angelastet werden, nicht einem anderen Schüler (%s) — war %s",
			echterSchuldner, fremder, gebucht)
	}
}

// Bleibt die Akte offen, während das Buch an einem anderen Platz zurückkommt, steht die
// Ausleihe dort weiter in der Liste. „Nicht zurückgegeben" auf diese Zeile darf das Buch, das
// im Regal steht, nicht aussondern und dem Kind keine Forderung anhängen. Der Weg über den
// Bescheid prüft dasselbe (bucheVerluste). „Beschädigt zurückgegeben" bleibt nach der Rückgabe
// möglich (TestReportDamage_ResetsAbholbereiteVormerkung).
func TestReportDamage_AusleiheInzwischenZurueck(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	copyID := seedSignaturMitExemplaren(t, pool, "ZurueckTest", 1)[0]
	schueler := seedSchueler(t, pool, "ZUR-A", "Eda", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)
	returnLoan(t, pool, loan)

	_, err := NewDamageRepository(pool).ReportDamage(ctx, copyID, loan, bearbeiter,
		"nicht zurückgegeben", SchadensArtNichtZurueck, 12.0)
	if !errors.Is(err, ErrAusleiheInzwischenZurueck) {
		t.Fatalf("erwartet ErrAusleiheInzwischenZurueck, war: %v", err)
	}
	if ausgesondert := ausgesonderteZahl(t, pool, []string{copyID}); ausgesondert != 0 {
		t.Error("das zurückgegebene Exemplar wurde ausgesondert")
	}
	var forderungen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schadensfaelle WHERE ausleihe_id = $1`, loan).Scan(&forderungen); err != nil {
		t.Fatal(err)
	}
	if forderungen != 0 {
		t.Errorf("%d Forderung(en) für ein zurückgegebenes Buch", forderungen)
	}
}

// Das Exemplar steht an der Ausleihe wie der Schuldner. Nennt die Anfrage ein anderes, trifft
// die Meldung trotzdem das geliehene Buch und lässt das genannte in Ruhe.
func TestReportDamage_ExemplarKommtAusDerAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "ExemplarTest", 2)
	geliehen, imRegal := ex[0], ex[1]
	schueler := seedSchueler(t, pool, "EXA-A", "Finn", "8a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, geliehen, schueler, bearbeiter)

	schadensID, err := NewDamageRepository(pool).ReportDamage(ctx, imRegal, loan, bearbeiter,
		"Wasserschaden", SchadensArtBeschaedigt, 6.0)
	if err != nil {
		t.Fatalf("Meldung abgelehnt: %v", err)
	}
	if ausgesondert := ausgesonderteZahl(t, pool, []string{imRegal}); ausgesondert != 0 {
		t.Error("das Exemplar aus der Anfrage wurde ausgesondert, es war nie verliehen")
	}
	if ausgesondert := ausgesonderteZahl(t, pool, []string{geliehen}); ausgesondert != 1 {
		t.Error("das geliehene Exemplar wurde nicht ausgesondert")
	}
	var amFall string
	if err := pool.QueryRow(ctx,
		`SELECT exemplar_id::text FROM schadensfaelle WHERE id = $1`, schadensID).Scan(&amFall); err != nil {
		t.Fatal(err)
	}
	if amFall != geliehen {
		t.Errorf("die Forderung nennt Exemplar %s, geliehen war %s", amFall, geliehen)
	}
}
