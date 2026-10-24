package inventur

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// exportSpalten sind die Spalten der Export-Abfrage in ihrer Reihenfolge.
var exportSpalten = []string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand", "signatur", "schlagworte", "topf", "standort", "lmf_barcode"}

func TestCSVExport(t *testing.T) {
	t.Run("successful export", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "schultraeger", "Bibliothek, Regal 3B", "LMF-9783060158423-001").
			AddRow("=Buch 2", "", "", "", 0, "", "", "", "", "", "", "-Keller", "") // Formelschutz: Titel und Standort

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		if ergebnis.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, erwartet %v", ergebnis.StatusCode, http.StatusOK)
		}
		if got := ergebnis.Header.Get("Content-Type"); got != "text/csv; charset=utf-8" {
			t.Errorf("Content-Type = %q, erwartet %q", got, "text/csv; charset=utf-8")
		}
		if got := ergebnis.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment; filename=\"bestand_export_") {
			t.Errorf("Content-Disposition = %q, erwartet darin %q", got, "attachment; filename=\"bestand_export_")
		}

		rumpf, err := io.ReadAll(ergebnis.Body)
		if err != nil {
			t.Fatalf("io.ReadAll: %v", err)
		}

		if !bytes.HasPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}) {
			t.Errorf("Antwort beginnt mit %q, erwartet das BOM für UTF-8", rumpf[:min(len(rumpf), 3)])
		}

		// Das BOM für den Vergleich entfernen.
		rumpfText := string(bytes.TrimPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}))
		textZeilen := strings.Split(strings.TrimSpace(rumpfText), "\n")
		if len(textZeilen) != 3 {
			t.Fatalf("len(strings.Split(strings.TrimSpace(rumpfText), \"\\n\")) = %d, erwartet %d", len(textZeilen), 3)
		}
		if textZeilen[0] != "Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand;Signatur;Schlagworte;Eigentum;Standort;LMF-Nummer" {
			t.Errorf("textZeilen[0] = %q, erwartet %q", textZeilen[0], "Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand;Signatur;Schlagworte;Eigentum;Standort;LMF-Nummer")
		}
		if textZeilen[1] != "Buch 1;Autor 1;Verlag 1;'1234567890;2021;Kategorie 1;BC1;Gut;Ma 8;Algebra | Geometrie;Schulträger;Bibliothek, Regal 3B;LMF-9783060158423-001" {
			t.Errorf("textZeilen[1] = %q, erwartet %q", textZeilen[1], "Buch 1;Autor 1;Verlag 1;'1234567890;2021;Kategorie 1;BC1;Gut;Ma 8;Algebra | Geometrie;Schulträger;Bibliothek, Regal 3B;LMF-9783060158423-001")
		}
		if textZeilen[2] != "'=Buch 2;;;;;;;;;;;'-Keller;" {
			t.Errorf("textZeilen[2] = %q, erwartet %q", textZeilen[2], "'=Buch 2;;;;;;;;;;;'-Keller;")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("database error before kopf sends 500", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnError(errors.New("db error"))

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		if ergebnis.StatusCode != http.StatusInternalServerError {
			t.Errorf("Status = %v, erwartet %v", ergebnis.StatusCode, http.StatusInternalServerError)
		}

		rumpf, leseFehler := io.ReadAll(ergebnis.Body)
		if leseFehler != nil {
			t.Fatalf("io.ReadAll: %v", leseFehler)
		}
		if !strings.Contains(string(rumpf), "Ein interner Serverfehler ist aufgetreten") {
			t.Errorf("string(rumpf) = %q, erwartet darin %q", string(rumpf), "Ein interner Serverfehler ist aufgetreten")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("database error after kopf breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "", "", "land", "", "").
			RowError(0, errors.New("connection lost"))

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		// Der Status bleibt 200, weil die Kopfzeilen schon gesendet sind.
		if ergebnis.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, erwartet %v", ergebnis.StatusCode, http.StatusOK)
		}

		rumpf, leseFehler := io.ReadAll(ergebnis.Body)
		if leseFehler != nil {
			t.Fatalf("io.ReadAll: %v", leseFehler)
		}

		if !bytes.HasPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}) {
			t.Errorf("Antwort beginnt mit %q, erwartet das BOM für UTF-8", rumpf[:min(len(rumpf), 3)])
		}

		if got := ergebnis.Header.Get("Content-Type"); got != "text/csv; charset=utf-8" {
			t.Errorf("Content-Type = %q, erwartet %q", got, "text/csv; charset=utf-8")
		}
		// Die Abfrage scheitert an der ersten Zeile des Bestands; die Kopfzeile der Datei steht
		// dann schon in der Antwort.
		kopfzeile, _, _ := strings.Cut(string(bytes.TrimPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF})), "\n")
		if want := "Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand;Signatur;Schlagworte;Eigentum;Standort;LMF-Nummer"; kopfzeile != want {
			t.Errorf("erste Zeile = %q, erwartet die Kopfzeile %q", kopfzeile, want)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}
