package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Der Round-Trip der zweiten Kundennummer (Schülerbücherei, Migration 109) über die
// ECHTEN Türen: anlegen, lesen, ändern, wieder lesen.
//
// Warum am ganzen Weg und nicht am Handler allein: Die Bugklasse hier heißt „still
// verworfenes JSON-Feld" — ein Feld, das die Maske schickt, der Server annimmt und
// irgendwo zwischen Struct, SQL und Antwort verliert. Das fällt in keinem Unit-Test auf,
// weil jede Hälfte für sich stimmt; auffallen würde es erst, wenn der Händler die
// Rechnung für die Schülerbücherei auf das Lernmittel-Konto stellt.
//
// Der dritte Fall ist der wichtige: Eine Anfrage OHNE das Feld darf die hinterlegte
// Nummer NICHT löschen (nil = unverändert), ein leerer String schon (ausdrücklich
// gelöscht). Ohne diese Unterscheidung genügte ein Aufrufer, der das Feld nicht kennt,
// um die Nummer stillschweigend zu verlieren.

// lieferantAnlegenUeberHandler legt einen Lieferanten über POST /api/lieferanten an.
func lieferantAnlegenUeberHandler(t *testing.T, srv *Server, rumpf string) SupplierResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/lieferanten", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.CreateSupplierHandler()(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Anlegen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var resp SupplierResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	return resp
}

// lieferantAendernUeberHandler schickt PUT /api/lieferanten/{id} und liefert die Antwort.
func lieferantAendernUeberHandler(t *testing.T, srv *Server, id, rumpf string) SupplierResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/"+id, strings.NewReader(rumpf))
	req.SetPathValue("id", id)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.UpdateSupplierHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Ändern: Status %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp SupplierResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	return resp
}

// zweitnummerAusListe liest die Zweitnummer über GET /api/lieferanten — die Tür, aus der
// die Maske ihren Stand bezieht.
func zweitnummerAusListe(t *testing.T, srv *Server, id string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ListSuppliersHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/lieferanten", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Liste: Status %d: %s", rec.Code, rec.Body.String())
	}
	var liste []SupplierResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatalf("Liste lesen: %v", err)
	}
	for _, l := range liste {
		if l.ID == id {
			return l.KundennummerSchultraeger
		}
	}
	t.Fatalf("Lieferant %s steht nicht in der Liste", id)
	return ""
}

func TestLieferantZweitnummer_RoundTripUeberDieEchtenTueren(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	angelegt := lieferantAnlegenUeberHandler(t, srv, `{"name":"Zweitnummer-Haendler","email":"zwei@example.invalid",
		"customerNumber":"K-EINS","kundennummer_schultraeger":" BIB-ZWEI "}`)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM lieferanten WHERE id = $1`, angelegt.ID); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	// 1. Anlegen: getrimmt gespeichert und in der Antwort.
	if angelegt.KundennummerSchultraeger != "BIB-ZWEI" {
		t.Errorf("Antwort nach dem Anlegen: %q, want %q", angelegt.KundennummerSchultraeger, "BIB-ZWEI")
	}
	if got := zweitnummerAusListe(t, srv, angelegt.ID); got != "BIB-ZWEI" {
		t.Errorf("Liste nach dem Anlegen: %q, want %q", got, "BIB-ZWEI")
	}

	// 2. Ändern MIT Feld: neuer Wert gilt.
	geaendert := lieferantAendernUeberHandler(t, srv, angelegt.ID,
		`{"name":"Zweitnummer-Haendler","email":"zwei@example.invalid","customerNumber":"K-EINS","kundennummer_schultraeger":"BIB-DREI"}`)
	if geaendert.KundennummerSchultraeger != "BIB-DREI" {
		t.Errorf("Antwort nach dem Ändern: %q, want %q", geaendert.KundennummerSchultraeger, "BIB-DREI")
	}
	if got := zweitnummerAusListe(t, srv, angelegt.ID); got != "BIB-DREI" {
		t.Errorf("Liste nach dem Ändern: %q, want %q", got, "BIB-DREI")
	}

	// 3. Ändern OHNE Feld: die Nummer bleibt. Das ist der Kern — ein Aufrufer, der das
	//    Feld nicht kennt (alter Client, anderes Werkzeug), darf sie nicht löschen.
	ohneFeld := lieferantAendernUeberHandler(t, srv, angelegt.ID,
		`{"name":"Zweitnummer-Haendler NEU","email":"zwei@example.invalid","customerNumber":"K-EINS"}`)
	if ohneFeld.KundennummerSchultraeger != "BIB-DREI" {
		t.Errorf("Antwort ohne Feld: %q — die Antwort spiegelt die Eingabe statt des Stands", ohneFeld.KundennummerSchultraeger)
	}
	if got := zweitnummerAusListe(t, srv, angelegt.ID); got != "BIB-DREI" {
		t.Errorf("OHNE Feld wurde die Zweitnummer gelöscht: %q, want %q", got, "BIB-DREI")
	}

	// 4. Leerer String: ausdrücklich gelöscht — die Unterscheidung muss in BEIDE
	//    Richtungen funktionieren, sonst ist die Nummer nicht mehr wegzubekommen.
	geleert := lieferantAendernUeberHandler(t, srv, angelegt.ID,
		`{"name":"Zweitnummer-Haendler NEU","email":"zwei@example.invalid","customerNumber":"K-EINS","kundennummer_schultraeger":""}`)
	if geleert.KundennummerSchultraeger != "" {
		t.Errorf("Antwort nach dem Leeren: %q, want leer", geleert.KundennummerSchultraeger)
	}
	if got := zweitnummerAusListe(t, srv, angelegt.ID); got != "" {
		t.Errorf("Leeren hat nicht gewirkt: %q", got)
	}

	// 5. Und die Bestellung nimmt danach wieder die erste Nummer (KundennummerFuer).
	var inDb string
	if err := pool.QueryRow(ctx,
		`SELECT kundennummer_schultraeger FROM lieferanten WHERE id = $1`, angelegt.ID).Scan(&inDb); err != nil {
		t.Fatal(err)
	}
	if inDb != "" {
		t.Errorf("Datenbank: %q, want leer", inDb)
	}
}

// Ein unbekannter Lieferant bleibt 404 — das RETURNING darf den Fall nicht zum 500 machen.
func TestLieferantZweitnummer_UnbekannterLieferantIst404(t *testing.T) {
	pool := pgTestPool(t)
	srv := &Server{DB: &db.Database{Pool: pool}}
	id := "00000000-0000-0000-0000-000000000000"
	req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/"+id, strings.NewReader(
		`{"name":"x","email":"x@example.invalid","customerNumber":"K","kundennummer_schultraeger":"B"}`))
	req.SetPathValue("id", id)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.UpdateSupplierHandler()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, want 404: %s", rec.Code, rec.Body.String())
	}
}
