package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

// TestBuchEingabeDecodesAlleFelder sichert die Regression ab, bei der
// jahrgangVon/jahrgangBis nicht im Request-Struct standen: der JSON-Decoder verwarf
// die Felder still, der Handler schrieb 0/0 in die DB und meldete trotzdem 200
// ("erfolgreich gespeichert", aber der Klassenbereich 11–13 war weg). Der Test
// dekodiert exakt die Nutzlast, die das Frontend sendet, und prüft, dass alle vom
// Formular gebundenen Felder ankommen.
func TestBuchEingabeDecodesAlleFelder(t *testing.T) {
	// Feldnamen entsprechen den json-Tags aus dem Frontend (BuchEingabefelder*.svelte).
	body := `{
		"isbn": "978-3-16-148410-0",
		"title": "Testtitel",
		"jahrgangVon": 11,
		"jahrgangBis": 13,
		"untertitel": "Ein Untertitel",
		"verlag": "Testverlag",
		"erscheinungsjahr": 2024,
		"beschreibung": "Beschreibungstext"
	}`

	var eingabe BuchEingabe
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&eingabe); err != nil {
		t.Fatalf("Decode fehlgeschlagen: %v", err)
	}

	if eingabe.JahrgangVon != 11 {
		t.Errorf("JahrgangVon: erwartet 11, bekam %d", eingabe.JahrgangVon)
	}
	if eingabe.JahrgangBis != 13 {
		t.Errorf("JahrgangBis: erwartet 13, bekam %d", eingabe.JahrgangBis)
	}
	if eingabe.Untertitel != "Ein Untertitel" {
		t.Errorf("Untertitel: erwartet 'Ein Untertitel', bekam %q", eingabe.Untertitel)
	}
	if eingabe.Verlag != "Testverlag" {
		t.Errorf("Verlag: erwartet 'Testverlag', bekam %q", eingabe.Verlag)
	}
	if eingabe.Erscheinungsjahr != 2024 {
		t.Errorf("Erscheinungsjahr: erwartet 2024, bekam %d", eingabe.Erscheinungsjahr)
	}
	if eingabe.Beschreibung != "Beschreibungstext" {
		t.Errorf("Beschreibung: erwartet 'Beschreibungstext', bekam %q", eingabe.Beschreibung)
	}
}

func TestBereinigeUndValidiereBuchEingabe(t *testing.T) {
	tests := []struct {
		name        string
		eingabe     BuchEingabe
		wantErr     bool
		errMsg      string
		wantEingabe *BuchEingabe
	}{
		{
			name: "Valid input",
			eingabe: BuchEingabe{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 5,
				Bestand:      zeigerAuf(10),
				Titel:        " Test Titel ",
			},
			wantErr: false,
		},
		{
			name: "Empty ISBN",
			eingabe: BuchEingabe{
				ISBN:         "",
				KlassenStufe: 5,
				Bestand:      zeigerAuf(10),
			},
			wantErr: true,
			errMsg:  "isbn darf nicht leer sein",
		},
		{
			name: "Invalid ISBN format",
			eingabe: BuchEingabe{
				ISBN:         "123",
				KlassenStufe: 5,
				Bestand:      zeigerAuf(10),
			},
			wantErr: true,
			errMsg:  "ungültiges ISBN-Format",
		},
		{
			name: "Negative gradeLevel",
			eingabe: BuchEingabe{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: -1,
				Bestand:      zeigerAuf(10),
			},
			wantErr: true,
			errMsg:  "gradeLevel muss zwischen 0 und 13 sein",
		},
		{
			name: "gradeLevel too high",
			eingabe: BuchEingabe{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 14,
				Bestand:      zeigerAuf(10),
			},
			wantErr: true,
			errMsg:  "gradeLevel muss zwischen 0 und 13 sein",
		},
		{
			name: "Negative stock",
			eingabe: BuchEingabe{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 5,
				Bestand:      zeigerAuf(-1),
			},
			wantErr: true,
			errMsg:  "stock muss >= 0 sein",
		},
		{
			name: "Trims spaces from fields",
			eingabe: BuchEingabe{
				ISBN:         "  978-3-16-148410-0  ",
				Titel:        "  Titel  ",
				Autor:        "  Autor  ",
				CoverURL:     "  URL  ",
				Fach:         "  Fach  ",
				Schulzweig:   "  Schulzweig  ",
				Medientyp:    "  Medientyp  ",
				Untertitel:   "  Untertitel  ",
				Verlag:       "  Verlag  ",
				Beschreibung: "  Beschreibung  ",
			},
			wantErr: false,
			wantEingabe: &BuchEingabe{
				ISBN:         "978-3-16-148410-0",
				Titel:        "Titel",
				Autor:        "Autor",
				CoverURL:     "URL",
				Fach:         "Fach",
				Schulzweig:   "Schulzweig",
				Medientyp:    "Medientyp",
				Untertitel:   "Untertitel",
				Verlag:       "Verlag",
				Beschreibung: "Beschreibung",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bereinigeUndValidiereBuchEingabe(&tt.eingabe)
			if (err != nil) != tt.wantErr {
				t.Errorf("bereinigeUndValidiereBuchEingabe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("bereinigeUndValidiereBuchEingabe() expected error message %q, got %q", tt.errMsg, err.Error())
			}
			if !tt.wantErr && tt.wantEingabe != nil {
				assert.Equal(t, *tt.wantEingabe, tt.eingabe)
			}
		})
	}
}

// zeigerAuf macht aus einer geschriebenen Zahl den Zeiger, den BuchEingabe.Bestand
// verlangt — "nicht mitgeschickt" ist dort ein eigener Fall (nil).
func zeigerAuf(n int) *int { return &n }

// Die drei folgenden Fälle sind das Gate zum Raster-Fund vom 23.08.2026: Beim ÄNDERN
// bedeutet ein fehlender oder geleerter Wert etwas anderes als beim Anlegen.
func TestBearbeiteBuchAktualisieren_LeerHeisstBeimAendernNichtVorgabe(t *testing.T) {
	t.Run("fehlender Bestand lässt die Exemplare unangetastet", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()

		// Kein einziger syncBookStock-Aufruf darf folgen: Das Feld "stock" fehlt im Rumpf.
		erwarteFachBekannt(mock, "Mathe")
		mock.ExpectBegin()
		beliebig := make([]any, 19)
		for i := range beliebig {
			beliebig[i] = pgxmock.AnyArg()
		}
		mock.ExpectExec("UPDATE buecher_titel").WithArgs(beliebig...).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		handler := &APIHandler{repo: NewBookRepository(mock), metadaten: stummerMetadatenClient()}
		rumpf := `{"isbn":"9783161484100","title":"Titel","author":"Autor","subject":"Mathe"}`
		w := ruf(t, handler, "/api/books/book-1", rumpf)

		if w.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200 — Rumpf: %s", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("fehlender Bestand hat trotzdem Exemplare angefasst: %v", err)
		}
	})

	t.Run("leerer Titel wird abgelehnt statt ersetzt", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()

		handler := &APIHandler{repo: NewBookRepository(mock), metadaten: stummerMetadatenClient()}
		rumpf := `{"isbn":"9783161484100","title":"","author":"Autor","subject":"Mathe"}`
		w := ruf(t, handler, "/api/books/book-1", rumpf)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400 — ein geleerter Titel darf nicht still zu "+
				"\"Unbekannter Titel\" werden. Rumpf: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "Unbekannter") {
			t.Error("die Antwort nennt den Platzhalter — er ist noch im Spiel")
		}
	})

	t.Run("leerer Autor wird abgelehnt statt ersetzt", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()

		handler := &APIHandler{repo: NewBookRepository(mock), metadaten: stummerMetadatenClient()}
		rumpf := `{"isbn":"9783161484100","title":"Titel","author":"","subject":"Mathe"}`
		w := ruf(t, handler, "/api/books/book-1", rumpf)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400 — Rumpf: %s", w.Code, w.Body.String())
		}
	})
}

// ruf schickt einen PUT an den Aktualisieren-Handler.
func ruf(t *testing.T, handler *APIHandler, pfad, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, pfad, strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BearbeiteBuchAktualisieren(w, req)
	return w
}

// stummerMetadatenClient liefert einen Client, dessen Nachschlag nichts findet — der
// Nachschlag darf die Aussage der Tests nicht verändern.
func stummerMetadatenClient() *MetadatenClient {
	return &MetadatenClient{httpClient: &http.Client{
		Transport: &mockTransport{roundTripFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
		}},
	}}
}
