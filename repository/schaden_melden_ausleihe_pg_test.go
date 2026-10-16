package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Bleibt das Formular der Schadensmeldung offen, während das Buch zurückgegeben und neu
// ausgeliehen wird, sondert „Melden" das jetzt verliehene Exemplar nicht aus.
func TestSchadenMelden_InzwischenNeuVerliehen(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewSchadensAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "RaceTest", 1)
	copyID := ex[0]
	schuelerA := legeSchuelerAn(t, pool, "RACE-A", "Anna", "7a")
	schuelerB := legeSchuelerAn(t, pool, "RACE-B", "Ben", "7b")
	bearbeiter := legeBearbeiterAn(t, pool)

	// Ausleihe 1 an Schüler A (die im Schadensformular referenzierte).
	alteLoan := legeAusleiheAn(t, pool, copyID, schuelerA, bearbeiter)
	// A gibt zurück, B leiht neu aus (die Ausleihe, die es zu schützen gilt).
	bucheRueckgabe(t, pool, alteLoan)
	legeAusleiheAn(t, pool, copyID, schuelerB, bearbeiter)

	// Der verspätete Klick auf „Schaden melden" nennt die alte Ausleihe.
	_, err := repo.Melde(ctx, copyID, alteLoan, bearbeiter, "Kaffeefleck", SchadensArtBeschaedigt, 5.0)
	if !errors.Is(err, ErrExemplarNeuVerliehen) {
		t.Fatalf("repo.Melde(…) = %v, erwartet ErrExemplarNeuVerliehen", err)
	}

	// Das Exemplar bleibt im Bestand, und die Ausleihe von B läuft weiter.
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

// Die Prüfung auf eine neue Ausleihe hält den gewöhnlichen Fall nicht auf: Ein Schaden an der
// eigenen, noch laufenden Ausleihe geht durch.
func TestSchadenMelden_Normalfall(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewSchadensAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "NormalTest", 1)
	copyID := ex[0]
	schueler := legeSchuelerAn(t, pool, "NORM-A", "Cora", "8a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, schueler, bearbeiter)

	schadensID, err := repo.Melde(ctx, copyID, loan, bearbeiter, "Riss im Einband", SchadensArtBeschaedigt, 3.0)
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
		t.Errorf("aussonderung_grund = %q, erwartet BESCHAEDIGUNG", grund)
	}
}

// Ein doppelt abgeschickter Klick auf „Schaden melden" mit derselben Ausleihe belastet den
// Schüler nicht zweimal: Der zweite Aufruf liefert den vorhandenen Schadensfall, und es
// entsteht genau einer. Die Sperre auf der Zeile der Ausleihe (FOR UPDATE) reiht auch zwei
// gleichzeitige Transaktionen hintereinander.
func TestSchadenMelden_Idempotent(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewSchadensAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "IdemTest", 1)
	copyID := ex[0]
	schueler := legeSchuelerAn(t, pool, "IDEM-A", "Dora", "9a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, schueler, bearbeiter)

	id1, err := repo.Melde(ctx, copyID, loan, bearbeiter, "Wasserschaden", SchadensArtBeschaedigt, 7.5)
	if err != nil {
		t.Fatalf("erster Report abgelehnt: %v", err)
	}
	id2, err := repo.Melde(ctx, copyID, loan, bearbeiter, "Wasserschaden", SchadensArtBeschaedigt, 7.5)
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
		t.Errorf("anzahl = %d, erwartet genau 1 Schadensfall für die Ausleihe (Doppelbelastung des Schülers)", anzahl)
	}
}

func legeSchuelerAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, 'Test', $3, 2030) RETURNING id`, barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Schüler %q anlegen: %v", barcode, err)
	}
	return id
}

func legeBearbeiterAn(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		 VALUES ('Bibliotheks', 'Kraft', 'dmg@example.org', 'mitarbeiter', true) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	return id
}

func legeAusleiheAn(t *testing.T, pool *pgxpool.Pool, copyID, schuelerID, bearbeiterID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, rueckgabe_frist)
		 VALUES ($1, $2, $3, CURRENT_DATE + 14) RETURNING id`, copyID, schuelerID, bearbeiterID).Scan(&id); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	return id
}

func bucheRueckgabe(t *testing.T, pool *pgxpool.Pool, ausleiheID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP WHERE id = $1`, ausleiheID); err != nil {
		t.Fatalf("Rückgabe buchen: %v", err)
	}
}

// Der Schadensfall wird dem Schuldner der Ausleihe zugeschrieben. Eine Kennung des Schuldners
// nimmt SchadensAbfragen.Melde nicht entgegen; dass die mitgeschickte schueler_id der Anfrage
// folgenlos bleibt, prüft api/schaden/schaden_schuldner_pg_test.go an der Route.
func TestSchadenMelden_SchuldnerAusAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewSchadensAbfragen(pool)

	ex := legeSignaturMitExemplarenAn(t, pool, "BindTest", 1)
	copyID := ex[0]
	echterSchuldner := legeSchuelerAn(t, pool, "BIND-A", "Anna", "7a")
	fremder := legeSchuelerAn(t, pool, "BIND-B", "Ben", "7b")
	bearbeiter := legeBearbeiterAn(t, pool)

	loan := legeAusleiheAn(t, pool, copyID, echterSchuldner, bearbeiter)

	schadensID, err := repo.Melde(ctx, copyID, loan, bearbeiter, "Riss", SchadensArtBeschaedigt, 4.0)
	if err != nil {
		t.Fatalf("Melde: %v", err)
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
// möglich (TestSchadenMelden_SetztAbholbereiteVormerkungZurueck).
func TestSchadenMelden_AusleiheInzwischenZurueck(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	copyID := legeSignaturMitExemplarenAn(t, pool, "ZurueckTest", 1)[0]
	schueler := legeSchuelerAn(t, pool, "ZUR-A", "Eda", "7a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, schueler, bearbeiter)
	bucheRueckgabe(t, pool, loan)

	_, err := NewSchadensAbfragen(pool).Melde(ctx, copyID, loan, bearbeiter,
		"nicht zurückgegeben", SchadensArtNichtZurueck, 12.0)
	if !errors.Is(err, ErrAusleiheInzwischenZurueck) {
		t.Fatalf("NewSchadensAbfragen(pool).Melde(…) = %v, erwartet ErrAusleiheInzwischenZurueck", err)
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
func TestSchadenMelden_ExemplarKommtAusDerAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "ExemplarTest", 2)
	geliehen, imRegal := ex[0], ex[1]
	schueler := legeSchuelerAn(t, pool, "EXA-A", "Finn", "8a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, geliehen, schueler, bearbeiter)

	schadensID, err := NewSchadensAbfragen(pool).Melde(ctx, imRegal, loan, bearbeiter,
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
