package bestellung

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/bestelllink"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
)

// setzeSchulangaben hinterlegt Name und Anschrift der Schule. Die vier Zeilen legt das Schema mit
// leerem Wert an; der Test stellt diesen Stand danach wieder her, statt sie zu löschen.
func setzeSchulangaben(t *testing.T, pool *pgxpool.Pool, werte map[string]string) {
	t.Helper()
	setze := func(werte map[string]string) {
		for schluessel, wert := range werte {
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
				ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
				t.Errorf("Einstellung %s setzen: %v", schluessel, err)
			}
		}
	}
	t.Cleanup(func() {
		setze(map[string]string{"schule_name": "", "schule_strasse": "", "schule_plz": "", "schule_ort": ""})
	})
	setze(werte)
}

// leseAnsicht liest die Antwort der Seite hinter dem Link.
func leseAnsicht(t *testing.T, rec *httptest.ResponseRecorder) oeffentlicheBestellung {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("Ansicht: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var ansicht oeffentlicheBestellung
	if err := json.Unmarshal(rec.Body.Bytes(), &ansicht); err != nil {
		t.Fatalf("Ansicht lesen: %v", err)
	}
	return ansicht
}

// Die Seite hinter dem Link ist das Einzige, was der Lieferant von der Bestellung sieht: wessen
// Bestellung es ist, unter welcher Kundennummer, mit welchen Positionen, ob es Etiketten gibt und
// bis wann der Link gilt. Nach der Bestätigung zeigt sie die Quittung statt des Knopfes; das
// entscheidet sie an bestaetigt_am.
func TestOeffentlicheBestellung_ZeigtDemLieferantenSeineBestellung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	setzeSchulangaben(t, pool, map[string]string{
		"schule_name": "Schule am Park", "schule_strasse": "Parkweg 2", "schule_plz": "61381", "schule_ort": "Friedrichsdorf",
	})
	token := bestellungMitEtiketten(t, srv, pool, 3)

	ansicht := leseAnsicht(t, holeOeffentlicheBestellung(srv, token))
	if ansicht.SchuleName != "Schule am Park" || ansicht.SchuleAnschrift != "Parkweg 2 · 61381 Friedrichsdorf" {
		t.Errorf("Absender %q, %q; erwartet „Schule am Park“, „Parkweg 2 · 61381 Friedrichsdorf“", ansicht.SchuleName, ansicht.SchuleAnschrift)
	}
	if ansicht.Kundennummer != "K-Naacher" || ansicht.AnzahlExemplare != 3 {
		t.Errorf("Kundennummer %q, Exemplare %d; erwartet K-Naacher und 3", ansicht.Kundennummer, ansicht.AnzahlExemplare)
	}
	if diff := cmp.Diff([]oeffentlichePosition{{TitelName: "LMF-Formatprobe", Menge: 3}}, ansicht.Positionen); diff != "" {
		t.Errorf("Positionen weichen ab (-erwartet +bekommen):\n%s", diff)
	}
	if ansicht.Bestelldatum.IsZero() || ansicht.LinkGueltigBis == nil || !ansicht.EtikettenVorhanden {
		t.Errorf("Bestelldatum %v, Link gültig bis %v, Etiketten vorhanden %t; erwartet Datum, Frist und Etiketten",
			ansicht.Bestelldatum, ansicht.LinkGueltigBis, ansicht.EtikettenVorhanden)
	}
	if ansicht.BestaetigtAm != nil {
		t.Errorf("vor der Bestätigung steht bestaetigt_am = %v; die Seite zeigte die Quittung statt des Knopfes", ansicht.BestaetigtAm)
	}

	if rec := bestaetigeUeberLink(srv, token, `{"etiketten_groesse":"klein"}`); rec.Code != http.StatusOK {
		t.Fatalf("Bestätigen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if ansicht := leseAnsicht(t, holeOeffentlicheBestellung(srv, token)); ansicht.BestaetigtAm == nil {
		t.Errorf("nach der Bestätigung fehlt bestaetigt_am; die Seite böte den Knopf noch einmal an")
	}
}

// Leere Felder der Anschrift fallen samt Trenner weg: Eine Zeile wie „ · 61381" stünde sonst
// unter dem Namen der Schule.
func TestOeffentlicheBestellung_AnschriftOhneLeereTeile(t *testing.T) {
	pool := pgtest.Pool(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	for _, f := range []struct {
		name, strasse, plz, ort, erwartet string
	}{
		{"alles", "Parkweg 2", "61381", "Friedrichsdorf", "Parkweg 2 · 61381 Friedrichsdorf"},
		{"ohne Straße", "", "61381", "Friedrichsdorf", "61381 Friedrichsdorf"},
		{"nur der Ort", "", "", "Friedrichsdorf", "Friedrichsdorf"},
		{"nur die Straße", "Parkweg 2", "", "", "Parkweg 2"},
		{"nichts", "", "", "", ""},
	} {
		setzeSchulangaben(t, pool, map[string]string{"schule_strasse": f.strasse, "schule_plz": f.plz, "schule_ort": f.ort})
		if got := srv.schulAnschrift(context.Background()); got != f.erwartet {
			t.Errorf("%s: Anschrift %q, erwartet %q", f.name, got, f.erwartet)
		}
	}
}

// Die Bestätigung über den Link ist der einzige Schreibweg ohne Anmeldung. Eine Größe oder ein
// Raster, die es nicht gibt, weist sie ab, bevor sie etwas schreibt; die Bestellung bleibt offen.
func TestOeffentlichBestaetigen_UnbekannteAngabenSchreibenNichts(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher", true)
	bestellungID, token := bestellungMitToken(t, pool, lieferant, bestelllink.TageVorgabe)

	for rumpf, meldung := range map[string]string{
		`{"etiketten_groesse":"mittel"}`:     "Die Größe der Etiketten muss klein oder groß sein",
		`{"etiketten_format":"gibtesnicht"}`: "Unbekanntes Format der Etiketten",
	} {
		rec := bestaetigeUeberLink(srv, token, rumpf)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400: %s", rumpf, rec.Code, rec.Body.String())
			continue
		}
		if got := tuertest.Fehlermeldung(t, rec); got != meldung {
			t.Errorf("%s: Meldung %q, erwartet %q", rumpf, got, meldung)
		}
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf WHERE id = $1 AND bestaetigt_am IS NULL`, bestellungID); n != 1 {
		t.Errorf("nach zwei abgewiesenen Bestätigungen ist die Bestellung nicht mehr offen")
	}
}

// Der Etikettenbogen hinter dem Link kennt zwei Größen. Eine andere weist er ab, und eine
// Bestellung ohne Vorab-Barcodes hat keinen Bogen: 404 statt eines leeren PDF.
func TestOeffentlicheEtiketten_UnbekannteGroesseUndKeineEtiketten(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	mitEtiketten := bestellungMitEtiketten(t, srv, pool, 2)
	if rec := etikettenBogenHolen(t, srv, mitEtiketten, "riesig", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("Größe „riesig“: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher ohne Etiketten", true)
	_, ohneEtiketten := bestellungMitToken(t, pool, lieferant, bestelllink.TageVorgabe)
	rec := etikettenBogenHolen(t, srv, ohneEtiketten, "klein", "")
	if rec.Code != http.StatusNotFound || tuertest.Fehlermeldung(t, rec) != "Zu dieser Bestellung gibt es keine Etiketten" {
		t.Errorf("Bestellung ohne Vorab-Barcodes: Status %d, erwartet 404 mit Hinweis: %s", rec.Code, rec.Body.String())
	}
}

// Der Bogen öffnet sich im Browser des Lieferanten, damit er ihn direkt druckt: als PDF und
// inline. Als Anhang landete er erst im Download-Ordner.
func TestOeffentlicheEtiketten_BogenOeffnetSichImBrowser(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	rec := etikettenBogenHolen(t, srv, bestellungMitEtiketten(t, srv, pool, 2), "klein", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %.200s", rec.Code, rec.Body.String())
	}
	if typ := rec.Header().Get("Content-Type"); typ != tuer.ContentTypePDF {
		t.Errorf("Content-Type %q, erwartet %q", typ, tuer.ContentTypePDF)
	}
	if art := rec.Header().Get("Content-Disposition"); art != `inline; filename="etiketten_klein.pdf"` {
		t.Errorf("Content-Disposition %q, erwartet inline mit dem Namen des Bogens", art)
	}
}
