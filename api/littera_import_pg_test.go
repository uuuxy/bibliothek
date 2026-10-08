package api

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die beiden Türen, über die der Bestand der Schule ins Programm kommt:
// POST /api/import/littera (Katalogisat-XML, CSV, XLSX) und POST
// /api/admin/import-bestand (die endgültige Bestands-CSV). Vor dem Echtstart läuft hier
// der ganze Katalog durch; gemessen am 07.10.2026 führte kein Go-Test 0,7 % der Datei aus
// (OFFEN.md 5.10).
//
// Geprüft wird über den ganzen Router mit Sitzung, nicht am nackten Handler: Die Tür hängt
// an Recht (manage_inventory), CSRF und der Mehrteil-Form, und genau dort saßen in diesem
// Projekt schon Fehler, die der isolierte Handler nicht zeigte.
//
// Je Zusicherung eine Gegenprobe: Was abgewiesen wird, darf nichts geschrieben haben.

// ladeImport schickt eine Datei als Mehrteil-Form an die Tür und liefert die Antwort.
func ladeImport(t *testing.T, rufeRoh func(t *testing.T, methode, pfad string, rumpf string, typ string) *httptest.ResponseRecorder,
	pfad, dateiname string, inhalt []byte) *httptest.ResponseRecorder {
	t.Helper()
	var koerper strings.Builder
	mw := multipart.NewWriter(&koerper)
	teil, err := mw.CreateFormFile("file", dateiname)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teil.Write(inhalt); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return rufeRoh(t, http.MethodPost, pfad, koerper.String(), mw.FormDataContentType())
}

// importAntwort liest die Zähler der Antwort.
func importAntwort(t *testing.T, rec *httptest.ResponseRecorder) LitteraImportResponse {
	t.Helper()
	var a LitteraImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("Antwort unlesbar (%s): %v", rec.Body.String(), err)
	}
	return a
}

func TestLitteraImport_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	adminID, _, rufeRoh := importWelt(t, pool)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion IN ('LITTERA_IMPORT', 'BESTAND_IMPORT')`)
	})

	// zaehle liefert Titel und Exemplare, die dieser Test angelegt haben könnte.
	zaehle := func(t *testing.T) (titel, exemplare int) {
		t.Helper()
		return zaehleZeilen(t, pool, `SELECT count(*) FROM buecher_titel WHERE titel LIKE 'E2E-Imp-%'`),
			zaehleZeilen(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE barcode_id LIKE 'IMP-%'`)
	}
	// spuren zählt die Protokolleinträge der Importe.
	spuren := func(t *testing.T, aktion string) int {
		t.Helper()
		return zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = $1 AND admin_id = $2`, aktion, adminID)
	}

	t.Run("CSV: zwei Titel mit je einem Exemplar", func(t *testing.T) {
		csv := "Titel;Verfasser;ISBN;Exemplarnummer\n" +
			"E2E-Imp-Mathe 7;Musterfrau;9783111111111;IMP-A1\n" +
			"E2E-Imp-Deutsch 8;Mustermann;9783222222222;IMP-A2\n"
		rec := ladeImport(t, rufeRoh, "/api/import/littera", "littera.csv", []byte(csv))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		a := importAntwort(t, rec)
		if a.Type != "csv" {
			t.Errorf("Art %q, erwartet csv", a.Type)
		}
		if a.NewTitles != 2 || a.ImportedCopies != 2 {
			t.Errorf("gemeldet: %d Titel, %d Exemplare — erwartet 2 und 2", a.NewTitles, a.ImportedCopies)
		}
		// Die Antwort ist nicht der Beleg: nachgezählt in der Datenbank.
		if titel, exemplare := zaehle(t); titel != 2 || exemplare != 2 {
			t.Errorf("in der Datenbank: %d Titel, %d Exemplare — erwartet 2 und 2", titel, exemplare)
		}
		if n := spuren(t, "LITTERA_IMPORT"); n != 1 {
			t.Errorf("%d Protokolleinträge, erwartet 1", n)
		}
	})

	t.Run("CSV ohne Pflichtspalte Barcode: 400 und nichts geschrieben", func(t *testing.T) {
		vorherT, vorherE := zaehle(t)
		rec := ladeImport(t, rufeRoh, "/api/import/littera", "littera.csv",
			[]byte("Titel;Verfasser\nE2E-Imp-Ohne Barcode;Niemand\n"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "barcode") {
			t.Errorf("die Meldung nennt die fehlende Spalte nicht: %s", rec.Body.String())
		}
		if titel, exemplare := zaehle(t); titel != vorherT || exemplare != vorherE {
			t.Errorf("eine abgewiesene Datei hat geschrieben: Titel %d→%d, Exemplare %d→%d",
				vorherT, titel, vorherE, exemplare)
		}
	})

	t.Run("XLSX: dieselben Spalten aus einer Arbeitsmappe", func(t *testing.T) {
		xlsx := baueXlsx(t, map[string][][]any{
			"Bestand": {
				{"Titel", "Verfasser", "ISBN", "Exemplarnummer"},
				{"E2E-Imp-Physik 9", "Musterfrau", "9783333333333", "IMP-B1"},
			},
		})
		rec := ladeImport(t, rufeRoh, "/api/import/littera", "littera.xlsx", xlsx)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if a := importAntwort(t, rec); a.Type != "xlsx" || a.NewTitles != 1 {
			t.Errorf("Art %q, %d neue Titel — erwartet xlsx und 1", a.Type, a.NewTitles)
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'IMP-B1'`); n != 1 {
			t.Errorf("das Exemplar aus der Arbeitsmappe steht %d mal in der Datenbank, erwartet 1", n)
		}
	})

	// Der Weg aus pkg/xlsxgrenze am Live-Pfad: Eine Datei, die excelize nicht lesen kann,
	// ist eine Auskunft (400) und kein Serverfehler.
	//
	// Was dieser Fall NICHT belegt: Keine der vier Eingaben löst einen Absturz der
	// Bibliothek aus, und ob der OLE-Container VOR excelize abgewiesen wird, ist von hier
	// aus nicht zu sehen — ohne die Abweisung lehnt excelize ihn selbst ab, auch mit 400
	// (am 08.10.2026 als Rot-Probe gefahren, sie blieb grün). Beides steht im Test des
	// Pakets (pkg/xlsxgrenze/mitmappe_test.go), wo der Absturz herstellbar und der
	// Fehlerwert unterscheidbar ist. Hier steht, dass die Tür nicht mit 500 antwortet.
	t.Run("XLSX unlesbar: 400, kein Serverfehler", func(t *testing.T) {
		vorherT, vorherE := zaehle(t)
		faelle := map[string][]byte{
			"kein Zip":           []byte("das ist keine Arbeitsmappe"),
			"OLE-Container":      {0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0, 0, 0, 0, 0, 0, 0},
			"leere Datei":        {},
			"Zip ohne Arbeitsm.": []byte("PK\x03\x04 nur der Kopf, nichts dahinter"),
		}
		for name, inhalt := range faelle {
			rec := ladeImport(t, rufeRoh, "/api/import/littera", "littera.xlsx", inhalt)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: Status %d, erwartet 400: %s", name, rec.Code, rec.Body.String())
			}
		}
		if titel, exemplare := zaehle(t); titel != vorherT || exemplare != vorherE {
			t.Errorf("eine unlesbare Datei hat geschrieben: Titel %d→%d, Exemplare %d→%d",
				vorherT, titel, vorherE, exemplare)
		}
	})

	// Ein XML, das kein Katalogisat ist (etwa ein Schlagwort-Export), ist ein Formatfehler
	// des Nutzers: 400 mit einem Satz, der sagt, was erwartet wird — nicht 500.
	t.Run("XML ohne Katalogisat: 400 mit Hinweis", func(t *testing.T) {
		rec := ladeImport(t, rufeRoh, "/api/import/littera", "schlagworte.xml",
			[]byte(`<?xml version="1.0" encoding="UTF-8"?><Schlagworte><Wort>Algebra</Wort></Schlagworte>`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(strings.ToLower(rec.Body.String()), "katalogisat") {
			t.Errorf("die Meldung sagt nicht, was erwartet wird: %s", rec.Body.String())
		}
	})

	t.Run("ohne Datei: 400", func(t *testing.T) {
		rec := rufeRoh(t, http.MethodPost, "/api/import/littera", "", "multipart/form-data; boundary=xyz")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestBestandImport_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	adminID, _, rufeRoh := importWelt(t, pool)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion = 'BESTAND_IMPORT'`)
	})

	t.Run("CSV mit Komma: Titel und Exemplar stehen in der Datenbank", func(t *testing.T) {
		csv := "Titel,Verfasser,Exemplarnummer,Zustand\n" +
			"E2E-Imp-Bestand Chemie,Musterfrau,IMP-C1,gut\n"
		rec := ladeImport(t, rufeRoh, "/api/admin/import-bestand", "bestand.csv", []byte(csv))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'IMP-C1'`); n != 1 {
			t.Errorf("das Exemplar steht %d mal in der Datenbank, erwartet 1", n)
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'BESTAND_IMPORT' AND admin_id = $1`,
			adminID); n != 1 {
			t.Errorf("%d Protokolleinträge, erwartet 1", n)
		}
	})

	// Die Tür nimmt ausdrücklich nur CSV. Eine Arbeitsmappe wird an der Endung abgewiesen,
	// bevor sie gelesen wird — anders als bei der Littera-Tür daneben, und das ist Absicht.
	t.Run("Nur CSV: eine Arbeitsmappe wird abgewiesen", func(t *testing.T) {
		rec := ladeImport(t, rufeRoh, "/api/admin/import-bestand", "bestand.xlsx",
			baueXlsx(t, map[string][][]any{"B": {{"Titel", "Exemplarnummer"}, {"X", "IMP-D1"}}}))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		// Der Wortlaut DIESER Abweisung, nicht irgendeiner mit „CSV" darin: Ohne die Prüfung
		// der Endung läuft die Arbeitsmappe in den CSV-Leser und scheitert dort mit
		// „CSV konnte nicht gelesen werden" — auch ein 400, aber aus einem anderen Grund.
		// Die Rot-Probe am 08.10.2026 war mit der früheren, lockeren Prüfung grün.
		if !strings.Contains(rec.Body.String(), "nur CSV-Dateien") {
			t.Errorf("die Meldung ist nicht die der Endungs-Prüfung: %s", rec.Body.String())
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'IMP-D1'`); n != 0 {
			t.Errorf("die abgewiesene Arbeitsmappe hat %d Exemplare geschrieben", n)
		}
	})

	t.Run("CSV nur mit Kopfzeile: 400, die Meldung nennt die fehlenden Datenzeilen", func(t *testing.T) {
		rec := ladeImport(t, rufeRoh, "/api/admin/import-bestand", "bestand.csv",
			[]byte("Titel,Exemplarnummer\n"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Datenzeilen") {
			t.Errorf("die Meldung nennt den Grund nicht: %s", rec.Body.String())
		}
	})

	t.Run("CSV ohne Pflichtspalte Titel: 400", func(t *testing.T) {
		rec := ladeImport(t, rufeRoh, "/api/admin/import-bestand", "bestand.csv",
			[]byte("Verfasser,Exemplarnummer\nMusterfrau,IMP-E1\n"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(strings.ToLower(rec.Body.String()), "titel") {
			t.Errorf("die Meldung nennt die fehlende Spalte nicht: %s", rec.Body.String())
		}
	})
}

// importWelt legt ein Admin-Konto mit Sitzung an und liefert einen Aufruf über den ganzen
// Router — dieselbe Bauart wie protokollWelt, nur mit frei wählbarem Inhaltstyp für die
// Mehrteil-Form.
func importWelt(t *testing.T, pool *pgxpool.Pool) (adminID string, sitzung string,
	rufeRoh func(t *testing.T, methode, pfad, rumpf, typ string) *httptest.ResponseRecorder) {
	t.Helper()
	adminID, sitzung, router := routerMitSitzung(t, pool, "import-tuer@example.org", "Imp", "Ort")
	return adminID, sitzung, func(t *testing.T, methode, pfad, rumpf, typ string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", typ)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, mitCSRF(req))
		return rec
	}
}
