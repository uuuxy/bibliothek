package katalog

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"
)

// sortiertWieErwartet gibt die Titel rückwärts hinein und vergibt die laufende Nummer aus dem
// Anlegen (sort_order) dabei aufsteigend: Die alte Reihenfolge wäre die umgekehrte.
func sortiertWieErwartet(t *testing.T, erwartet []string) {
	t.Helper()
	buecher := make([]repository.KatalogTitel, 0, len(erwartet))
	for i := len(erwartet) - 1; i >= 0; i-- {
		buecher = append(buecher, repository.KatalogTitel{Titel: erwartet[i], Reihenfolge: len(buecher) + 1})
	}
	sortiereBuecherNachTitel(buecher)
	ist := make([]string, len(buecher))
	for i, b := range buecher {
		ist[i] = b.Titel
	}
	if diff := cmp.Diff(erwartet, ist); diff != "" {
		t.Errorf("sortiereBuecherNachTitel: Reihenfolge weicht ab (-erwartet +ist):\n%s", diff)
	}
}

// Die Titelliste steht nach dem Titel, wie ein deutsches Register: Umlaute bei ihrem
// Grundbuchstaben, Zahlen nach ihrer Größe, Groß- und Kleinschreibung trennt keine Buchstaben.
// Dieselbe Reihenfolge gibt der Browser mit Intl.Collator('de', {numeric: true}); beide Seiten
// lesen dieselben Fälle (Bauart wie auflagenText.faelle.json). Ordnen sie verschieden, steht
// ein Titel am Server woanders als in einer Liste des Browsers.
func TestSortiereBuecherNachTitel_WieImBrowser(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/utils/titelReihenfolge.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Fall     string   `json:"fall"`
			Erwartet []string `json:"erwartet"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(pruefung.Faelle) < 10 {
		t.Fatalf("len(pruefung.Faelle) = %v, erwartet mindestens %v: liest der Test noch titelReihenfolge.faelle.json?", len(pruefung.Faelle), 10)
	}

	for _, f := range pruefung.Faelle {
		t.Run(f.Fall, func(t *testing.T) {
			if len(f.Erwartet) < 3 {
				t.Fatalf("len(f.Erwartet) = %v, erwartet mindestens %v", len(f.Erwartet), 3)
			}
			sortiertWieErwartet(t, f.Erwartet)
		})
	}

	// Liest die JavaScript-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../../frontend/src/lib/utils/titelReihenfolge.test.js")
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !strings.Contains(string(vitest), "./titelReihenfolge.faelle.json") {
		t.Errorf("string(vitest) = %q, erwartet darin %q", string(vitest), "./titelReihenfolge.faelle.json")
	}
	if !strings.Contains(string(vitest), "new Intl.Collator('de', { numeric: true })") {
		t.Errorf("string(vitest) = %q, erwartet darin %q", string(vitest), "new Intl.Collator('de', { numeric: true })")
	}
}

// Leerzeichen am Rand eines Titels zählen nicht.
func TestSortiereBuecherNachTitel_OhneLeerzeichenAmRand(t *testing.T) {
	sortiertWieErwartet(t, []string{"Apfel", " Banane", "Clown "})
}

// Gleiche Titel behalten die Reihenfolge, in der die Abfrage sie liefert.
func TestSortiereBuecherNachTitel_GleicheTitelBleibenInAbfrageReihenfolge(t *testing.T) {
	buecher := []repository.KatalogTitel{
		{ID: "c", Titel: "Mathe 7"}, {ID: "x", Titel: "Zebra"}, {ID: "a", Titel: "Mathe 7"}, {ID: "b", Titel: "Mathe 7"},
	}
	sortiereBuecherNachTitel(buecher)
	ist := make([]string, len(buecher))
	for i, b := range buecher {
		ist[i] = b.ID
	}
	if diff := cmp.Diff([]string{"c", "a", "b", "x"}, ist); diff != "" {
		t.Errorf("sortiereBuecherNachTitel: Reihenfolge weicht ab (-erwartet +ist):\n%s", diff)
	}
}

func TestTitelListe(t *testing.T) {
	faelle := []struct {
		name           string
		adresse        string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "Success - empty params",
			adresse: "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				dateStr := "2023-01-01"
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel", "verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage", "listenpreis", "mehrjahresband", "werk_id", "werk_rang",
					}).AddRow(
						"b1", "123", "Book 1", "Author 1", "Sig 1", "url", "Math", "G", false, int64(5), int64(10), int64(2), &dateStr, 0, "Buch", 5, 10, "Sub", "Ver", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "", 0,
					))
				// Die Suchwörter der gelieferten Titel (repository.ladeSuchwoerterDerTitel).
				m.ExpectQuery("(?s)SELECT tsw.titel_id.*FROM schlagworte sw").
					WithArgs([]string{"b1"}).
					WillReturnRows(pgxmock.NewRows([]string{"titel_id", "woerter"}).
						AddRow("b1", []string{"Fantasy", "Tierfantasy"}))
				// Die Standorte der Exemplare im Bestand (repository.ladeStandorteDerTitel).
				m.ExpectQuery("(?s)SELECT e.titel_id::text, e.standort.*FROM buecher_exemplare e").
					WithArgs([]string{"b1"}).
					WillReturnRows(pgxmock.NewRows([]string{"titel_id", "standort", "anzahl"}).
						AddRow("b1", "Lehrerschrank", 2))
			},
			expectedStatus: http.StatusOK,
			// schlagworte: null heißt „nicht geladen" — die Liste bleibt schlank;
			// nur der Einzel-Read lädt sie, und ein PUT mit null lässt sie unangetastet.
			// suchwoerter trägt die Wörter, über die die Suche im Browser den Titel findet,
			// standorte die Standorte der Exemplare für die Spalte der Titel-Verwaltung.
			expectedBody: `{"data":[{"id":"b1","isbn":"123","title":"Book 1","author":"Author 1","signatur":"Sig 1","coverUrl":"url","subject":"Math","track":"G","istLernmittel":false,"stock":10,"verfuegbar":5,"gesamt":10,"imZulauf":2,"lastCounted":"2023-01-01","sortOrder":0,"medientyp":"Buch","jahrgangVon":5,"jahrgangBis":10,"mehrjahresband":false,"untertitel":"Sub","auflage":"4. Aufl. 2023","listenpreis":null,"verlag":"Ver","erscheinungsjahr":2020,"erweiterteEigenschaften":{},"schlagworte":null,"suchwoerter":["Fantasy","Tierfantasy"],"standorte":[{"standort":"Lehrerschrank","anzahl":2}]}]}`,
		},
		{
			name:    "Success - synonym translation",
			adresse: "/api/books?q=powi",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "politik", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel", "verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage", "listenpreis", "mehrjahresband", "werk_id", "werk_rang",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			// Die Aufräumsicht der Verwaltung: dasselbe Prädikat mit NOT.
			name:    "Success - Aufräumsicht bestand=ohne",
			adresse: "/api/books?bestand=ohne",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*WHERE NOT EXISTS.*").
					WithArgs("", "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel", "verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage", "listenpreis", "mehrjahresband", "werk_id", "werk_rang",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name:           "Error - unbekannte Sicht",
			adresse:        "/api/books?bestand=alle",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Unbekannter Wert für den Bestand"}`,
		},
		{
			name:           "Error - query string too long",
			adresse:        "/api/books?q=" + strings.Repeat("a", 201),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Der Suchbegriff ist zu lang, höchstens 200 Zeichen"}`,
		},
		{
			// Die Liste kennt keinen Filter nach Klasse mehr: Der Parameter wird nicht gelesen,
			// auch kein unlesbarer Wert.
			name:    "Success - gradeLevel wird nicht mehr gelesen",
			adresse: "/api/books?gradeLevel=abc&grade=7",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel", "verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage", "listenpreis", "mehrjahresband", "werk_id", "werk_rang",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name:    "Error - DB fails",
			adresse: "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "", 50000).
					WillReturnError(errors.New("db fail"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ein interner Serverfehler ist aufgetreten"}`,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool: %v", err)
			}
			defer mock.Close()

			tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

			if f.setupMock != nil {
				f.setupMock(mock)
			}

			anfrage := httptest.NewRequest(http.MethodGet, f.adresse, nil)
			w := httptest.NewRecorder()

			tueren.TitelListe()(w, anfrage)

			ergebnis := w.Result()
			defer func() {
				if err := ergebnis.Body.Close(); err != nil {
					t.Fatalf("ergebnis.Body.Close: %v", err)
				}
			}()
			if ergebnis.StatusCode != f.expectedStatus {
				t.Errorf("Status = %v, erwartet %v", ergebnis.StatusCode, f.expectedStatus)
			}

			if f.expectedBody != "" {
				rumpf, err := io.ReadAll(ergebnis.Body)
				if err != nil {
					t.Fatalf("io.ReadAll: %v", err)
				}
				if diff := cmp.Diff(jsonWert(t, f.expectedBody), jsonWert(t, string(rumpf))); diff != "" {
					t.Errorf("Antwort weicht ab (-erwartet +ist):\n%s", diff)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("offene Erwartungen: %v", err)
			}
		})
	}
}

// jsonWert liest einen JSON-Text in Go-Werte, damit der Vergleich weder auf die Folge der
// Schlüssel noch auf Leerraum achtet.
func jsonWert(t *testing.T, text string) any {
	t.Helper()
	var wert any
	if err := json.Unmarshal([]byte(text), &wert); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", text, err)
	}
	return wert
}
