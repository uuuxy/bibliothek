package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration 116: Jede Bewegung eines Exemplars stempelt letzte_bewegung_am, jede Ausleihe
// trägt erfasst_am. Der Wächter des Nachbuchens (Stufe 2, Commit 11) verlässt sich darauf:
// Ein Scan, der älter ist als die letzte Bewegung, wird gemeldet statt gebucht. Fehlt der
// Stempel an einem Schreiber, hält der Wächter dort still — und bucht einen alten Scan über
// eine jüngere Wirklichkeit. Geprüft wird darum jeder Schreiber einzeln, am echten SQL.
func TestBewegungsstempel_JederSchreiberSetztIhn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	loans := NewLoanRepository(pool)

	stempel := func() *time.Time {
		var s *time.Time
		if err := pool.QueryRow(ctx, `SELECT letzte_bewegung_am FROM buecher_exemplare WHERE id = $1`, f.exemplarID).Scan(&s); err != nil {
			t.Fatalf("Stempel lesen: %v", err)
		}
		return s
	}
	if s := stempel(); s != nil {
		t.Fatalf("vor der ersten Bewegung muss der Stempel NULL sein, ist %v", s)
	}

	// Ausleihe: Stempel gesetzt, erfasst_am an der Ausleihe.
	tx := beginne(t, pool)
	loan, err := loans.CreateLoanTx(ctx, tx, f.exemplarID, f.schuelerID, f.bearbeiterID, time.Now().AddDate(0, 0, 14), false)
	if err != nil {
		t.Fatalf("ausleihen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	nachAusleihe := stempel()
	if nachAusleihe == nil {
		t.Fatal("Ausleihe setzt letzte_bewegung_am nicht")
	}
	var erfasst *time.Time
	if err := pool.QueryRow(ctx, `SELECT erfasst_am FROM ausleihen WHERE id = $1`, loan.ID).Scan(&erfasst); err != nil || erfasst == nil {
		t.Fatalf("erfasst_am an der Ausleihe fehlt: %v", err)
	}

	// Rückgabe: Stempel rückt vor.
	warte(t, pool)
	tx = beginne(t, pool)
	if err := loans.ReturnLoanTx(ctx, tx, loan.ID, f.bearbeiterID, false); err != nil {
		t.Fatalf("zurückgeben: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	nachRueckgabe := stempel()
	if nachRueckgabe == nil || !nachRueckgabe.After(*nachAusleihe) {
		t.Fatalf("Rückgabe rückt den Stempel nicht vor: %v → %v", nachAusleihe, nachRueckgabe)
	}

	// Aussonderung (Soft-Delete): Stempel rückt vor.
	warte(t, pool)
	if err := NewBookRepository(pool).DecommissionCopy(ctx, f.exemplarID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	nachAussonderung := stempel()
	if nachAussonderung == nil || !nachAussonderung.After(*nachRueckgabe) {
		t.Fatalf("Aussonderung rückt den Stempel nicht vor: %v → %v", nachRueckgabe, nachAussonderung)
	}

	// Rückholen: Stempel rückt vor.
	warte(t, pool)
	tx = beginne(t, pool)
	if _, err := HoleExemplarZurueck(ctx, tx, f.exemplarID, f.bearbeiterID, nil); err != nil {
		t.Fatalf("zurückholen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	nachRueckholen := stempel()
	if nachRueckholen == nil || !nachRueckholen.After(*nachAussonderung) {
		t.Fatalf("Rückholen rückt den Stempel nicht vor: %v → %v", nachAussonderung, nachRueckholen)
	}
}

// Der Stempel läuft nie rückwärts. Der Wächter des Nachbuchens weist Scans ab, die älter sind als
// die letzte Bewegung; setzt ein Schreiber den Stempel auf einen früheren Zeitpunkt zurück, gilt
// jeder Scan zwischen diesem Zeitpunkt und der echten letzten Bewegung wieder als frisch — und
// bucht über die jüngere Wirklichkeit (Rasterdurchgang 15.09.2026, OFFEN.md 5.15). Zwei Wege
// dorthin: ein Schreiber mit übergebenem Zeitpunkt (Nachbuchen), und ein Schreiber mit
// CURRENT_TIMESTAMP in einer Transaktion, die vor der letzten Bewegung begonnen hat.
func TestBewegungsstempel_LaeuftNieRueckwaerts(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	stempel := stempelLeser(t, pool, f.exemplarID)
	jetzt := time.Now()
	vorDreissig, vorZwanzig, vorZehn := jetzt.Add(-30*time.Minute), jetzt.Add(-20*time.Minute), jetzt.Add(-10*time.Minute)

	tx := beginne(t, pool)
	loan, err := CreateLoanZumTx(ctx, tx, CreateLoanParams{
		ExemplarID:     f.exemplarID,
		LeserID:        f.schuelerID,
		BearbeiterID:   f.bearbeiterID,
		RueckgabeFrist: jetzt.AddDate(0, 0, 14),
		IstDauerleihe:  false,
		Zeitpunkt:      &vorDreissig,
	})
	if err != nil {
		t.Fatalf("ausleihen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := StempleBewegungZum(ctx, pool, f.exemplarID, nil); err != nil {
		t.Fatalf("stempeln: %v", err)
	}
	zuletzt := stempel()

	if err := StempleBewegungZum(ctx, pool, f.exemplarID, &vorZehn); err != nil {
		t.Fatalf("früher stempeln: %v", err)
	}
	if s := stempel(); !s.Equal(zuletzt) {
		t.Errorf("StempleBewegungZum mit früherem Zeitpunkt: %v → %v", zuletzt, s)
	}

	tx = beginne(t, pool)
	if err := ReturnLoanZumTx(ctx, tx, loan.ID, f.bearbeiterID, false, &vorZwanzig); err != nil {
		t.Fatalf("zurückgeben: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if s := stempel(); !s.Equal(zuletzt) {
		t.Errorf("ReturnLoanZumTx mit früherem Zeitpunkt: %v → %v", zuletzt, s)
	}

	tx = beginne(t, pool)
	if _, err := HoleExemplarZurueck(ctx, tx, f.exemplarID, f.bearbeiterID, &vorZehn); err != nil {
		t.Fatalf("zurückholen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if s := stempel(); !s.Equal(zuletzt) {
		t.Errorf("HoleExemplarZurueck mit früherem Zeitpunkt: %v → %v", zuletzt, s)
	}
}

func TestBewegungsstempel_LangeTransaktionSetztIhnNichtZurueck(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	stempel := stempelLeser(t, pool, f.exemplarID)

	tx := beginne(t, pool)
	loan, err := CreateLoanZumTx(ctx, tx, CreateLoanParams{
		ExemplarID:     f.exemplarID,
		LeserID:        f.schuelerID,
		BearbeiterID:   f.bearbeiterID,
		RueckgabeFrist: time.Now().AddDate(0, 0, 14),
		IstDauerleihe:  false,
		Zeitpunkt:      nil,
	})
	if err != nil {
		t.Fatalf("ausleihen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	lang := beginne(t, pool)
	defer db.SafeRollback(ctx, lang)
	if _, err := lang.Exec(ctx, `SELECT now()`); err != nil { // legt den Beginn der Transaktion fest
		t.Fatalf("now: %v", err)
	}
	warte(t, pool)
	if err := StempleBewegungZum(ctx, pool, f.exemplarID, nil); err != nil {
		t.Fatalf("stempeln: %v", err)
	}
	zuletzt := stempel()

	if _, err := meldeSchaden(ctx, lang, meldeSchadenParams{
		copyID:       f.exemplarID,
		loanID:       loan.ID,
		benutzerID:   f.bearbeiterID,
		beschreibung: "Stempel-Test",
		art:          SchadensArtNichtZurueck,
		betrag:       10,
	}); err != nil {
		t.Fatalf("Schaden melden: %v", err)
	}
	if err := lang.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if s := stempel(); !s.Equal(zuletzt) {
		t.Errorf("Schadensmeldung aus einer früher begonnenen Transaktion: %v → %v", zuletzt, s)
	}
}

// stempelLeser liest letzte_bewegung_am; NULL bricht ab, weil jeder Fall hier schon bewegt ist.
func stempelLeser(t *testing.T, pool *pgxpool.Pool, exemplarID string) func() time.Time {
	return func() time.Time {
		t.Helper()
		var s *time.Time
		if err := pool.QueryRow(context.Background(), `SELECT letzte_bewegung_am FROM buecher_exemplare WHERE id = $1`, exemplarID).Scan(&s); err != nil || s == nil {
			t.Fatalf("Stempel lesen: %v %v", s, err)
		}
		return *s
	}
}

// warte lässt die Datenbank-Uhr sichtbar weiterlaufen; CURRENT_TIMESTAMP hat Mikrosekunden,
// zwei Statements in Folge können aber innerhalb derselben Auflösung liegen.
func warte(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `SELECT pg_sleep(0.01)`); err != nil {
		t.Fatalf("pg_sleep: %v", err)
	}
}

type stempelFall struct{ exemplarID, schuelerID, bearbeiterID string }

func stempelAufbau(t *testing.T, pool *pgxpool.Pool) stempelFall {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var f stempelFall
	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Stem', 'Pel', $1, 'mitarbeiter', true) RETURNING id`,
		"stempel-"+suffix+"@schule.invalid").Scan(&f.bearbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Anna', 'Bewegt', '07B', 2031) RETURNING id`, "S-"+suffix).Scan(&f.schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Stempel-Testband', 'Prüfer', 'Buch', false) RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 10.00) RETURNING id`, titelID, "B-ST-"+suffix).Scan(&f.exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	return f
}

func beginne(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	return tx
}
