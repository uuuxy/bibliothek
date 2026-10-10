package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Bestellsuche liest jede Spalte in ihr Feld; jede trägt einen eigenen Wert. Der Bestand
// zählt die Exemplare, die nicht ausgesondert sind, auch ein bestelltes. Ein Titel, der außer
// seinem Titel nichts trägt und kein Exemplar hat, steht mit leeren Feldern in der Liste: Er
// lässt sich bestellen. Ohne eigenes Cover nennt der Treffer die Cover-Adresse der DNB.
func TestSucheTitelZumBestellen_JedeSpalteKommtInIhremFeldAn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var voll, karg, ohneCover string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, isbn, verlag, cover_url, signatur, ist_lernmittel)
		VALUES ('Suchfelder Probeband', 'Autorin der Suchfelder', '9780000577061', 'Verlag der Suchfelder',
		        '/uploads/covers/suchfelder.webp', 'Suf 7', true) RETURNING id::text`).Scan(&voll); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Suchfelder Kargband') RETURNING id::text`).
		Scan(&karg); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Suchfelder Leihband', '9780000577085')
		RETURNING id::text`).Scan(&ohneCover); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	for _, sql := range []string{
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'SUF-REGAL')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus) VALUES ($1, 'SUF-BESTELLT', false, 'bestellt')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, 'SUF-AUSGESONDERT', false, true, 'AUSSORTIERT')`,
	} {
		if _, err := tx.Exec(ctx, sql, voll); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}

	treffer, err := SucheTitelZumBestellen(ctx, tx, "Suchfelder")
	if err != nil {
		t.Fatalf("SucheTitelZumBestellen: %v", err)
	}
	ist := map[string]BestellsucheTreffer{}
	for _, tr := range treffer {
		ist[tr.ID] = tr
	}
	soll := map[string]BestellsucheTreffer{
		voll: {
			ID: voll, Titel: "Suchfelder Probeband", Autor: "Autorin der Suchfelder", ISBN: "9780000577061",
			Verlag: "Verlag der Suchfelder", CoverURL: "/uploads/covers/suchfelder.webp", Signatur: "Suf 7",
			IstLernmittel: true, Bestand: 2,
		},
		karg: {ID: karg, Titel: "Suchfelder Kargband"},
		ohneCover: {
			ID: ohneCover, Titel: "Suchfelder Leihband", ISBN: "9780000577085",
			CoverURL: "https://portal.dnb.de/opac/mvb/cover?isbn=9780000577085",
		},
	}
	if len(treffer) != len(soll) {
		t.Errorf("%d Treffer, erwartet %d: %+v", len(treffer), len(soll), treffer)
	}
	for id, want := range soll {
		if ist[id] != want {
			t.Errorf("Treffer:\n  ist  %+v\n  soll %+v", ist[id], want)
		}
	}
}

// Die Bestellsuche findet einen Titel über den Volltext, über ein Stück seines Titels oder
// seines Autors und über ein Stück seiner ISBN. Jeder Suchtext trifft über genau einen der
// Wege: Die Stücke enden mitten im Wort, der Volltext nennt die Wörter in anderer Folge und Form.
func TestSucheTitelZumBestellen_FindetUeberJedenWeg(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, isbn)
		VALUES ('Wegprobe Zeitungen der Stadt', 'Verfasserprobe Meier', '9780000913357') RETURNING id::text`).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	for _, f := range []struct{ weg, suchtext string }{
		{"Volltext", "Stadt Zeitung Wegprobe"},
		{"Stück des Titels", "Wegprobe Zeitu"},
		{"Stück des Autors", "Verfasserprobe Mei"},
		{"Stück der ISBN", "00009133"},
	} {
		treffer, err := SucheTitelZumBestellen(ctx, tx, f.suchtext)
		if err != nil {
			t.Fatalf("%s: %v", f.weg, err)
		}
		if len(treffer) != 1 || treffer[0].ID != id {
			t.Errorf("%s, Suche %q: %+v, erwartet den Titel %s", f.weg, f.suchtext, treffer, id)
		}
	}
	treffer, err := SucheTitelZumBestellen(ctx, tx, "Wegprobe Zeitschrift")
	if err != nil {
		t.Fatal(err)
	}
	if len(treffer) != 0 {
		t.Errorf("Suchtext, den kein Weg trifft: %+v, erwartet keinen Treffer", treffer)
	}
}

// Die Bestellsuche liefert höchstens 50 Treffer und wählt sie nach dem Rang des Volltexts: Ein
// Titel, der das Wort des Suchtexts trägt, steht vor denen, die es nur als Stück im Autor
// tragen, wo auch immer sein Titel im Alphabet steht. Gleichrangige stehen nach ihrem Titel.
func TestSucheTitelZumBestellen_HoechstensFuenfzigUndDerBesteZuerst(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	for _, anfang := range []string{"Aaa", "Zzz"} {
		if _, err := tx.Exec(ctx, `INSERT INTO buecher_titel (titel, autor)
			SELECT $1 || ' Rangfolge ' || lpad(n::text, 3, '0'), 'Rangprobenautor' FROM generate_series(1, 51) n`, anfang); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Mmm Rangprobe')`); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	treffer, err := SucheTitelZumBestellen(ctx, tx, "Rangprobe")
	if err != nil {
		t.Fatalf("SucheTitelZumBestellen: %v", err)
	}
	if len(treffer) != 50 {
		t.Fatalf("%d Treffer, erwartet 50 von 103", len(treffer))
	}
	if treffer[0].Titel != "Mmm Rangprobe" || treffer[1].Titel != "Aaa Rangfolge 001" || treffer[49].Titel != "Aaa Rangfolge 049" {
		t.Errorf("Reihenfolge: zuerst %q, dann %q, zuletzt %q; erwartet „Mmm Rangprobe\", „Aaa Rangfolge 001\", „Aaa Rangfolge 049\"",
			treffer[0].Titel, treffer[1].Titel, treffer[49].Titel)
	}
}

// ISBNsImKatalog nennt von den gefragten ISBNs die, die ein Titel trägt, unter ihrer
// Normalform. Auch die Spalte geht durch die Normalform: Zwei Titel, die Migration 140 oder
// 157 als Dublette stehen ließ, tragen ihre ISBN beide in alter Schreibweise.
func TestISBNsImKatalog_NenntVorhandeneUnterIhrerNormalform(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	if _, err := tx.Exec(ctx, `ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_isbn_normalform`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES
		('ISBN-Menge Normalform', '9780000913364'),
		('ISBN-Menge Altform eins', '978-0-00-091338-8'),
		('ISBN-Menge Altform zwei', '978 0 00 091338 8')`); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	menge, err := ISBNsImKatalog(ctx, tx, []string{"9780000913364", "9780000913388", "9780000913371"})
	if err != nil {
		t.Fatalf("ISBNsImKatalog: %v", err)
	}
	_, normal := menge["9780000913364"]
	_, alt := menge["9780000913388"]
	if len(menge) != 2 || !normal || !alt {
		t.Errorf("Menge %v, erwartet 9780000913364 und 9780000913388", menge)
	}
}

// Ohne gefragte ISBN ist die Menge leer, und die Datenbank wird nicht gefragt.
func TestISBNsImKatalog_OhneISBNKeineAbfrage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	menge, err := ISBNsImKatalog(context.Background(), mock, nil)
	if err != nil {
		t.Fatalf("ISBNsImKatalog: %v", err)
	}
	if len(menge) != 0 {
		t.Errorf("Menge %v, erwartet leer", menge)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
