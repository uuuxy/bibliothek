package bestellung

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// bedarfSpalten spiegelt die Projektion aus LadeBestellbedarf.
func bedarfSpalten() []string {
	return []string{"id", "titel", "autor", "isbn", "verlag", "signatur",
		"erscheinungsjahr", "cover_url", "gesamt", "kinder", "ist_lernmittel", "auflagen"}
}

func TestBestellbedarfAbfrage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	// Wann ein Buch auf die Liste kommt, prüfen die Tests am Postgres; hier geht es um die
	// Projektion: Bestand 3, Bedarf 5, also fehlen 2.
	mock.ExpectQuery("WITH titel AS").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(bedarfSpalten()).
			AddRow("1", "LMF-Mathe 7", "Verlag", "12345", "Klett", "Ma 7", 2023, "", 3, 5, true, []byte(nil)))

	results, err := repository.LadeBestellbedarf(context.Background(), server.DB.Pool, time.Now())
	if err != nil {
		t.Fatalf("LadeBestellbedarf: unerwarteter Fehler: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, erwartet 1 Treffer", len(results))
	}

	got := results[0]
	if got.Titel != "LMF-Mathe 7" {
		t.Errorf("Titel = %q, erwartet 'LMF-Mathe 7'", got.Titel)
	}
	if got.GesamtBestand != 3 || got.Bedarf != 5 || got.Fehlen != 2 {
		t.Errorf("Bestand %d, Bedarf %d, fehlen %d; erwartet 3, 5 und 2", got.GesamtBestand, got.Bedarf, got.Fehlen)
	}
	// Das Lernmittel-Kennzeichen wird der Topf-Vorschlag im Warenkorb (Migration 109).
	if !got.IstLernmittel {
		t.Error("IstLernmittel: erwartet true — ohne das Kennzeichen landet der Titel im Bücherei-Topf")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// rufeBestellbedarf ruft die Tür gegen den Mock und liefert die Antwort nach Feldern.
func rufeBestellbedarf(t *testing.T, mock pgxmock.PgxPoolIface) map[string]json.RawMessage {
	t.Helper()
	server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	rec := httptest.NewRecorder()
	server.Bestellbedarf().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein Objekt: %v (%s)", err, rec.Body.String())
	}
	return antwort
}

// Eine leere Liste geht als [] hinaus, nicht als null, mit der Grundlage der Rechnung: Daran
// unterscheidet die Seite „es fehlt nichts" von „noch kein Schulbuch ausgegeben".
func TestBestellbedarf_LeereListeMitGrundlage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	// Leere Einstellungen heißen Vorgabe: Warnung an.
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
	mock.ExpectQuery("WITH titel AS").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(bedarfSpalten()))
	mock.ExpectQuery(`WITH\s+kinder AS`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"kinder", "mit_schulbuch"}).AddRow(600, 0))

	antwort := rufeBestellbedarf(t, mock)
	for feld, want := range map[string]string{"zeilen": "[]", "kinder": "600", "kinder_mit_schulbuch": "0", "ausgeschaltet": "false"} {
		if got := string(antwort[feld]); got != want {
			t.Errorf("%s = %s, erwartet %s", feld, got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// Ist die Warnung aus, sagt die Antwort das, statt eine leere Liste wie „es fehlt nichts"
// aussehen zu lassen, und rechnet nichts.
func TestBestellbedarf_AusgeschaltetSagtEsDazu(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	aus := "false"
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("bestellbedarf_warnung_aktiv", &aus))

	antwort := rufeBestellbedarf(t, mock)
	if got := string(antwort["ausgeschaltet"]); got != "true" {
		t.Errorf("ausgeschaltet = %s, erwartet true", got)
	}
	if got := string(antwort["zeilen"]); got != "[]" {
		t.Errorf("zeilen = %s, erwartet []", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}
