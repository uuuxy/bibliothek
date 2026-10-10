package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// geraeteAufbau legt über q ein Gerät, einen Leser und einen Bearbeiter an; kuerzel hält ihre
// Nummer, ihren Namen und ihre Adresse auseinander.
func geraeteAufbau(t *testing.T, q DBQueryer, kuerzel string) (geraetID, leserID, bearbeiterID string) {
	t.Helper()
	ctx := context.Background()
	if err := q.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Sperrprobe Tablet', $1) RETURNING id::text`,
		"GAS-"+kuerzel).Scan(&geraetID); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if err := q.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art) VALUES ('Sperrprobe', $1, 'lehrkraft') RETURNING id::text`,
		kuerzel).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	if err := q.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Sperrprobe', $1, $2, 'mitarbeiter', true) RETURNING id::text`,
		kuerzel, "sperrprobe-"+kuerzel+"@schule.invalid").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	return geraetID, leserID, bearbeiterID
}

// Das Gerät zu einer Nummer kommt mit jeder Spalte in ihrem Feld an. Gesucht wird über die
// Nummer auf dem Gehäuse, nicht über die Seriennummer; eine unbekannte Nummer ist pgx.ErrNoRows.
func TestLiesGeraetNachNummer_JedeSpalteKommtInIhremFeldAn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var ganz, karg string
	if err := tx.QueryRow(ctx, `INSERT INTO geraete (modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz)
		VALUES ('Feldprobe Tablet', 'SN-FELD-1', 'GAF-1', 'Ladekabel, Stift', false, true, 'Kratzer am Rand') RETURNING id::text`).
		Scan(&ganz); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Feldprobe Laptop', 'GAF-2') RETURNING id::text`).
		Scan(&karg); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}

	seriennummer, notiz := "SN-FELD-1", "Kratzer am Rand"
	faelle := map[string]Geraet{
		"GAF-1": {ID: ganz, Modellname: "Feldprobe Tablet", Seriennummer: &seriennummer, BarcodeID: "GAF-1",
			Zubehoer: "Ladekabel, Stift", IstAusleihbar: false, IstAusgesondert: true, ZustandNotiz: &notiz},
		"GAF-2": {ID: karg, Modellname: "Feldprobe Laptop", BarcodeID: "GAF-2", IstAusleihbar: true},
	}
	for nummer, soll := range faelle {
		ist, err := LiesGeraetNachNummer(ctx, tx, nummer)
		if err != nil {
			t.Fatalf("%s: %v", nummer, err)
		}
		if !reflect.DeepEqual(ist, soll) {
			t.Errorf("%s:\n  ist  %+v\n  soll %+v", nummer, ist, soll)
		}
	}
	for _, nummer := range []string{"SN-FELD-1", "GAF-3"} {
		if _, err := LiesGeraetNachNummer(ctx, tx, nummer); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("%s: Fehler %v, erwartet pgx.ErrNoRows", nummer, err)
		}
	}
}

// Die offene Ausleihe eines Geräts kommt mit jeder Spalte in ihrem Feld an. Ein Gerät, dessen
// letzte Ausleihe zurückgegeben ist, ist frei.
func TestSperreOffeneGeraeteAusleihe_NurDieOffeneMitJederSpalte(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	verliehen, leserID, bearbeiterID := geraeteAufbau(t, tx, "offen")
	frei, _, _ := geraeteAufbau(t, tx, "frei")
	ausgeliehen := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	frist := time.Date(2026, 2, 16, 21, 59, 59, 0, time.UTC)
	for _, geraetID := range []string{verliehen, frei} {
		if _, err := tx.Exec(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id)
			VALUES ($1, $2, '2026-01-10 09:00:00+00', '2026-01-24 21:59:59+00', '2026-01-20 09:00:00+00', $3)`,
			geraetID, leserID, bearbeiterID); err != nil {
			t.Fatalf("zurückgegebene Ausleihe anlegen: %v", err)
		}
	}
	var offeneID string
	if err := tx.QueryRow(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5, true) RETURNING id::text`, verliehen, leserID, ausgeliehen, frist, bearbeiterID).
		Scan(&offeneID); err != nil {
		t.Fatalf("offene Ausleihe anlegen: %v", err)
	}

	ist, offen, err := SperreOffeneGeraeteAusleihe(ctx, tx, verliehen)
	if err != nil || !offen {
		t.Fatalf("verliehenes Gerät: offen=%v, Fehler %v", offen, err)
	}
	if ist.ID != offeneID || ist.GeraetID == nil || *ist.GeraetID != verliehen || ist.SchuelerID == nil || *ist.SchuelerID != leserID ||
		ist.BearbeiterID == nil || *ist.BearbeiterID != bearbeiterID {
		t.Errorf("Kennungen der offenen Ausleihe: %+v", ist)
	}
	if !ist.AusgeliehenAm.Equal(ausgeliehen) || !ist.RueckgabeFrist.Equal(frist) || ist.RueckgabeAm != nil {
		t.Errorf("Zeiten: ausgeliehen %s, Frist %s, Rückgabe %v", ist.AusgeliehenAm, ist.RueckgabeFrist, ist.RueckgabeAm)
	}
	if ist.IstFremdrueckgabe || !ist.IstHandapparat {
		t.Errorf("Merkmale: Fremdrückgabe %v, Dauerleihe %v; erwartet false und true", ist.IstFremdrueckgabe, ist.IstHandapparat)
	}

	if _, offen, err := SperreOffeneGeraeteAusleihe(ctx, tx, frei); err != nil || offen {
		t.Errorf("Gerät mit zurückgegebener Ausleihe: offen=%v, Fehler %v; erwartet frei", offen, err)
	}
}

// Wer die offene Ausleihe eines Geräts liest, sperrt ihre Zeile: Eine zweite Transaktion wartet
// und findet danach keine offene Ausleihe mehr, wenn die erste die Rückgabe gebucht hat.
func TestSperreOffeneGeraeteAusleihe_ZweiteTransaktionWartet(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id = 'GAS-warten')`,
			`DELETE FROM geraete WHERE barcode_id = 'GAS-warten'`,
			`DELETE FROM benutzer WHERE email = 'sperrprobe-warten@schule.invalid'`,
			`DELETE FROM leser WHERE vorname = 'Sperrprobe' AND nachname = 'warten'`,
		} {
			if _, err := pool.Exec(ctx, sql); err != nil {
				t.Errorf("aufräumen (%s): %v", sql, err)
			}
		}
	})
	geraetID, leserID, bearbeiterID := geraeteAufbau(t, pool, "warten")
	if _, err := LeiheGeraetAus(ctx, pool, GeraeteAusleihe{
		GeraetID: geraetID, LeserID: leserID, BearbeiterID: bearbeiterID, RueckgabeFrist: time.Now().AddDate(0, 0, 14),
	}); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	erste := beginne(t, pool)
	defer db.SafeRollback(ctx, erste)
	ausleihe, offen, err := SperreOffeneGeraeteAusleihe(ctx, erste, geraetID)
	if err != nil || !offen {
		t.Fatalf("erste Transaktion: offen=%v, Fehler %v", offen, err)
	}

	type antwort struct {
		offen bool
		err   error
	}
	zweite := make(chan antwort, 1)
	fertig := make(chan error, 1)
	go func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			zweite <- antwort{err: err}
			fertig <- err
			return
		}
		defer db.SafeRollback(ctx, tx)
		_, offen, err := SperreOffeneGeraeteAusleihe(ctx, tx, geraetID)
		zweite <- antwort{offen: offen, err: err}
		fertig <- err
	}()

	ueberschneidung(t, pool, erste, fertig)
	if len(fertig) > 0 {
		t.Fatalf("die zweite Transaktion hat nicht gewartet: %+v", <-zweite)
	}
	if err := BucheGeraeteRueckgabe(ctx, erste, ausleihe.ID, bearbeiterID, false); err != nil {
		t.Fatalf("Rückgabe buchen: %v", err)
	}
	if err := erste.Commit(ctx); err != nil {
		t.Fatalf("erste Transaktion abschließen: %v", err)
	}
	if a := <-zweite; a.err != nil || a.offen {
		t.Errorf("zweite Transaktion nach der Rückgabe: offen=%v, Fehler %v; erwartet frei", a.offen, a.err)
	}
}

// Die Ausleihe eines Geräts trägt Gerät, Leser, Bearbeiter, Frist und das Merkmal der
// Dauerleihe, jede Angabe in ihrer Spalte. Ein zweites Mal lässt sich ein verliehenes Gerät
// nicht ausleihen: Die Datenbank weist es an uniq_ausleihen_aktiv_geraet ab.
func TestLeiheGeraetAus_SchreibtJedeAngabeUndNurEinmal(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	frist := time.Date(2026, 3, 20, 21, 59, 59, 0, time.UTC)
	for _, dauerleihe := range []bool{true, false} {
		kuerzel := "leihe-frist"
		if dauerleihe {
			kuerzel = "leihe-dauer"
		}
		geraetID, leserID, bearbeiterID := geraeteAufbau(t, tx, kuerzel)
		id, err := LeiheGeraetAus(ctx, tx, GeraeteAusleihe{
			GeraetID: geraetID, LeserID: leserID, BearbeiterID: bearbeiterID, RueckgabeFrist: frist, IstDauerleihe: dauerleihe,
		})
		if err != nil {
			t.Fatalf("%s: %v", kuerzel, err)
		}
		var (
			istID, istGeraet, istLeser, istBearbeiter string
			istFrist                                  time.Time
			istDauerleihe, offen, ohneExemplar        bool
		)
		if err := tx.QueryRow(ctx, `SELECT id::text, geraet_id::text, schueler_id::text, bearbeiter_id::text, rueckgabe_frist,
			       ist_handapparat, rueckgabe_am IS NULL, exemplar_id IS NULL
			FROM ausleihen WHERE geraet_id = $1`, geraetID).
			Scan(&istID, &istGeraet, &istLeser, &istBearbeiter, &istFrist, &istDauerleihe, &offen, &ohneExemplar); err != nil {
			t.Fatalf("%s: Ausleihe lesen: %v", kuerzel, err)
		}
		if istID != id || istGeraet != geraetID || istLeser != leserID || istBearbeiter != bearbeiterID ||
			!istFrist.Equal(frist) || istDauerleihe != dauerleihe || !offen || !ohneExemplar {
			t.Errorf("%s: Ausleihe %s an Gerät %s, Leser %s, Bearbeiter %s, Frist %s, Dauerleihe %v, offen %v",
				kuerzel, istID, istGeraet, istLeser, istBearbeiter, istFrist, istDauerleihe, offen)
		}

		if dauerleihe {
			continue
		}
		// Zuletzt, weil der Fehler die Transaktion des Tests beendet.
		_, err = LeiheGeraetAus(ctx, tx, GeraeteAusleihe{
			GeraetID: geraetID, LeserID: leserID, BearbeiterID: bearbeiterID, RueckgabeFrist: frist,
		})
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "uniq_ausleihen_aktiv_geraet" {
			t.Errorf("zweite Ausleihe desselben Geräts: Fehler %v, erwartet 23505 an uniq_ausleihen_aktiv_geraet", err)
		}
	}
}

// Die Rückgabe steht mit Zeitpunkt, Bearbeiter der Rückgabe und dem Merkmal der Fremdrückgabe
// an der Ausleihe; wer ausgeliehen hat, bleibt stehen. Eine Ausleihe, die es nicht gibt, ist
// ErrGeraeteAusleiheFehlt.
func TestBucheGeraeteRueckgabe_SchreibtJedeAngabe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	for _, fremd := range []bool{true, false} {
		kuerzel := "zurueck-selbst"
		if fremd {
			kuerzel = "zurueck-fremd"
		}
		geraetID, leserID, ausgeber := geraeteAufbau(t, tx, kuerzel)
		_, _, annehmer := geraeteAufbau(t, tx, kuerzel+"-annahme")
		id, err := LeiheGeraetAus(ctx, tx, GeraeteAusleihe{
			GeraetID: geraetID, LeserID: leserID, BearbeiterID: ausgeber, RueckgabeFrist: time.Now().AddDate(0, 0, 14),
		})
		if err != nil {
			t.Fatalf("%s: Ausleihe anlegen: %v", kuerzel, err)
		}
		if err := BucheGeraeteRueckgabe(ctx, tx, id, annehmer, fremd); err != nil {
			t.Fatalf("%s: %v", kuerzel, err)
		}
		var (
			zurueck, istFremd               bool
			istAusgeber, istAnnehmer, leser string
		)
		if err := tx.QueryRow(ctx, `SELECT rueckgabe_am = CURRENT_TIMESTAMP, ist_fremdrueckgabe, bearbeiter_id::text,
			       rueckgabe_bearbeiter_id::text, schueler_id::text
			FROM ausleihen WHERE id = $1`, id).Scan(&zurueck, &istFremd, &istAusgeber, &istAnnehmer, &leser); err != nil {
			t.Fatalf("%s: Ausleihe lesen: %v", kuerzel, err)
		}
		if !zurueck || istFremd != fremd || istAusgeber != ausgeber || istAnnehmer != annehmer || leser != leserID {
			t.Errorf("%s: zurück %v, Fremdrückgabe %v, ausgegeben von %s, angenommen von %s, Leser %s",
				kuerzel, zurueck, istFremd, istAusgeber, istAnnehmer, leser)
		}
	}

	err := BucheGeraeteRueckgabe(ctx, tx, "3f2504e0-4f89-11d3-9a0c-0305e82c3301", "3f2504e0-4f89-11d3-9a0c-0305e82c3302", false)
	if !errors.Is(err, ErrGeraeteAusleiheFehlt) {
		t.Errorf("unbekannte Ausleihe: Fehler %v, erwartet ErrGeraeteAusleiheFehlt", err)
	}
}

// LeserNameUndKlasse nennt Vorname, Nachname und Klasse und füllt sonst kein Feld: Die Antwort
// der Theke trägt den Vorbesitzer eines Geräts, mehr als seinen Namen braucht sie nicht. Ein
// Kollege hat keine Klasse, ein gelöschter Leser ist pgx.ErrNoRows.
func TestLeserNameUndKlasse_NurNameUndKlasse(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var schueler, kollege, geloescht string
	if err := tx.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('A-NUK-1', 'Vorname der Probe', 'Nachname der Probe', '07B', 2031) RETURNING id::text`).Scan(&schueler); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art) VALUES ('Kollegin', 'Ohneklasse', 'lehrkraft')
		RETURNING id::text`).Scan(&kollege); err != nil {
		t.Fatalf("Kollegin anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art, deleted_at) VALUES ('Gelöscht', 'Probe', 'lehrkraft', now())
		RETURNING id::text`).Scan(&geloescht); err != nil {
		t.Fatalf("gelöschten Leser anlegen: %v", err)
	}

	faelle := map[string]Student{
		schueler: {Vorname: "Vorname der Probe", Nachname: "Nachname der Probe", Klasse: "07B"},
		kollege:  {Vorname: "Kollegin", Nachname: "Ohneklasse"},
	}
	for id, soll := range faelle {
		ist, err := LeserNameUndKlasse(ctx, tx, id)
		if err != nil {
			t.Fatalf("%s: %v", soll.Nachname, err)
		}
		if !reflect.DeepEqual(ist, soll) {
			t.Errorf("%s:\n  ist  %+v\n  soll %+v", soll.Nachname, ist, soll)
		}
	}
	if _, err := LeserNameUndKlasse(ctx, tx, geloescht); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("gelöschter Leser: Fehler %v, erwartet pgx.ErrNoRows", err)
	}
}
