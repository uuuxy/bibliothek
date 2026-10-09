package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/pdf"

	"github.com/google/uuid"
)

// POST /api/print/schueler-etiketten liefert den Klebebogen mit Name, Klasse und Barcode
// markierter Schüler. Den Erzeuger des PDFs prüft pdf/etikett_schueler_test.go; hier steht
// die Tür davor, die gemessen am 07.10.2026 kein Go-Test ausführte (2,3 %, OFFEN.md 5.10).
//
// Ihre Zusagen, je mit Gegenprobe: Der Bogen trägt, was in der Datenbank steht — die
// Anfrage nennt nur Kennungen, keine Namen. Ein unbekanntes Format wird abgewiesen statt
// still auf die Vorgabe gedreht (sonst merkt man es erst am verdruckten Bogen). Startposition,
// leere Auswahl, zu große Auswahl und gelöschte Schüler sind Auskünfte, kein Serverfehler.

func TestSchuelerEtiketten_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	_, rufe := protokollWelt(t, pool)

	bogen := func(t *testing.T, rumpf any) *httptest.ResponseRecorder {
		t.Helper()
		roh, err := json.Marshal(rumpf)
		if err != nil {
			t.Fatal(err)
		}
		return rufe(t, http.MethodPost, "/api/print/schueler-etiketten", string(roh))
	}
	erwarte400 := func(t *testing.T, rec *httptest.ResponseRecorder, steckt string) {
		t.Helper()
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), steckt) {
			t.Errorf("die Meldung nennt %q nicht: %s", steckt, rec.Body.String())
		}
	}

	schueler := seedSchueler(t, pool, "S-ETI-001", "Lea", "7b")

	t.Run("der Bogen trägt Name, Klasse und Barcode aus der Datenbank", func(t *testing.T) {
		rec := bogen(t, map[string]any{"schuelerIds": []string{schueler}})
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if typ := rec.Header().Get("Content-Type"); !strings.Contains(typ, "application/pdf") {
			t.Errorf("Content-Type %q, erwartet ein PDF", typ)
		}
		text := pdfText(t, rec.Body.Bytes())
		// Erwartet wird, was in der Zeile STEHT, nicht was der Test angelegt hat: Die Klasse
		// „7b" speichert der Trigger trg_schueler_klasse_vokabular in der Form der Schule.
		var nachname, vorname, klasse string
		if err := pool.QueryRow(t.Context(),
			`SELECT nachname, vorname, klasse FROM leser WHERE id = $1`, schueler).
			Scan(&nachname, &vorname, &klasse); err != nil {
			t.Fatal(err)
		}
		for _, erwartet := range []string{nachname + ", " + vorname, klasse, "S-ETI-001"} {
			if !strings.Contains(text, erwartet) {
				t.Errorf("%q steht nicht auf dem Bogen", erwartet)
			}
		}
	})

	// Der Testdruck des Ausweis-Designers: ein Beispiel-Etikett, kein echter Schüler.
	t.Run("Muster druckt das Beispiel, keinen Schüler", func(t *testing.T) {
		rec := bogen(t, map[string]any{"muster": true})
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		text := pdfText(t, rec.Body.Bytes())
		if !strings.Contains(text, pdf.MusterSchuelerEtikett.BarcodeID) {
			t.Errorf("das Muster-Etikett fehlt auf dem Bogen")
		}
		if strings.Contains(text, "S-ETI-001") {
			t.Error("der Musterdruck trägt einen echten Schüler")
		}
	})

	t.Run("unbekanntes Format: 400 statt der Vorgabe", func(t *testing.T) {
		erwarte400(t, bogen(t, map[string]any{"formatId": "erfunden_9999", "schuelerIds": []string{schueler}}),
			"erfunden_9999")
	})

	t.Run("Startposition außerhalb des Bogens: 400", func(t *testing.T) {
		erwarte400(t, bogen(t, map[string]any{"startPosition": 9999, "schuelerIds": []string{schueler}}),
			"Startposition")
	})

	t.Run("nichts markiert: 400", func(t *testing.T) {
		erwarte400(t, bogen(t, map[string]any{}), "keine Schüler markiert")
	})

	t.Run("mehr als die Grenze je Auftrag: 400", func(t *testing.T) {
		ids := make([]string, MaxSchuelerEtiketten+1)
		for i := range ids {
			ids[i] = uuid.NewString()
		}
		erwarte400(t, bogen(t, map[string]any{"schuelerIds": ids}),
			fmt.Sprintf("höchstens %d", MaxSchuelerEtiketten))
	})

	t.Run("gelöschte Schüler: 400 mit Hinweis", func(t *testing.T) {
		erwarte400(t, bogen(t, map[string]any{"schuelerIds": []string{uuid.NewString()}}),
			"inzwischen gelöscht")
	})
}
