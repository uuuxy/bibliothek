package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pdftest"
)

// Der Druck der Kontoauszüge folgt der Suche der Oberfläche (entschieden am 21.09.2026).
//
// Zwei Kommentare sicherten das seit dem 04.09.2026 zu („was auf dem Bildschirm steht,
// steht auf dem Papier"), die Tür kannte aber nur `klasse`: Wer „Müller" suchte und
// druckte, bekam alle Kontoauszüge. Ein einzelner Laufzettel ließ sich gar nicht
// nachdrucken — der Einzeldruck der Schülerakte hat keine Freigabezeile.
//
// Die Oberfläche schickt bei aktiver Suche die Kennungen der sichtbaren Zeilen. Der Server
// formuliert die Suche NICHT ein zweites Mal; er schneidet die Kennungen mit seiner
// eigenen Abgänger-Abfrage. Diese Tests halten am fertigen PDF fest, was daraus folgt.

// kontoauszugPDF ruft den Druck mit der gegebenen Query und liefert Antwort und Text.
func kontoauszugPDF(t *testing.T, srv *Server, query string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.GetGraduatesPDFHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger/pdf"+query, nil))
	if rec.Code != http.StatusOK {
		return rec, ""
	}
	return rec, strings.Join(pdftest.Texte(t, rec.Body.Bytes()), " ")
}

func TestAbgaengerDruck_FolgtDerAuswahl(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	anna := schuelerMitBuch(t, pool, "S-1", "Anna", "9H1")
	schuelerMitBuch(t, pool, "S-2", "Bea", "9H1")
	cem := schuelerMitBuch(t, pool, "S-3", "Cem", "10R1")
	// Kein Abgänger: 10G1 ist keine Abschlussklasse. Hat ein Buch, steht aber nicht in der
	// Liste — und darf auch über seine Kennung keine Seite bekommen.
	finn := schuelerMitBuch(t, pool, "S-6", "Finn", "10G1")

	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: saisonUhr}

	t.Run("ohne ids wie bisher: alle Abgänger", func(t *testing.T) {
		rec, text := kontoauszugPDF(t, srv, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d — %s", rec.Code, rec.Body.String())
		}
		for _, name := range []string{"Anna", "Bea", "Cem"} {
			if !strings.Contains(text, name) {
				t.Errorf("%s fehlt im Gesamtdruck", name)
			}
		}
		if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "Kontoauszuege_Abgaenger.pdf") {
			t.Errorf("Dateiname = %q", got)
		}
	})

	t.Run("eine Kennung: genau dieser Laufzettel", func(t *testing.T) {
		rec, text := kontoauszugPDF(t, srv, "?ids="+anna)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d — %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(text, "Anna") {
			t.Errorf("Anna fehlt, obwohl ausgewählt")
		}
		for _, name := range []string{"Bea", "Cem"} {
			if strings.Contains(text, name) {
				t.Errorf("%s steht im PDF, obwohl nicht ausgewählt — der Druck folgt der Auswahl nicht", name)
			}
		}
		if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "Kontoauszuege_Auswahl.pdf") {
			t.Errorf("Dateiname = %q, want Kontoauszuege_Auswahl.pdf", got)
		}
	})

	t.Run("die Auswahl ist ein Schnitt, keine zweite Tür", func(t *testing.T) {
		// Finn ist kein Abgänger. Seine Kennung neben der von Cem: gedruckt wird nur Cem.
		_, text := kontoauszugPDF(t, srv, "?ids="+cem+","+finn)
		if !strings.Contains(text, "Cem") || strings.Contains(text, "Finn") {
			t.Errorf("erwartet nur Cem; Text: %.200s", text)
		}
		// Nur Finn: nichts zu drucken — 404 wie bei einer leeren Liste, kein Kontoauszug
		// eines Schülers, den die Abgängerliste nicht zeigt.
		rec, _ := kontoauszugPDF(t, srv, "?ids="+finn)
		if rec.Code != http.StatusNotFound {
			t.Errorf("nur ein Nicht-Abgänger gewählt: Status = %d, want 404", rec.Code)
		}
	})

	t.Run("Klasse und Auswahl gelten zusammen", func(t *testing.T) {
		rec, _ := kontoauszugPDF(t, srv, "?klasse=10R1&ids="+anna)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Anna ist nicht in 10R1: Status = %d, want 404", rec.Code)
		}
	})

	t.Run("leere Auswahl ist ein Fehler, nicht 'alle'", func(t *testing.T) {
		// Eine Suche ohne Treffer schickte `ids=`. Als „keine Einengung" gelesen, druckte
		// „nichts gefunden" sämtliche Kontoauszüge.
		for _, query := range []string{"?ids=", "?ids=,", "?ids=%20"} {
			rec, _ := kontoauszugPDF(t, srv, query)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: Status = %d, want 400", query, rec.Code)
			}
		}
	})

	t.Run("keine Kennung ist 400, nicht 500", func(t *testing.T) {
		for _, query := range []string{"?ids=mueller", "?ids=" + anna + ",x", "?ids=urn:uuid:" + anna} {
			rec, _ := kontoauszugPDF(t, srv, query)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: Status = %d, want 400 — %s", query, rec.Code, rec.Body.String())
			}
		}
	})
}

// Außerhalb der Saison bleibt es beim 404 — auch mit einer Auswahl.
func TestAbgaengerDruck_AuswahlAendertDieSaisonNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	anna := schuelerMitBuch(t, pool, "S-1", "Anna", "9H1")

	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: winterUhr}
	rec, _ := kontoauszugPDF(t, srv, "?ids="+anna)
	if rec.Code != http.StatusNotFound {
		t.Errorf("im Oktober mit Auswahl: Status = %d, want 404", rec.Code)
	}
}
