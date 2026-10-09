package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Der Stichtag räumt den Altbestand auf: Jedes Exemplar ohne Vermerk, das bis zu diesem Tag in
// den Bestand kam, gilt danach als beklebt, der Tag selbst eingeschlossen. Die Lieferung vom
// Tag danach bleibt auf der Liste der fehlenden Etiketten, ein ausgesondertes Exemplar bleibt
// unberührt. Für ein bestelltes Exemplar, das noch nicht eingetroffen ist, zählt der Tag der
// Bestellung. Die Antwort nennt die Zahl der Exemplare, die vorher keinen Vermerk hatten.
func TestEtikettenAltbestand_StichtagTrifftNurDenAltbestand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	_, rufe := protokollWelt(t, pool)
	titel := titelMitMeldebestand(t, pool, "Altbestand Etiketten", 0)

	type zeile struct {
		barcode, erworben, bestellstatus, grund string
		gedruckt                                bool
	}
	for _, z := range []zeile{
		{barcode: "ALT-ET-1", erworben: "2020-05-10"},
		{barcode: "ALT-ET-STICHTAG", erworben: "2024-12-31"},
		{barcode: "ALT-ET-DANACH", erworben: "2025-01-01"},
		{barcode: "ALT-ET-AUSGESONDERT", erworben: "2019-03-01", grund: "AUSSORTIERT"},
		{barcode: "ALT-ET-BESTELLT", erworben: "2021-09-01", bestellstatus: "bestellt"},
		{barcode: "ALT-ET-SCHON", erworben: "2018-02-02", gedruckt: true},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare
				(titel_id, barcode_id, erworben_am, etikett_gedruckt, bestellstatus, ist_ausleihbar,
				 ist_ausgesondert, aussonderung_grund)
			VALUES ($1, $2, $3::date, $4, NULLIF($5, ''), $5 = '' AND $6 = '', $6 <> '', NULLIF($6, ''))`,
			titel, z.barcode, z.erworben, z.gedruckt, z.bestellstatus, z.grund); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", z.barcode, err)
		}
	}

	rec := rufe(t, http.MethodPost, "/api/exemplare/etiketten-altbestand", `{"bis":"31.12.2024"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Stichtag in falscher Form: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1 AND etikett_gedruckt`, titel); n != 1 {
		t.Fatalf("%d Exemplare mit Vermerk nach der abgelehnten Anfrage, erwartet 1", n)
	}

	rec = rufe(t, http.MethodPost, "/api/exemplare/etiketten-altbestand", `{"bis":"2024-12-31"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Altbestand vermerken: Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Markiert int `json:"markiert"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort unlesbar: %v — %s", err, rec.Body.String())
	}
	if antwort.Markiert != 3 {
		t.Errorf("die Antwort nennt %d vermerkte Exemplare, erwartet 3", antwort.Markiert)
	}

	for barcode, soll := range map[string]bool{
		"ALT-ET-1":            true,
		"ALT-ET-STICHTAG":     true,
		"ALT-ET-DANACH":       false,
		"ALT-ET-AUSGESONDERT": false,
		"ALT-ET-BESTELLT":     true,
		"ALT-ET-SCHON":        true,
	} {
		var ist bool
		if err := pool.QueryRow(ctx, `SELECT etikett_gedruckt FROM buecher_exemplare WHERE barcode_id = $1`, barcode).Scan(&ist); err != nil {
			t.Fatalf("%s lesen: %v", barcode, err)
		}
		if ist != soll {
			t.Errorf("%s: Vermerk %v, erwartet %v", barcode, ist, soll)
		}
	}
}
