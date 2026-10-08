package inventur

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sortiertWieErwartet gibt die Titel rückwärts hinein und vergibt die laufende Nummer aus dem
// Anlegen (sort_order) dabei aufsteigend: Die alte Reihenfolge wäre die umgekehrte.
func sortiertWieErwartet(t *testing.T, erwartet []string) {
	t.Helper()
	buecher := make([]Book, 0, len(erwartet))
	for i := len(erwartet) - 1; i >= 0; i-- {
		buecher = append(buecher, Book{Title: erwartet[i], SortOrder: len(buecher) + 1})
	}
	sortiereBuecherNachTitel(buecher)
	ist := make([]string, len(buecher))
	for i, b := range buecher {
		ist[i] = b.Title
	}
	assert.Equal(t, erwartet, ist)
}

// Die Titelliste steht nach dem Titel, wie ein deutsches Register: Umlaute bei ihrem
// Grundbuchstaben, Zahlen nach ihrer Größe, Groß- und Kleinschreibung trennt keine Buchstaben.
// Dieselbe Reihenfolge gibt der Browser mit Intl.Collator('de', {numeric: true}); beide Seiten
// lesen dieselben Fälle (Bauart wie auflagenText.faelle.json). Ordnen sie verschieden, steht
// ein Titel am Server woanders als in einer Liste des Browsers.
func TestSortiereBuecherNachTitel_WieImBrowser(t *testing.T) {
	const faelleDatei = "../frontend/src/lib/utils/titelReihenfolge.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	require.NoError(t, err)
	var pruefung struct {
		Faelle []struct {
			Fall     string   `json:"fall"`
			Erwartet []string `json:"erwartet"`
		} `json:"faelle"`
	}
	require.NoError(t, json.Unmarshal(roh, &pruefung))
	require.GreaterOrEqual(t, len(pruefung.Faelle), 10, "liest der Test noch titelReihenfolge.faelle.json?")

	for _, f := range pruefung.Faelle {
		t.Run(f.Fall, func(t *testing.T) {
			require.GreaterOrEqual(t, len(f.Erwartet), 3)
			sortiertWieErwartet(t, f.Erwartet)
		})
	}

	// Liest die JavaScript-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../frontend/src/lib/utils/titelReihenfolge.test.js")
	require.NoError(t, err)
	assert.Contains(t, string(vitest), "./titelReihenfolge.faelle.json")
	assert.Contains(t, string(vitest), "new Intl.Collator('de', { numeric: true })")
}

// Leerzeichen am Rand eines Titels zählen nicht.
func TestSortiereBuecherNachTitel_OhneLeerzeichenAmRand(t *testing.T) {
	sortiertWieErwartet(t, []string{"Apfel", " Banane", "Clown "})
}

// Gleiche Titel behalten die Reihenfolge, in der die Abfrage sie liefert.
func TestSortiereBuecherNachTitel_GleicheTitelBleibenInAbfrageReihenfolge(t *testing.T) {
	buecher := []Book{
		{ID: "c", Title: "Mathe 7"}, {ID: "x", Title: "Zebra"}, {ID: "a", Title: "Mathe 7"}, {ID: "b", Title: "Mathe 7"},
	}
	sortiereBuecherNachTitel(buecher)
	ist := make([]string, len(buecher))
	for i, b := range buecher {
		ist[i] = b.ID
	}
	assert.Equal(t, []string{"c", "a", "b", "x"}, ist)
}

func TestBearbeiteBuecherListe(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - empty params",
			url:  "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				dateStr := "2023-01-01"
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel", "verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage", "listenpreis", "mehrjahresband", "werk_id", "werk_rang",
					}).AddRow(
						"b1", "123", "Book 1", "Author 1", "Sig 1", "url", "Math", "G", false, int64(5), int64(10), int64(2), &dateStr, 0, "Buch", 5, 10, "Sub", "Ver", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "", 0,
					))
				// Die Suchwörter der gelieferten Titel (repository.SuchwoerterDerTitel).
				m.ExpectQuery("(?s)SELECT tsw.titel_id.*FROM schlagworte sw").
					WithArgs([]string{"b1"}).
					WillReturnRows(pgxmock.NewRows([]string{"titel_id", "woerter"}).
						AddRow("b1", []string{"Fantasy", "Tierfantasy"}))
				// Die Standorte der Exemplare im Bestand (repository.StandorteDerTitel).
				m.ExpectQuery("(?s)SELECT e.titel_id::text, e.standort.*FROM buecher_exemplare e").
					WithArgs([]string{"b1"}).
					WillReturnRows(pgxmock.NewRows([]string{"titel_id", "standort", "anzahl"}).
						AddRow("b1", "Lehrerschrank", 2))
			},
			expectedStatus: http.StatusOK,
			// schlagworte: null heißt „nicht geladen" — die Liste bleibt schlank (Migration 138);
			// nur der Einzel-Read lädt sie, und ein PUT mit null lässt sie unangetastet.
			// suchwoerter trägt die Wörter, über die die Suche im Browser den Titel findet,
			// standorte die Standorte der Exemplare für die Spalte der Titel-Verwaltung.
			expectedBody: `{"data":[{"id":"b1","isbn":"123","title":"Book 1","author":"Author 1","signatur":"Sig 1","coverUrl":"url","subject":"Math","track":"G","istLernmittel":false,"stock":10,"verfuegbar":5,"gesamt":10,"imZulauf":2,"lastCounted":"2023-01-01","sortOrder":0,"medientyp":"Buch","jahrgangVon":5,"jahrgangBis":10,"mehrjahresband":false,"untertitel":"Sub","auflage":"4. Aufl. 2023","listenpreis":null,"verlag":"Ver","erscheinungsjahr":2020,"erweiterteEigenschaften":{},"schlagworte":null,"suchwoerter":["Fantasy","Tierfantasy"],"standorte":[{"standort":"Lehrerschrank","anzahl":2}]}]}`,
		},
		{
			name: "Success - synonym translation",
			url:  "/api/books?q=powi",
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
			// Die Aufräumsicht der Verwaltung: dasselbe Prädikat mit NOT (OFFEN.md 9.4).
			name: "Success - Aufräumsicht bestand=ohne",
			url:  "/api/books?bestand=ohne",
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
			url:            "/api/books?bestand=alle",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"ungültiger query-parameter bestand (erlaubt: ohne)"}`,
		},
		{
			name:           "Error - query string too long",
			url:            "/api/books?q=" + strings.Repeat("a", 201),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"suchbegriff zu lang (max. 200 zeichen)"}`,
		},
		{
			// Die Liste kennt keinen Filter nach Klasse mehr: Der Parameter wird nicht gelesen,
			// auch kein unlesbarer Wert.
			name: "Success - gradeLevel wird nicht mehr gelesen",
			url:  "/api/books?gradeLevel=abc&grade=7",
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
			name: "Error - DB fails",
			url:  "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", "", 50000).
					WillReturnError(errors.New("db fail"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ein interner Datenbankfehler ist aufgetreten."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.BearbeiteBuecherListe(w, req)

			res := w.Result()
			defer func() { require.NoError(t, res.Body.Close()) }()
			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			if tt.expectedBody != "" {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
