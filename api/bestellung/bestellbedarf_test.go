package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// bedarfSpalten spiegelt die Projektion aus LadeBestellbedarf.
func bedarfSpalten() []string {
	return []string{"id", "titel", "autor", "isbn", "verlag", "signatur",
		"erscheinungsjahr", "cover_url", "meldebestand", "verfuegbar", "gesamt", "ist_lernmittel", "auflagen"}
}

func TestBestellbedarfAbfrage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	server := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: mock}}}

	// Echter Fehlbestand: gesamt 3 < Meldebestand 5 (ein Titel hat Exemplare verloren).
	// Ein verliehener Klassensatz (gesamt 30 >= 5) taucht dagegen gar nicht auf — das
	// prüft der PG-Test; hier geht es um die Projektion beider Bestandszahlen.
	mock.ExpectQuery("WITH titel AS").
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows(bedarfSpalten()).
			AddRow("1", "LMF-Mathe 7", "Verlag", "12345", "Klett", "Ma 7", 2023, "", 5, 1, 3, true, []byte(nil)))

	results, err := repository.LadeBestellbedarf(context.Background(), server.DB.Pool, "", 5)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: unerwarteter Fehler: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("erwartet 1 Treffer, waren %d", len(results))
	}

	got := results[0]
	if got.Titel != "LMF-Mathe 7" {
		t.Errorf("Titel: erwartet 'LMF-Mathe 7', war %q", got.Titel)
	}
	if got.VerfuegbarBestand != 1 {
		t.Errorf("VerfuegbarBestand: erwartet 1, war %d", got.VerfuegbarBestand)
	}
	// Beide Bestandszahlen müssen ankommen — der Gesamtbestand ist die Nachbestell-Schwelle.
	if got.GesamtBestand != 3 {
		t.Errorf("GesamtBestand: erwartet 3, war %d", got.GesamtBestand)
	}
	// Das Lernmittel-Kennzeichen wird der Topf-Vorschlag im Warenkorb (Migration 109).
	if !got.IstLernmittel {
		t.Error("IstLernmittel: erwartet true — ohne das Kennzeichen landet der Titel im Bücherei-Topf")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// TestBestellbedarfFilterVorgabeIstLMF sichert die fachliche Vorauswahl ab: Nachbestellt
// werden Lernmittel; der Freihandbestand besteht überwiegend aus bewussten
// Einzelstücken (Prüf-/Leseexemplare). Ohne diese Vorgabe enthielte die Liste
// praktisch den gesamten Katalog und wäre unbenutzbar.
func TestBestellbedarfFilterVorgabeIstLMF(t *testing.T) {
	faelle := []struct {
		name, query, wantFragment string
	}{
		// Das Prädikat ist die Spalte buecher_titel.ist_lernmittel (Migration 093).
		{"ohne Parameter", "", "AND t.ist_lernmittel"},
		{"type=lmf", "?type=lmf", "AND t.ist_lernmittel"},
		{"type=freihand", "?type=freihand", "AND NOT t.ist_lernmittel"},
		{"type=alle", "?type=alle", ""},
		{"unbekannter Wert", "?type=kaputt", ""},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/bestellungen"+f.query, nil)
			if got := bestellbedarfFilter(r); got != f.wantFragment {
				t.Errorf("erwartet %q, war %q", f.wantFragment, got)
			}
		})
	}
}

// TestBestellbedarfLeereListeIstArray: wie bei der Schülerliste darf eine leere Liste
// nicht als null herausgehen — das Frontend ruft .length darauf auf.
func TestBestellbedarfLeereListeIstArray(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	// Die Tür liest zuerst die Einstellungen (Warnung aktiv? Schwelle?), dann den
	// Bestellbedarf — beide Queries müssen der Reihe nach erwartet werden. Leere
	// Zeilen der Einstellungen ⇒ Vorgaben (Warnung an, Schwelle 3).
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
	mock.ExpectQuery("WITH titel AS").
		WithArgs(3). // Vorgabe der Schwelle (leere Einstellungen ⇒ 3)
		WillReturnRows(pgxmock.NewRows(bedarfSpalten()))

	server := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: mock}}}
	rec := httptest.NewRecorder()
	server.Bestellbedarf().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen", nil))

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("erwartet [], war: %s", body)
	}
}

// Die Nachbestellmenge steht auf einem Dokument, das an einen Händler geht, und wird von
// niemandem gegengelesen. Rechnete sie mit dem Titel-Feld meldebestand, das niemand mehr pflegt
// (Vorgabe 5), während die Auswahl der Titel der Schwelle folgt, stünde bei Schwelle 30 und 10
// vorhandenen Exemplaren dort "-5".
func TestNachbestellmenge(t *testing.T) {
	faelle := map[string]struct {
		schwelle, gesamt, want int
	}{
		"halber Klassensatz fehlt":      {30, 15, 15},
		"nichts vorhanden":              {30, 0, 30},
		"Soll genau erreicht":           {30, 30, 0},
		"mehr als das Soll vorhanden":   {5, 12, 0},
		"Schwelle 0 (Warnung aus/leer)": {0, 3, 0},
	}

	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			if got := nachbestellmenge(f.schwelle, f.gesamt); got != f.want {
				t.Errorf("nachbestellmenge(%d, %d) = %d, want %d", f.schwelle, f.gesamt, got, f.want)
			}
			if got := nachbestellmenge(f.schwelle, f.gesamt); got < 0 {
				t.Errorf("negative Bestellmenge auf der Händler-Liste: %d", got)
			}
		})
	}
}
