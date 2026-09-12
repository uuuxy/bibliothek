package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
)

// Der Ausleiher-Reiter der Buch-Akte beantwortet die Frage „wer hat den Titel gerade?".
//
// Bis zum 12.09.2026 fragte er nur nach Schülern: `JOIN schueler s ON a.schueler_id = s.id`
// ist ein INNER JOIN, und eine Ausleihe an eine Lehrkraft trägt keine schueler_id. Damit
// verschwieg der Reiter genau die Exemplare, die im Handapparat einer Lehrkraft liegen —
// der Zähler „Ausleiher (n)" zeigte weniger, als ausgeliehen sind, und wer ein Exemplar
// suchte, suchte es im Regal. Die Titel-Historie daneben kennt den Fall seit dem 22.08.
// (title_history_pg_test.go); der Reiter blieb zurück.
func TestTitleBorrowers_LehrerAusleiheStehtDrin(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID, ex1, ex2, schuelerID, lehrerID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Ausleiher-Titel') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-AUSL-1') RETURNING id`, titelID).Scan(&ex1); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-AUSL-2') RETURNING id`, titelID).Scan(&ex2); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-AUSL', 'Mia', 'Muster', '7a', 2030) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('L-AUSL', 'Lena', 'Lehr', 'lehr-ausl@example.org', 'kollegium', true) RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatal(err)
	}
	// Zwei laufende Ausleihen desselben Titels: eine Schülerin (Frist in 3 Tagen), eine
	// Lehrkraft (Frist in 10 Tagen) — sortiert wird nach Frist, die Schülerin steht oben.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now(), now() + interval '3 days')`, ex1, schuelerID); err != nil {
		t.Fatalf("Schüler-Ausleihe: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now(), now() + interval '10 days')`, ex2, lehrerID); err != nil {
		t.Fatalf("Lehrer-Ausleihe: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/ausleiher", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.GetTitleBorrowersHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	var got []TitleBorrower
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("erwartet 2 Ausleiher (Schülerin UND Lehrkraft), bekam %d: %+v", len(got), got)
	}
	if got[0].Vorname != "Mia" || got[0].Klasse != "07A" || got[0].ExemplarBarcode != "B-AUSL-1" {
		t.Errorf("Schüler-Ausleihe: %+v", got[0])
	}
	if got[1].Vorname != "Lena" || got[1].Nachname != "Lehr" {
		t.Errorf("Lehrer-Ausleihe fehlt oder trägt den falschen Namen: %+v", got[1])
	}
	if got[1].Klasse != "Lehrer" {
		t.Errorf("Lehrer-Ausleihe muss als 'Lehrer' ausgewiesen sein (der Klassenfilter des Reiters liest genau dieses Feld): %+v", got[1])
	}
	if got[1].SchuelerBarcode != "L-AUSL" || got[1].ExemplarBarcode != "B-AUSL-2" {
		t.Errorf("Lehrer-Ausleihe: Barcodes falsch: %+v", got[1])
	}
}
