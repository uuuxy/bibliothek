package druck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

const lmfNummer = "LMF-9783060158423-001"

// druckeBogenMit schickt einen Druckauftrag mit den Nummern, wie das Druck-Center ihn schickt.
func druckeBogenMit(t *testing.T, srv *Tueren, nummern ...string) *httptest.ResponseRecorder {
	t.Helper()
	items := make([]string, 0, len(nummern))
	for _, n := range nummern {
		items = append(items, `{"BarcodeID": "`+n+`", "Titel": "Natur und Technik", "Autor": ""}`)
	}
	payload := `{"formatId": "zweckform_l4760", "startPosition": 1, "isQR": false, "items": [` + strings.Join(items, ",") + `]}`
	rec := httptest.NewRecorder()
	srv.Etikettenbogen()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
	return rec
}

// vergebeNummern ruft die Tür der neuen Nummern; ohne kontoID ohne Anmeldung.
func vergebeNummern(t *testing.T, srv *Tueren, kontoID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/exemplare/neue-nummern", strings.NewReader(rumpf))
	if kontoID != "" {
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: kontoID, Rolle: auth.RolleMitarbeiter}))
	}
	rec := httptest.NewRecorder()
	srv.VergebeNeueNummern()(rec, req)
	return rec
}

// Der Weg im Druck-Center: Der Druck lehnt die LMF-Nummer ab und nennt sie, die Tür vergibt eine
// kurze, und mit ihr druckt der Bogen.
func TestNeueNummer_VomAbgelehntenDruckZumBogen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	titelID := testdaten.LegeTitelAn(t, pool, "Natur und Technik - Biologie 7/10", 0)
	testdaten.LegeExemplarAn(t, pool, titelID, "B-10001", true, "")
	testdaten.LegeExemplarAn(t, pool, titelID, lmfNummer, true, "")

	rec := druckeBogenMit(t, srv, "B-10001", lmfNummer, lmfNummer)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Druck mit LMF-Nummer: Status %d, erwartet 409", rec.Code)
	}
	var abgelehnt langeNummernAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &abgelehnt); err != nil {
		t.Fatalf("Ablehnung ist kein JSON (%v): %s", err, pdftest.ErsteBytes(rec.Body.Bytes(), 80))
	}
	if !slices.Equal(abgelehnt.Nummern, []string{lmfNummer}) || abgelehnt.Error == "" {
		t.Fatalf("Ablehnung %+v, erwartet die LMF-Nummer einmal und eine Meldung", abgelehnt)
	}

	rec = vergebeNummern(t, srv, bearbeiter, `{"barcode_ids": ["`+lmfNummer+`"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("neue Nummer: Status %d — %s", rec.Code, rec.Body.String())
	}
	var vergeben neueNummernAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &vergeben); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	neu := vergeben.Nummern[lmfNummer]
	if !regexp.MustCompile(`^B-[0-9]+$`).MatchString(neu) || len(vergeben.Nummern) != 1 {
		t.Fatalf("Antwort %v, erwartet zu %s eine Nummer B-…", vergeben.Nummern, lmfNummer)
	}

	rec = druckeBogenMit(t, srv, "B-10001", neu)
	if rec.Code != http.StatusOK {
		t.Fatalf("Druck mit der neuen Nummer: Status %d — %s", rec.Code, rec.Body.String())
	}
	if text := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n"); !strings.Contains(text, neu) {
		t.Errorf("die neue Nummer %s fehlt auf dem Bogen", neu)
	}
}

// Die Tür vergibt nur für LMF-Nummern eine neue, nur mit Anmeldung und nur für ein Exemplar, das
// die Nummer trägt. Eine abgelehnte Anfrage ändert nichts.
func TestVergebeNeueNummern_Ablehnungen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	titelID := testdaten.LegeTitelAn(t, pool, "Natur und Technik - Biologie 7/10", 0)
	testdaten.LegeExemplarAn(t, pool, titelID, "B-10001", true, "")
	testdaten.LegeExemplarAn(t, pool, titelID, lmfNummer, true, "")

	faelle := []struct {
		name    string
		kontoID string
		rumpf   string
		status  int
	}{
		{"ohne Anmeldung", "", `{"barcode_ids": ["` + lmfNummer + `"]}`, http.StatusUnauthorized},
		// Eine Nummer, die aufs Etikett passt, klebt schon am Buch; sie zu ersetzen machte ihr
		// Etikett unbrauchbar.
		{"eine kurze Nummer dabei", bearbeiter, `{"barcode_ids": ["` + lmfNummer + `", "B-10001"]}`, http.StatusBadRequest},
		{"keine Nummer", bearbeiter, `{"barcode_ids": []}`, http.StatusBadRequest},
		{"Nummer ohne Exemplar", bearbeiter, `{"barcode_ids": ["LMF-9783060158423-999"]}`, http.StatusNotFound},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if rec := vergebeNummern(t, srv, f.kontoID, f.rumpf); rec.Code != f.status {
				t.Errorf("Status %d, erwartet %d — %s", rec.Code, f.status, rec.Body.String())
			}
		})
	}

	var anzahl int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM buecher_exemplare WHERE barcode_id IN ($1, 'B-10001')`, lmfNummer).Scan(&anzahl); err != nil {
		t.Fatalf("Bestand lesen: %v", err)
	}
	if anzahl != 2 {
		t.Errorf("%d der beiden Exemplare tragen ihre Nummer noch, erwartet beide", anzahl)
	}
}
