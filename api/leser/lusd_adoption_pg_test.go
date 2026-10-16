package leser

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Zuordnung im Import: Eine LUSD-Zeile, deren ID im Bestand fehlt, trifft über Name und
// Geburtsdatum auf einen bestehenden ID-losen Schüler (Handanlage oder Littera). Statt ihn
// zu duplizieren, wird die LUSD-ID nachgetragen (Adoption) und die Klasse übernommen. Ohne
// Geburtsdatum oder bei Mehrdeutigkeit wird nicht adoptiert, sondern regulär neu angelegt.
func TestLUSDZuordnung_AdoptiertWaiseStattDuplikat(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	geb := time.Date(2010, 5, 4, 0, 0, 0, 0, time.UTC)
	gebStr := "2010-05-04"

	// Ein ID-loser Bestandsschüler (z. B. aus Littera importiert), Klasse 7a.
	var waiseID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Mia','Waise','7a','W-ADOPT-1',2030,$1,NULL) RETURNING id`, geb).Scan(&waiseID); err != nil {
		t.Fatalf("Waise anlegen: %v", err)
	}

	rec := lusd.Zeile{
		LUSDID: "LUSD-ADOPT-1", Vorname: "Mia", Nachname: "Waise", Klasse: "8a",
		Geburtsdatum: &geb, Nr: 1,
	}

	// Vorschau: als Adoption erkannt, nicht als Neuzugang.
	prev, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptionen) != 1 {
		t.Fatalf("len(prev.Adoptionen) = %d, erwartet 1 Adoption (Neuzugaenge: %d)", len(prev.Adoptionen), len(prev.Neuzugaenge))
	}
	if prev.Adoptionen[0].SchuelerID != waiseID || prev.Adoptionen[0].LUSDID != "LUSD-ADOPT-1" {
		t.Errorf("Adoption falsch: %+v", prev.Adoptionen[0])
	}
	if prev.Adoptionen[0].Geburtsdatum != gebStr {
		t.Errorf("Geburtsdatum falsch: %q", prev.Adoptionen[0].Geburtsdatum)
	}
	if len(prev.Neuzugaenge) != 0 {
		t.Errorf("len(prev.Neuzugaenge) = %d, erwartet keine Neuzugänge", len(prev.Neuzugaenge))
	}

	// Anwenden: derselbe Datensatz bekommt die LUSD-ID + neue Klasse, kein Duplikat.
	if _, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{rec}, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE nachname='Waise' AND vorname='Mia' AND deleted_at IS NULL`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("anzahl = %d, erwartet 1 Datensatz (adoptiert, nicht dupliziert)", anzahl)
	}
	var lusd, klasse string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(lusd_id,''), klasse FROM schueler WHERE id=$1`, waiseID).Scan(&lusd, &klasse); err != nil {
		t.Fatal(err)
	}
	if lusd != "LUSD-ADOPT-1" {
		t.Errorf("LUSD-ID nicht nachgetragen: %q", lusd)
	}
	if klasse != "08A" {
		t.Errorf("Klasse nicht übernommen: %q", klasse)
	}
}

// Ohne Geburtsdatum in der CSV-Zeile gibt es keinen sicheren Abgleich — dann nicht
// adoptieren, sondern regulär als Neuzugang behandeln.
func TestLUSDZuordnung_OhneGeburtsdatumKeineAdoption(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	geb := time.Date(2011, 3, 3, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Tom','Waise','7b','W-ADOPT-2',2030,$1,NULL)`, geb); err != nil {
		t.Fatalf("Waise anlegen: %v", err)
	}

	rec := lusd.Zeile{LUSDID: "LUSD-ADOPT-2", Vorname: "Tom", Nachname: "Waise", Klasse: "8b", Geburtsdatum: nil}
	prev, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptionen) != 0 {
		t.Errorf("ohne Geburtsdatum darf nicht adoptiert werden, waren %d Adoptionen", len(prev.Adoptionen))
	}
	if len(prev.Neuzugaenge) != 1 {
		t.Errorf("len(prev.Neuzugaenge) = %d, erwartet 1 Neuzugang", len(prev.Neuzugaenge))
	}
}

// Zwei ID-lose Schüler mit gleichem Namen und Geburtsdatum (Schreibvarianten) wären
// mehrdeutig, und der Import adoptierte keinen. Seit Migration 108 kann dieser Zustand gar
// nicht entstehen: Der Namensindex rechnet in derselben Normalform wie der LUSD-Schlüssel,
// die zweite Schreibvariante wird an der Datenbank abgewiesen. Der Test belegt das Verbot
// und dass der eine verbliebene Schüler eindeutig adoptiert wird. Die Mehrdeutig-Behandlung
// im Code bleibt als Sicherheitsnetz stehen.
func TestLUSDZuordnung_SchreibvarianteKannNichtMehrdeutigWerden(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	geb := time.Date(2012, 6, 6, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('anna','Zwilling','7c','W-AMB-a',2030,$1,NULL)`, geb); err != nil {
		t.Fatalf("Waise anlegen: %v", err)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Anna','Zwilling','7c','W-AMB-b',2030,$1,NULL)`, geb)
	if err == nil || !strings.Contains(err.Error(), "unique_schueler_name_gebdatum") {
		t.Fatalf("Schreibvariante desselben Menschen wurde angelegt — Migration 108 greift nicht: %v", err)
	}

	rec := lusd.Zeile{LUSDID: "LUSD-AMB", Vorname: "Anna", Nachname: "Zwilling", Klasse: "8c", Geburtsdatum: &geb}
	prev, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptionen) != 1 {
		t.Errorf("der eine verbliebene Schüler muss eindeutig adoptiert werden, waren %d Adoptionen", len(prev.Adoptionen))
	}
	if len(prev.Neuzugaenge) != 0 {
		t.Errorf("len(prev.Neuzugaenge) = %d, erwartet 0 Neuzugänge", len(prev.Neuzugaenge))
	}
}
