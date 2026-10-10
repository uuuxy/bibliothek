package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Gegen das Ausleihlimit zählen die offenen Ausleihen des Lesers an Büchern, die kein
// Lernmittel sind: nicht das Schulbuch, nicht das zurückgegebene Buch, nicht das Gerät und nicht
// das Buch eines anderen Lesers.
func TestZaehleOffeneBuechereiAusleihen_NurOffeneBuecherOhneLernmittel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	geraetID, leserID, bearbeiterID := geraeteAufbau(t, tx, "limit")
	_, andererLeser, _ := geraeteAufbau(t, tx, "limit-anderer")
	var buecherei, schulbuch string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Limit-Probe Bücherei') RETURNING id::text`).Scan(&buecherei); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('Limit-Probe Schulbuch', true) RETURNING id::text`).Scan(&schulbuch); err != nil {
		t.Fatal(err)
	}
	leihe := func(nummer, titelID, leser string, zurueck bool) {
		t.Helper()
		if _, err := tx.Exec(ctx, `WITH e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id)
			INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			SELECT id, $3, $4, now() - interval '5 days', now() + interval '9 days', CASE WHEN $5 THEN now() END FROM e`,
			titelID, nummer, leser, bearbeiterID, zurueck); err != nil {
			t.Fatalf("Ausleihe %s anlegen: %v", nummer, err)
		}
	}
	leihe("LIM-1", buecherei, leserID, false)
	leihe("LIM-2", buecherei, leserID, false)
	leihe("LIM-ZURUECK", buecherei, leserID, true)
	leihe("LIM-SCHULBUCH", schulbuch, leserID, false)
	leihe("LIM-ANDERER", buecherei, andererLeser, false)
	if _, err := tx.Exec(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, bearbeiter_id, rueckgabe_frist)
		VALUES ($1, $2, $3, now() + interval '14 days')`, geraetID, leserID, bearbeiterID); err != nil {
		t.Fatalf("Geräte-Ausleihe anlegen: %v", err)
	}

	ist, err := ZaehleOffeneBuechereiAusleihen(ctx, tx, leserID)
	if err != nil {
		t.Fatalf("ZaehleOffeneBuechereiAusleihen: %v", err)
	}
	if ist != 2 {
		t.Errorf("%d Ausleihen gezählt, erwartet 2", ist)
	}
}

// Ein Exemplar ist reserviert, solange es abholbereit für jemanden im Fach liegt und die
// Abholfrist läuft: nicht mehr nach ihrem Ablauf, nicht für eine wartende Vormerkung und nicht,
// wenn ein anderes Exemplar bereitliegt.
func TestReservierungAmExemplar_NurAbholbereitInDerFrist(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var schueler string
	if err := tx.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('A-RES-1', 'Vorname der Reservierung', 'Nachname der Reservierung', '06C', 2032) RETURNING id::text`).Scan(&schueler); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Je Fall ein eigener Titel: Ein Leser hat je Titel höchstens eine Vormerkung.
	exemplar := func(name, status, frist string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `WITH t AS (INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id),
			     e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, $2 FROM t RETURNING id, titel_id),
			     v AS (INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
			           SELECT titel_id, $3, $4::text, id, now() + $5::interval FROM e WHERE $4::text <> '')
			SELECT id::text FROM e`, "Reservierung "+name, "RES-"+name, schueler, status, frist).Scan(&id); err != nil {
			t.Fatalf("%s anlegen: %v", name, err)
		}
		return id
	}
	bereit := exemplar("bereit", "abholbereit", "2 days")
	abgelaufen := exemplar("abgelaufen", "abholbereit", "-1 hour")
	wartend := exemplar("wartend", "wartend", "2 days")
	frei := exemplar("frei", "", "0")

	ist, da, err := ReservierungAmExemplar(ctx, tx, bereit)
	if err != nil || !da {
		t.Fatalf("bereitliegendes Exemplar: bereit=%v, Fehler %v", da, err)
	}
	if soll := (AbholfachReservierung{LeserID: schueler, Vorname: "Vorname der Reservierung", Nachname: "Nachname der Reservierung"}); ist != soll {
		t.Errorf("Reservierung %+v, erwartet %+v", ist, soll)
	}
	for name, id := range map[string]string{"Abholfrist abgelaufen": abgelaufen, "Vormerkung wartend": wartend, "ohne Vormerkung": frei} {
		if _, da, err := ReservierungAmExemplar(ctx, tx, id); err != nil || da {
			t.Errorf("%s: bereit=%v, Fehler %v; erwartet nicht reserviert", name, da, err)
		}
	}
}
