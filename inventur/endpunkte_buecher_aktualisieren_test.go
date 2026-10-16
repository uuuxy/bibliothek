package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

// TestTitelEingabe_LiestAlleFelder sichert die Regression ab, bei der
// jahrgangVon/jahrgangBis nicht im Request-Struct standen: der JSON-Decoder verwarf
// die Felder still, der Handler schrieb 0/0 in die DB und meldete trotzdem 200
// ("erfolgreich gespeichert", aber der Klassenbereich 11–13 war weg). Der Test
// dekodiert exakt die Nutzlast, die das Frontend sendet, und prüft, dass alle vom
// Formular gebundenen Felder ankommen.
func TestTitelEingabe_LiestAlleFelder(t *testing.T) {
	// Feldnamen entsprechen den json-Tags aus dem Frontend (BuchEingabefelder*.svelte).
	rumpf := `{
		"isbn": "978-3-16-148410-0",
		"title": "Testtitel",
		"jahrgangVon": 11,
		"jahrgangBis": 13,
		"untertitel": "Ein Untertitel",
		"verlag": "Testverlag",
		"erscheinungsjahr": 2024
	}`

	var eingabe titelEingabe
	if err := json.NewDecoder(strings.NewReader(rumpf)).Decode(&eingabe); err != nil {
		t.Fatalf("Decode fehlgeschlagen: %v", err)
	}

	if eingabe.JahrgangVon != 11 {
		t.Errorf("JahrgangVon = %d, erwartet 11", eingabe.JahrgangVon)
	}
	if eingabe.JahrgangBis != 13 {
		t.Errorf("JahrgangBis = %d, erwartet 13", eingabe.JahrgangBis)
	}
	if eingabe.Untertitel != "Ein Untertitel" {
		t.Errorf("Untertitel = %q, erwartet 'Ein Untertitel'", eingabe.Untertitel)
	}
	if eingabe.Verlag != "Testverlag" {
		t.Errorf("Verlag = %q, erwartet 'Testverlag'", eingabe.Verlag)
	}
	if eingabe.Erscheinungsjahr != 2024 {
		t.Errorf("Erscheinungsjahr = %d, erwartet 2024", eingabe.Erscheinungsjahr)
	}
}

func TestBereinigeTitelEingabe(t *testing.T) {
	faelle := []struct {
		name        string
		eingabe     titelEingabe
		wantErr     bool
		errMsg      string
		wantEingabe *titelEingabe
	}{
		{
			name: "Valid input",
			eingabe: titelEingabe{
				ISBN:    "978-3-16-148410-0",
				Bestand: zeigerAuf(10),
				Titel:   " Test Titel ",
			},
			wantErr: false,
		},
		// Die ISBN prüft der Schreibpfad gegen den gespeicherten Titel (pruefeAenderung): Ein
		// Titel ohne ISBN oder mit einer Nummer, die keine ISBN ist, bleibt speicherbar.
		{
			name: "Ohne ISBN",
			eingabe: titelEingabe{
				ISBN:    "",
				Bestand: zeigerAuf(10),
			},
			wantErr: false,
		},
		{
			name: "Nummer ohne ISBN-Form",
			eingabe: titelEingabe{
				ISBN:    "123",
				Bestand: zeigerAuf(10),
			},
			wantErr: false,
		},
		{
			name: "Negative stock",
			eingabe: titelEingabe{
				ISBN:    "978-3-16-148410-0",
				Bestand: zeigerAuf(-1),
			},
			wantErr: true,
			errMsg:  "Der Bestand darf nicht negativ sein",
		},
		// chk_listenpreis_nonneg lehnte ab, bevor die Tür prüfte: Beim Ändern
		// kam eine 500 „buch konnte nicht aktualisiert werden", ohne zu sagen, was erlaubt ist.
		{
			name: "Negativer Listenpreis",
			eingabe: titelEingabe{
				ISBN:        "978-3-16-148410-0",
				Listenpreis: zeigerAufPreis(-0.01),
			},
			wantErr: true,
			errMsg:  "Der Listenpreis darf nicht negativ sein. Leer lassen, wenn er unbekannt ist.",
		},
		{
			name: "Listenpreis 0 ist erlaubt",
			eingabe: titelEingabe{
				ISBN:        "978-3-16-148410-0",
				Listenpreis: zeigerAufPreis(0),
			},
			wantErr: false,
		},
		{
			name: "Trims spaces from fields",
			eingabe: titelEingabe{
				ISBN:       "  978-3-16-148410-0  ",
				Titel:      "  Titel  ",
				Autor:      "  Autor  ",
				CoverURL:   "  URL  ",
				Fach:       "  Fach  ",
				Schulzweig: "  Schulzweig  ",
				Medientyp:  "  Medientyp  ",
				Untertitel: "  Untertitel  ",
				Verlag:     "  Verlag  ",
			},
			wantErr: false,
			wantEingabe: &titelEingabe{
				ISBN:       "978-3-16-148410-0",
				Titel:      "Titel",
				Autor:      "Autor",
				CoverURL:   "URL",
				Fach:       "Fach",
				Schulzweig: "Schulzweig",
				Medientyp:  "Medientyp",
				Untertitel: "Untertitel",
				Verlag:     "Verlag",
			},
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			err := bereinigeTitelEingabe(&f.eingabe)
			if (err != nil) != f.wantErr {
				t.Errorf("bereinigeTitelEingabe(…) = Fehler %v, erwartet Fehler: %v", err, f.wantErr)
				return
			}
			if f.wantErr && err.Error() != f.errMsg {
				t.Errorf("bereinigeTitelEingabe(…) = Fehler %q, erwartet %q", err.Error(), f.errMsg)
			}
			if !f.wantErr && f.wantEingabe != nil {
				if diff := cmp.Diff(*f.wantEingabe, f.eingabe); diff != "" {
					t.Errorf("f.eingabe weicht ab (-erwartet +ist):\n%s", diff)
				}
			}
		})
	}
}

// zeigerAuf macht aus einer geschriebenen Zahl den Zeiger, den titelEingabe.Bestand
// verlangt — "nicht mitgeschickt" ist dort ein eigener Fall (nil).
func zeigerAuf(n int) *int { return &n }

// zeigerAufPreis: dasselbe für den Listenpreis — nil heißt „nicht erfasst", 0 ist ein Preis.
func zeigerAufPreis(p float64) *float64 { return &p }

// Die drei folgenden Fälle: Beim Ändern bedeutet ein fehlender oder geleerter Wert etwas anderes
// als beim Anlegen.
func TestTitelAendern_LeerHeisstBeimAendernNichtVorgabe(t *testing.T) {
	t.Run("fehlender Bestand lässt die Exemplare unangetastet", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()

		// Kein einziger gleicheBestandAb-Aufruf darf folgen: Das Feld "stock" fehlt im Rumpf.
		erwarteFachBekannt(mock, "Mathe")
		mock.ExpectBegin()
		beliebig := make([]any, 21) // eine Stelle je Spalte, die Kennung und die genannten Felder
		for i := range beliebig {
			beliebig[i] = pgxmock.AnyArg()
		}
		erwarteTitelstand(mock, "9783161484100", "Autor")
		mock.ExpectExec("UPDATE buecher_titel").WithArgs(beliebig...).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		tueren := &Tueren{abfragen: NewTitelAbfragen(mock), metadaten: stummerMetadatenClient()}
		rumpf := `{"isbn":"9783161484100","title":"Titel","author":"Autor","subject":"Mathe"}`
		w := ruf(t, tueren, "0f8fad5b-d9cb-469f-a165-70867728950e", rumpf)

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

		tueren := &Tueren{abfragen: NewTitelAbfragen(mock), metadaten: stummerMetadatenClient()}
		rumpf := `{"isbn":"9783161484100","title":"","author":"Autor","subject":"Mathe"}`
		w := ruf(t, tueren, "0f8fad5b-d9cb-469f-a165-70867728950e", rumpf)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400 — ein geleerter Titel darf nicht still zu "+
				"\"Unbekannter Titel\" werden. Rumpf: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "Unbekannter") {
			t.Error("die Antwort nennt den Platzhalter — er ist noch im Spiel")
		}
	})

	// Ein leerer Autor ist nur ein Fehler, wenn der Titel einen trägt. Ohne die Ausnahme
	// ließe sich ein Titel ohne Autor nicht mehr ändern, seit das Speichern die
	// Katalogdienste nicht mehr nach einem fragt.
	for _, fall := range []struct {
		name     string
		hatAutor bool
		status   int
	}{
		{"leerer Autor wird abgelehnt, wenn der Titel einen trägt", true, http.StatusBadRequest},
		{"ein Titel ohne Autor bleibt ohne ihn speicherbar", false, http.StatusOK},
	} {
		t.Run(fall.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			const id = "0f8fad5b-d9cb-469f-a165-70867728950e"
			erwarteFachBekannt(mock, "Mathe")
			mock.ExpectBegin()
			autor := ""
			if fall.hatAutor {
				autor = "Funke, Cornelia"
			}
			erwarteTitelstand(mock, "9783161484100", autor)
			if fall.hatAutor {
				mock.ExpectRollback()
			} else {
				beliebig := make([]any, 21)
				for i := range beliebig {
					beliebig[i] = pgxmock.AnyArg()
				}
				mock.ExpectExec("UPDATE buecher_titel").WithArgs(beliebig...).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
			}

			tueren := &Tueren{abfragen: NewTitelAbfragen(mock), metadaten: stummerMetadatenClient()}
			w := ruf(t, tueren, id, `{"isbn":"9783161484100","title":"Titel","author":"","subject":"Mathe"}`)

			if w.Code != fall.status {
				t.Fatalf("Status %d, erwartet %d — Rumpf: %s", w.Code, fall.status, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "Unbekannter") {
				t.Error("die Antwort nennt den Platzhalter")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Erwartungen: %v", err)
			}
		})
	}
}

// ruf schickt einen PUT an den Aktualisieren-Handler. Den Platzhalter {id} füllt im
// Betrieb der Mux (api_routen.go).
func ruf(t *testing.T, tueren *Tueren, id, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	anfrage := httptest.NewRequest(http.MethodPut, "/api/books/"+id, strings.NewReader(rumpf))
	anfrage.SetPathValue("id", id)
	anfrage.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tueren.TitelAendern(w, anfrage)
	return w
}

// stummerMetadatenClient liefert einen Client, dessen Katalogdienste nichts finden.
func stummerMetadatenClient() *MetadatenClient {
	return &MetadatenClient{httpClient: &http.Client{
		Transport: &mockTransport{roundTripFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
		}},
	}}
}
