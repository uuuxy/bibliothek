package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der Reiter „Ausleiher" der Buchakte beantwortet die Frage „wer hat den Titel gerade?".
// Fragte er nur nach Schülern, verschwiege er genau die Exemplare, die im Handapparat einer
// Lehrkraft liegen: Der Zähler „Ausleiher (n)" zeigte weniger, als ausgeliehen sind, und wer
// ein Exemplar sucht, suchte es im Regal. Die Titel-Historie daneben prüft denselben Fall
// (titel_historie_pg_test.go).
//
// Beim Kollegen steht das Wort seiner Art statt „Lehrer"; ein Fachbereich ist keine
// Lehrkraft (Migration 153).
func TestTitelAusleiher_AusleiheEinerLehrkraftStehtDrin(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID, ex1, ex2, ex3, schuelerID, lehrerID, fachbereichID string
	titelID = testdaten.LegeTitelAn(t, pool, "Ausleiher-Titel", 5)
	ex1 = testdaten.LegeExemplarAn(t, pool, titelID, "B-AUSL-1", true, "")
	ex2 = testdaten.LegeExemplarAn(t, pool, titelID, "B-AUSL-2", true, "")
	ex3 = testdaten.LegeExemplarAn(t, pool, titelID, "B-AUSL-3", true, "")
	// Ein Fachbereich hat kein Konto, nur die Leserzeile (leserart.HatKonto).
	fachbereichID = testdaten.LegeLeserAn(t, pool, "L-AUSL-FB", "Fachbereich", "Erdkunde", "fachbereich")
	schuelerID = testdaten.LegeSchuelerMitNamenAn(t, pool, "S-AUSL", "Mia", "Muster", "7a", 2030)
	lehrerID = testdaten.LegeKontoAn(t, pool, "Lena", "Lehr", "lehr-ausl@example.org", "kollegium", true)
	// Der Ausweis steht an der Leserzeile (Migration 125) — der Reiter zeigt ihn von dort.
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = 'L-AUSL'
		WHERE id = (SELECT leser_id FROM benutzer WHERE id = $1)`, lehrerID); err != nil {
		t.Fatal(err)
	}
	// Zwei laufende Ausleihen desselben Titels: eine Schülerin (Frist in 3 Tagen), eine
	// Lehrkraft (Frist in 10 Tagen) — sortiert wird nach Frist, die Schülerin steht oben.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now(), now() + interval '3 days')`, ex1, schuelerID); err != nil {
		t.Fatalf("Schüler-Ausleihe: %v", err)
	}
	// ist_handapparat wie der echte Schreiber (legeAusleiheAn: jeder Nicht-Schüler).
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, ist_handapparat)
		VALUES ($1, (SELECT leser_id FROM benutzer WHERE id = $2), now(), now() + interval '10 days', true)`, ex2, lehrerID); err != nil {
		t.Fatalf("Lehrer-Ausleihe: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, ist_handapparat)
		VALUES ($1, $2, now(), now() + interval '20 days', true)`, ex3, fachbereichID); err != nil {
		t.Fatalf("Fachbereich-Ausleihe: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/ausleiher", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.TitelAusleiher()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var got []ausleiherZeile
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Ausleiher: %d, erwartet 3 (Schülerin, Lehrkraft, Fachbereich): %+v", len(got), got)
	}
	if got[0].Vorname != "Mia" || got[0].Klasse != "07A" || got[0].ExemplarBarcode != "B-AUSL-1" {
		t.Errorf("Schüler-Ausleihe: %+v", got[0])
	}
	if got[1].Vorname != "Lena" || got[1].Nachname != "Lehr" {
		t.Errorf("Lehrer-Ausleihe fehlt oder trägt den falschen Namen: %+v", got[1])
	}
	// Der Klassenfilter des Reiters liest genau dieses Feld.
	if got[1].Klasse != "Lehrkraft" {
		t.Errorf("Lehrer-Ausleihe muss als 'Lehrkraft' ausgewiesen sein: %+v", got[1])
	}
	if got[2].Nachname != "Erdkunde" || got[2].Klasse != "Fachbereich" || !got[2].IstDauerleihe {
		t.Errorf("Fachbereich-Ausleihe muss als 'Fachbereich' ausgewiesen sein, nicht als Lehrkraft: %+v", got[2])
	}
	if got[1].AusleiherBarcode != "L-AUSL" || got[1].ExemplarBarcode != "B-AUSL-2" {
		t.Errorf("Lehrer-Ausleihe: Barcodes falsch: %+v", got[1])
	}
	// Die Dauerleihe trägt ihr Merkmal, die Schüler-Ausleihe nicht: Danach entscheiden
	// Liste und Druck, ob eine abgelaufene Frist rot wird.
	if !got[1].IstDauerleihe {
		t.Errorf("Lehrer-Ausleihe ohne ist_dauerleihe — Liste und Druck färben sie nach einem Jahr rot: %+v", got[1])
	}
	if got[0].IstDauerleihe {
		t.Errorf("Schüler-Ausleihe als Dauerleihe markiert: %+v", got[0])
	}
}
