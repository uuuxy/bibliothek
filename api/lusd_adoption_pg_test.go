package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"
)

// Auto-Matching: Eine LUSD-Zeile, deren ID im Bestand fehlt, trifft über
// Name+Geburtsdatum auf einen bestehenden ID-losen Schüler (Handanlage/Littera).
// Statt ihn zu duplizieren, wird die LUSD-ID nachgetragen (Adoption) und die
// Klasse übernommen. Ohne Geburtsdatum oder bei Mehrdeutigkeit wird NICHT
// adoptiert, sondern regulär neu angelegt. Betreiber-Entscheidung 18.08.2026.
func TestLusdAutoMatching_AdoptiertWaiseStattDuplikat(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	geb := time.Date(2010, 5, 4, 0, 0, 0, 0, time.UTC)
	gebStr := "2010-05-04"

	// Ein ID-loser Bestandsschüler (z. B. aus Littera importiert), Klasse 7a.
	var waiseID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Mia','Waise','7a','W-ADOPT-1',2030,$1,NULL) RETURNING id`, geb).Scan(&waiseID); err != nil {
		t.Fatalf("Waise anlegen: %v", err)
	}

	rec := parsedStudentRow{
		LusdID: "LUSD-ADOPT-1", Vorname: "Mia", Nachname: "Waise", Klasse: "8a",
		GebDatum: &geb, LineNum: 1,
	}

	// Vorschau: als Adoption erkannt, NICHT als Neuzugang.
	prev, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptions) != 1 {
		t.Fatalf("erwartet 1 Adoption, waren %d (NewStudents: %d)", len(prev.Adoptions), len(prev.NewStudents))
	}
	if prev.Adoptions[0].SchuelerID != waiseID || prev.Adoptions[0].LusdID != "LUSD-ADOPT-1" {
		t.Errorf("Adoption falsch: %+v", prev.Adoptions[0])
	}
	if prev.Adoptions[0].Geburtsdatum != gebStr {
		t.Errorf("Geburtsdatum falsch: %q", prev.Adoptions[0].Geburtsdatum)
	}
	if len(prev.NewStudents) != 0 {
		t.Errorf("keine Neuzugänge erwartet, waren %d", len(prev.NewStudents))
	}

	// Anwenden: derselbe Datensatz bekommt die LUSD-ID + neue Klasse, KEIN Duplikat.
	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE nachname='Waise' AND vorname='Mia' AND deleted_at IS NULL`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("erwartet 1 Datensatz (adoptiert, nicht dupliziert), waren %d", anzahl)
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

// Ohne Geburtsdatum in der CSV-Zeile gibt es keinen sicheren Abgleich — dann NICHT
// adoptieren, sondern regulär als Neuzugang behandeln.
func TestLusdAutoMatching_OhneGeburtsdatumKeineAdoption(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	geb := time.Date(2011, 3, 3, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Tom','Waise','7b','W-ADOPT-2',2030,$1,NULL)`, geb); err != nil {
		t.Fatalf("Waise anlegen: %v", err)
	}

	rec := parsedStudentRow{LusdID: "LUSD-ADOPT-2", Vorname: "Tom", Nachname: "Waise", Klasse: "8b", GebDatum: nil}
	prev, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptions) != 0 {
		t.Errorf("ohne Geburtsdatum darf nicht adoptiert werden, waren %d Adoptionen", len(prev.Adoptions))
	}
	if len(prev.NewStudents) != 1 {
		t.Errorf("erwartet 1 Neuzugang, waren %d", len(prev.NewStudents))
	}
}

// Zwei ID-lose Schüler mit gleichem Name+Geburtsdatum (case-Varianten) sind
// MEHRDEUTIG — dann NICHT adoptieren (lieber neu anlegen als falsch zusammenführen).
func TestLusdAutoMatching_MehrdeutigKeineAdoption(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	geb := time.Date(2012, 6, 6, 0, 0, 0, 0, time.UTC)
	for i, vn := range []string{"anna", "Anna"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
			VALUES ($1,'Zwilling','7c',$2,2030,$3,NULL)`, vn, "W-AMB-"+string(rune('a'+i)), geb); err != nil {
			t.Fatalf("Waise %d anlegen: %v", i, err)
		}
	}

	rec := parsedStudentRow{LusdID: "LUSD-AMB", Vorname: "Anna", Nachname: "Zwilling", Klasse: "8c", GebDatum: &geb}
	prev, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptions) != 0 {
		t.Errorf("mehrdeutiger Match darf nicht adoptiert werden, waren %d Adoptionen", len(prev.Adoptions))
	}
	if len(prev.NewStudents) != 1 {
		t.Errorf("erwartet 1 Neuzugang, waren %d", len(prev.NewStudents))
	}
}
