package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"
)

// Ein aus Littera übernommener Schüler trägt als Herkunftsmarke lusd_id = 'littera:<ID>'
// (internal/littera/schreiber_personen.go). Kommt derselbe Schüler später über den
// LUSD-Export mit seiner echten LUSD-ID, muss er ADOPTIERT werden — genau wie eine
// Handanlage ohne ID. Andernfalls entsteht für jeden Littera-Schüler ein Duplikat:
// Die Waisen-Suche fragt `lusd_id IS NULL`, der Littera-Schüler fällt durch, die CSV-Zeile
// wird Neuzugang, und der Unique-Index unique_schueler_name_gebdatum greift nicht, weil
// er Zeilen mit gesetzter lusd_id ausnimmt (Migration 074).
func TestLusdAutoMatching_LitteraSchuelerWirdAdoptiert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	geb := time.Date(2012, 9, 15, 0, 0, 0, 0, time.UTC)

	var litteraID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_id)
		VALUES ('Lena','Littera','6b','L-4711',2031,$1,'littera:4711') RETURNING id`, geb).Scan(&litteraID); err != nil {
		t.Fatalf("Littera-Schüler anlegen: %v", err)
	}

	rec := parsedStudentRow{
		LusdID: "LUSD-9001", Vorname: "Lena", Nachname: "Littera", Klasse: "7b",
		GebDatum: &geb, LineNum: 1,
	}

	prev, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Adoptions) != 1 || len(prev.NewStudents) != 0 {
		t.Fatalf("erwartet 1 Adoption und 0 Neuzugänge, waren %d Adoptionen / %d Neuzugänge",
			len(prev.Adoptions), len(prev.NewStudents))
	}

	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{rec}, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE nachname='Littera' AND vorname='Lena' AND deleted_at IS NULL`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("erwartet 1 Datensatz (adoptiert), waren %d — der Littera-Schüler wurde dupliziert", anzahl)
	}
	var lusd, klasse string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(lusd_id,''), klasse FROM schueler WHERE id=$1`, litteraID).Scan(&lusd, &klasse); err != nil {
		t.Fatal(err)
	}
	if lusd != "LUSD-9001" {
		t.Errorf("LUSD-ID nicht nachgetragen (noch %q)", lusd)
	}
	if klasse != "7b" {
		t.Errorf("Klasse nicht übernommen: %q", klasse)
	}
}
