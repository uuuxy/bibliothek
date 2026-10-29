package bestellung

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ladeBestellKonfiguration ruft die Tür, wie der Warenkorb es beim Öffnen tut.
func ladeBestellKonfiguration(t *testing.T, srv *Tueren, pool *pgxpool.Pool) bestellKonfiguration {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.BestellKonfiguration(repository.NewEinstellungsAbfragen(pool), repository.NewLieferantenAbfragen(pool)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen/konfiguration", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort bestellKonfiguration
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	return antwort
}

// Der Warenkorb warnt vor dem Bestellen, wenn der Hauptlieferant keinen Bestätigungs-Link
// bekäme: Es gibt einen Hauptlieferanten, aber keine öffentliche Adresse. Ein Feld aus lauter
// Leerzeichen ist keine Adresse, daraus entstünde kein Link.
func TestBestellKonfiguration_WarntWennDerLinkNichtEntstehenKann(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	// Ein Hauptlieferant aus einem früheren Test dieses Laufs zählte sonst mit.
	if _, err := pool.Exec(t.Context(), `UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant`); err != nil {
		t.Fatalf("bisherigen Hauptlieferanten räumen: %v", err)
	}
	t.Cleanup(func() { testdaten.SetzeOeffentlicheAdresse(t, pool, "") })
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ohneAdresse := func() bool {
		t.Helper()
		return ladeBestellKonfiguration(t, srv, pool).BestelllinkOhneAdresse
	}

	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	if ohneAdresse() {
		t.Error("ohne Hauptlieferant warnt die Konfiguration, obwohl kein Link verschickt würde")
	}
	testdaten.LegeLieferantAn(t, pool, "Naacher", true)
	if !ohneAdresse() {
		t.Error("Hauptlieferant ohne öffentliche Adresse: keine Warnung")
	}
	testdaten.SetzeOeffentlicheAdresse(t, pool, "   ")
	if !ohneAdresse() {
		t.Error("eine Adresse aus lauter Leerzeichen gilt als hinterlegt: keine Warnung")
	}
	testdaten.SetzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	if ohneAdresse() {
		t.Error("mit öffentlicher Adresse warnt die Konfiguration weiter")
	}
}

// Der Warenkorb zeigt das Preisfeld nur, wenn die Schule Preise erfasst. Ohne Eintrag gilt die
// Vorgabe „an“; erst „false“ schaltet das Feld ab.
func TestBestellKonfiguration_FolgtDerEinstellungDerPreise(t *testing.T) {
	pool := pgtest.Pool(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ohneEintrag := `DELETE FROM system_einstellungen WHERE schluessel = 'preise_erfassen'`
	pgtest.RaeumeAuf(t, pool, ohneEintrag)
	t.Cleanup(func() { pgtest.RaeumeAuf(t, pool, ohneEintrag) })
	setze := func(wert string) {
		t.Helper()
		if _, err := pool.Exec(t.Context(), `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ('preise_erfassen', $1)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, wert); err != nil {
			t.Fatalf("preise_erfassen setzen: %v", err)
		}
	}

	if !ladeBestellKonfiguration(t, srv, pool).PreiseErfassen {
		t.Error("ohne Eintrag zeigt der Warenkorb kein Preisfeld; die Vorgabe ist „an“")
	}
	setze("false")
	if ladeBestellKonfiguration(t, srv, pool).PreiseErfassen {
		t.Error("Preise sind abgeschaltet, der Warenkorb zeigt das Preisfeld trotzdem")
	}
	setze("true")
	if !ladeBestellKonfiguration(t, srv, pool).PreiseErfassen {
		t.Error("Preise sind eingeschaltet, der Warenkorb zeigt kein Preisfeld")
	}
}
