package api

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
)

// Die Barcode-Liste der Theke am echten Postgres (Stufe 2, Commit 12).
//
// Geprüft wird, worauf sich die Offline-Einordnung verlässt: Die Liste enthält JEDES nicht
// ausgesonderte Exemplar (eine gekappte Liste machte vorhandene Bücher offline zu
// „unklar"), sie enthält KEIN ausgesondertes, sie ändert ihren Stand, wenn sich der
// Bestand ändert, und bei unverändertem Stand antwortet sie 304 statt die Liste erneut zu
// schicken. Rot am alten Code: Die Route gab es nicht.
func TestBuchbarcodes_VollstaendigUndMitStand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	titelID := seedMonitorTitel(t, pool, "Barcode-Titel", "Aut Or", false, 0)
	da := []string{"B-BC-0001", "1234567890123", "LMF-BC-0003"}
	for _, b := range da {
		exemplar(t, pool, titelID, b, true, "")
	}
	weg := exemplar(t, pool, titelID, "B-BC-WEG", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, weg); err != nil {
		t.Fatalf("aussondern: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	hole := func(ifNoneMatch, accept string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/action/buchbarcodes", nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		req = req.WithContext(t.Context())
		rec := httptest.NewRecorder()
		srv.BuchbarcodesHandler()(rec, req)
		return rec
	}

	rec := hole("", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort BuchbarcodesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON: %v", err)
	}
	if antwort.Anzahl != len(antwort.Barcodes) {
		t.Errorf("Anzahl %d ≠ Liste %d", antwort.Anzahl, len(antwort.Barcodes))
	}
	enthalten := map[string]bool{}
	for _, b := range antwort.Barcodes {
		enthalten[b] = true
	}
	for _, b := range da {
		if !enthalten[b] {
			t.Errorf("Barcode %q fehlt — offline gälte dieses Buch als unklar", b)
		}
	}
	// Ein ausgesondertes Exemplar gehört hinein: Kommt ein verloren gemeldetes Buch offline
	// zurück, holt das Nachbuchen es in den Umlauf („nur_reaktiviert"). Fehlte seine Nummer,
	// gälte es an der Theke als unklar und sperrte die Zuordnung (Rasterdurchgang 15.09.2026,
	// OFFEN.md 5.15). Die Nummer bleibt eine Buchnummer, auch wenn das Buch abgeschrieben ist.
	if !enthalten["B-BC-WEG"] {
		t.Error("ein ausgesondertes Exemplar fehlt in der Liste — offline gälte es als unklar, obwohl das Nachbuchen es zurückholt")
	}
	if antwort.Stand == "" || rec.Header().Get("ETag") != `"`+antwort.Stand+`"` {
		t.Errorf("Stand %q, ETag %q — sie müssen übereinstimmen", antwort.Stand, rec.Header().Get("ETag"))
	}

	// Unverändert: 304, kein Rumpf.
	rec = hole(`"`+antwort.Stand+`"`, "")
	if rec.Code != http.StatusNotModified {
		t.Errorf("unveränderter Bestand: Status %d, erwartet 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 mit Rumpf (%d Byte)", rec.Body.Len())
	}

	// Ein neues Exemplar ändert den Stand — sonst holte der Rechner es nie.
	exemplar(t, pool, titelID, "B-BC-NEU", true, "")
	rec = hole(`"`+antwort.Stand+`"`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("nach einem neuen Exemplar: Status %d, erwartet 200 mit neuer Liste", rec.Code)
	}
	var zweite BuchbarcodesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &zweite); err != nil {
		t.Fatalf("zweite Antwort: %v", err)
	}
	if zweite.Stand == antwort.Stand {
		t.Error("der Stand blieb gleich, obwohl ein Exemplar dazugekommen ist")
	}
	if zweite.Anzahl != antwort.Anzahl+1 {
		t.Errorf("Anzahl %d, erwartet %d", zweite.Anzahl, antwort.Anzahl+1)
	}

	// Ein GELÖSCHTES Exemplar ändert den Stand ebenfalls — das ist der Fall, für den die
	// Anzahl im Stand steht: Beim Löschen ändert sich max(aktualisiert_am) nicht, wenn das
	// gelöschte Exemplar nicht das zuletzt geänderte war. Ohne die Anzahl behielte der
	// Theken-Rechner eine Nummer, die es nicht mehr gibt.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, "1234567890123"); err != nil {
		t.Fatalf("Exemplar löschen: %v", err)
	}
	rec = hole(`"`+zweite.Stand+`"`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("nach dem Löschen: Status %d, erwartet 200 mit neuer Liste", rec.Code)
	}
	var dritte BuchbarcodesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dritte); err != nil {
		t.Fatalf("dritte Antwort: %v", err)
	}
	if dritte.Stand == zweite.Stand {
		t.Error("der Stand blieb gleich, obwohl ein Exemplar gelöscht wurde — die Anzahl fehlt im Stand")
	}
	for _, b := range dritte.Barcodes {
		if b == "1234567890123" {
			t.Error("der gelöschte Barcode steht noch in der Liste")
		}
	}

	// Gepackt: derselbe Inhalt, Content-Encoding gesetzt.
	rec = hole("", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Accept-Encoding gzip → Content-Encoding %q", rec.Header().Get("Content-Encoding"))
	}
	entpacker, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gepackte Antwort nicht lesbar: %v", err)
	}
	var gepackt BuchbarcodesResponse
	if err := json.NewDecoder(entpacker).Decode(&gepackt); err != nil {
		t.Fatalf("gepackte Antwort: %v", err)
	}
	if gepackt.Anzahl != dritte.Anzahl {
		t.Errorf("gepackt %d Einträge, ungepackt %d", gepackt.Anzahl, dritte.Anzahl)
	}
}
