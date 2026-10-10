package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Die Liste des Zulaufs liest jede Spalte in ihr Feld. Jede Spalte trägt einen eigenen Wert:
// Mit gleichen fiele nicht auf, wenn zwei vertauscht ankämen.
func TestExemplareImZulauf_JedeSpalteKommtInIhremFeldAn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	bestellt := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	angelegt := time.Date(2026, 3, 5, 11, 0, 0, 0, time.UTC)
	var titelID, bestellungID, exemplarID string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, cover_url)
		VALUES ('Zulauf-Felder Titel', '9780000577016', '/uploads/covers/zulauf-felder.webp') RETURNING id::text`).
		Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, bestelldatum)
		VALUES ('Zulauf-Felder Händler', 'zulauf-felder@example.org', $1) RETURNING id::text`, bestellt).
		Scan(&bestellungID); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_exemplare
			(titel_id, barcode_id, ist_ausleihbar, bestellstatus, bestellung_id, zustand_notiz, erstellt_am)
		VALUES ($1, 'ZUF-1', false, 'im_zulauf', $2, 'Zulauf-Felder Notiz', $3) RETURNING id::text`,
		titelID, bestellungID, angelegt).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	zeilen, err := ExemplareImZulauf(ctx, tx)
	if err != nil {
		t.Fatalf("ExemplareImZulauf: %v", err)
	}
	var ist *ZulaufExemplar
	for i := range zeilen {
		if zeilen[i].ExemplarID == exemplarID {
			ist = &zeilen[i]
		}
	}
	if ist == nil {
		t.Fatalf("das Exemplar %s fehlt unter %d Zeilen des Zulaufs", exemplarID, len(zeilen))
	}
	if ist.TitelID != titelID || !ist.ErstelltAm.Equal(angelegt) || ist.ZustandNotiz != "Zulauf-Felder Notiz" ||
		ist.Titel != "Zulauf-Felder Titel" || ist.ISBN != "9780000577016" ||
		ist.CoverURL != "/uploads/covers/zulauf-felder.webp" {
		t.Errorf("Exemplar und Titel: %+v", *ist)
	}
	if ist.BestellungID == nil || *ist.BestellungID != bestellungID {
		t.Errorf("Bestellung: %v, erwartet %s", ist.BestellungID, bestellungID)
	}
	if ist.LieferantName == nil || *ist.LieferantName != "Zulauf-Felder Händler" {
		t.Errorf("Lieferant: %v", ist.LieferantName)
	}
	if ist.Bestelldatum == nil || !ist.Bestelldatum.Equal(bestellt) {
		t.Errorf("Bestelldatum: %v, erwartet %s", ist.Bestelldatum, bestellt)
	}
}

// Im Zulauf steht, was bestellt und noch nicht eingetroffen ist, das jüngste zuerst. Ein
// Exemplar ohne Bestellstatus gehört nicht dazu, auch wenn es nicht ausleihbar ist. Bestellung,
// Lieferant und Bestelldatum fehlen am Altbestand, eine Notiz und eine ISBN können fehlen:
// Keines davon bricht das Lesen ab.
func TestExemplareImZulauf_NurBestelltesUndDasJuengsteZuerst(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var titelID string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Zulauf-Auswahl Titel') RETURNING id::text`).
		Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	for _, sql := range []string{
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, erstellt_am)
			VALUES ($1, 'ZUW-ALT', false, 'bestellt', '2026-01-10 09:00:00+00')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, erstellt_am)
			VALUES ($1, 'ZUW-NEU', false, 'im_zulauf', '2026-02-10 09:00:00+00')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZUW-REGAL')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
			VALUES ($1, 'ZUW-WERKSTATT', false, 'beim Buchbinder')`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, 'ZUW-AUSGESONDERT', false, true, 'AUSSORTIERT')`,
	} {
		if _, err := tx.Exec(ctx, sql, titelID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}

	zeilen, err := ExemplareImZulauf(ctx, tx)
	if err != nil {
		t.Fatalf("ExemplareImZulauf: %v", err)
	}
	var nummern []string
	for _, z := range zeilen {
		if z.TitelID != titelID {
			continue
		}
		var nummer string
		if err := tx.QueryRow(ctx, `SELECT barcode_id FROM buecher_exemplare WHERE id = $1`, z.ExemplarID).Scan(&nummer); err != nil {
			t.Fatalf("Nummer lesen: %v", err)
		}
		nummern = append(nummern, nummer)
		if z.BestellungID != nil || z.LieferantName != nil || z.Bestelldatum != nil {
			t.Errorf("%s: Bestellung %v, Lieferant %v, Datum %v, erwartet ohne Bestellung", nummer, z.BestellungID, z.LieferantName, z.Bestelldatum)
		}
		if z.ZustandNotiz != "" || z.ISBN != "" {
			t.Errorf("%s: Notiz %q und ISBN %q, erwartet beide leer", nummer, z.ZustandNotiz, z.ISBN)
		}
	}
	if len(nummern) != 2 || nummern[0] != "ZUW-NEU" || nummern[1] != "ZUW-ALT" {
		t.Errorf("im Zulauf stehen %v, erwartet ZUW-NEU vor ZUW-ALT und sonst nichts", nummern)
	}
}

// Das Cover in den Listen der Bestellung (sqlCoverOderDNB), gemessen am Zulauf: der eigene
// Eintrag des Titels, sonst die Cover-Adresse der DNB zu seiner ISBN, sonst nichts. Ein leerer
// Eintrag gilt als keiner, und die Adresse nennt die ISBN ohne Bindestriche. Mit Bindestrichen
// oder leer trägt sie nur eine Zeile aus der Zeit vor dem Trigger; sie kommt deshalb an ihm
// vorbei in die Tabelle.
func TestExemplareImZulauf_CoverFaelltAufDieDNBZurueck(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	eigen, leer := "/uploads/covers/zulauf-cover.webp", ""
	const dnb = "https://portal.dnb.de/opac/mvb/cover?isbn="
	isbn := func(s string) *string { return &s }
	faelle := []struct {
		name        string
		isbn, cover *string
		soll        string
	}{
		{"eigenes Cover", isbn("9780000577023"), &eigen, eigen},
		{"leerer Eintrag", isbn("9780000577030"), &leer, dnb + "9780000577030"},
		{"kein Eintrag", isbn("9780000577047"), nil, dnb + "9780000577047"},
		{"ISBN mit Bindestrichen", isbn("978-0-00-057705-4"), nil, dnb + "9780000577054"},
		{"weder Cover noch ISBN", nil, nil, ""},
		{"leere ISBN", &leer, nil, ""},
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_isbn_normalform`); err != nil {
		t.Fatal(err)
	}
	titelZuFall := map[string]string{}
	for i, f := range faelle {
		var titelID string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, cover_url) VALUES ($1, $2, $3) RETURNING id::text`,
			"Zulauf-Cover "+f.name, f.isbn, f.cover).Scan(&titelID); err != nil {
			t.Fatalf("%s: Titel anlegen: %v", f.name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, false, 'bestellt')`, titelID, "ZUC-"+string(rune('1'+i))); err != nil {
			t.Fatalf("%s: Exemplar anlegen: %v", f.name, err)
		}
		titelZuFall[titelID] = f.name
	}

	zeilen, err := ExemplareImZulauf(ctx, tx)
	if err != nil {
		t.Fatalf("ExemplareImZulauf: %v", err)
	}
	cover := map[string]string{}
	for _, z := range zeilen {
		if name, ok := titelZuFall[z.TitelID]; ok {
			cover[name] = z.CoverURL
		}
	}
	for _, f := range faelle {
		ist, da := cover[f.name]
		if !da {
			t.Errorf("%s: das Exemplar fehlt im Zulauf", f.name)
			continue
		}
		if ist != f.soll {
			t.Errorf("%s: Cover %q, erwartet %q", f.name, ist, f.soll)
		}
	}
}

// Einbuchen gibt die genannten Exemplare frei, die bestellt sind, und nur die. Ein Exemplar
// ohne Bestellstatus bleibt, wie es ist, mit seiner Notiz: eines beim Buchbinder, ein
// ausgesondertes, das der Händler nicht lieferte, eines im Regal. Die Antwort nennt je
// freigegebenem Exemplar Nummer, Titel, Autor und ob sein Etikett gedruckt ist.
func TestBucheZulaufEin_GibtNurGenannteBestellteExemplareFrei(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var mitAutor, ohneAutor string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor) VALUES ('Einbuch-Probe mit Autor', 'Autorin der Einbuch-Probe')
		RETURNING id::text`).Scan(&mitAutor); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Einbuch-Probe ohne Autor') RETURNING id::text`).
		Scan(&ohneAutor); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	ids := map[string]string{}
	lege := func(nummer, sql string, titelID string) {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, sql+` RETURNING id::text`, titelID, nummer).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", nummer, err)
		}
		ids[nummer] = id
	}
	lege("EIN-GEDRUCKT", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, zustand_notiz, etikett_gedruckt)
		VALUES ($1, $2, false, 'im_zulauf', 'Im Zulauf - Händler', true)`, mitAutor)
	lege("EIN-UNGEDRUCKT", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, zustand_notiz)
		VALUES ($1, $2, false, 'bestellt', 'bestellt')`, ohneAutor)
	lege("EIN-NICHT-GENANNT", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, zustand_notiz)
		VALUES ($1, $2, false, 'im_zulauf', 'Im Zulauf - Händler')`, mitAutor)
	lege("EIN-WERKSTATT", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
		VALUES ($1, $2, false, 'beim Buchbinder')`, mitAutor)
	lege("EIN-AUSGESONDERT", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, zustand_notiz)
		VALUES ($1, $2, false, true, 'AUSSORTIERT', 'Händler liefert nicht')`, mitAutor)
	lege("EIN-REGAL", `INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz) VALUES ($1, $2, 'Eselsohr')`, mitAutor)

	eingebucht, err := BucheZulaufEin(ctx, tx, []string{
		ids["EIN-GEDRUCKT"], ids["EIN-UNGEDRUCKT"], ids["EIN-WERKSTATT"], ids["EIN-AUSGESONDERT"], ids["EIN-REGAL"],
	})
	if err != nil {
		t.Fatalf("BucheZulaufEin: %v", err)
	}
	antwort := map[string]EingebuchtesExemplar{}
	for _, e := range eingebucht {
		antwort[e.BarcodeID] = e
	}
	soll := map[string]EingebuchtesExemplar{
		"EIN-GEDRUCKT":   {BarcodeID: "EIN-GEDRUCKT", Titel: "Einbuch-Probe mit Autor", Autor: "Autorin der Einbuch-Probe", EtikettGedruckt: true},
		"EIN-UNGEDRUCKT": {BarcodeID: "EIN-UNGEDRUCKT", Titel: "Einbuch-Probe ohne Autor"},
	}
	if len(eingebucht) != len(soll) {
		t.Errorf("%d Exemplare eingebucht, erwartet %d: %+v", len(eingebucht), len(soll), eingebucht)
	}
	for nummer, want := range soll {
		if antwort[nummer] != want {
			t.Errorf("%s: Antwort %+v, erwartet %+v", nummer, antwort[nummer], want)
		}
	}

	type stand struct {
		ausleihbar, ausgesondert bool
		status, notiz            string
	}
	danach := map[string]stand{
		"EIN-GEDRUCKT":      {ausleihbar: true},
		"EIN-UNGEDRUCKT":    {ausleihbar: true},
		"EIN-NICHT-GENANNT": {status: "im_zulauf", notiz: "Im Zulauf - Händler"},
		"EIN-WERKSTATT":     {notiz: "beim Buchbinder"},
		"EIN-AUSGESONDERT":  {ausgesondert: true, notiz: "Händler liefert nicht"},
		"EIN-REGAL":         {ausleihbar: true, notiz: "Eselsohr"},
	}
	for nummer, want := range danach {
		var ist stand
		if err := tx.QueryRow(ctx, `SELECT ist_ausleihbar, ist_ausgesondert, coalesce(bestellstatus, ''), coalesce(zustand_notiz, '')
			FROM buecher_exemplare WHERE id = $1`, ids[nummer]).
			Scan(&ist.ausleihbar, &ist.ausgesondert, &ist.status, &ist.notiz); err != nil {
			t.Fatalf("%s lesen: %v", nummer, err)
		}
		if ist != want {
			t.Errorf("%s nach dem Einbuchen: %+v, erwartet %+v", nummer, ist, want)
		}
	}
}
