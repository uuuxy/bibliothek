package inventurlauf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/code39"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Ein Lernmittel mit kurzer Nummer kann noch sein altes LMF-Etikett tragen. Der Scan davon
// erfasst es in der Inventur, und die Antwort nennt die Nummer, die es jetzt trägt.
func TestScanVerbuchen_AltesLMFEtikett(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	signatur := "INV-LMF-" + suffix
	nummer := "B-LMFI-" + suffix
	exemplarID := testdaten.LegeExemplarMitSignaturAn(t, pool, signatur, nummer)
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET lmf_barcode = $1 WHERE id = $2`,
		"LMF-9783060158423-"+suffix, exemplarID); err != nil {
		t.Fatalf("altes Etikett setzen: %v", err)
	}

	lauf, err := repository.NewInventurAbfragen(pool).LegeLaufAn(ctx, "signature",
		repository.InventurBereich{Signatur: &signatur}, "LMF", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	rec := inventurScan(t, srv, lauf.ID, "LMF-9783060158423-"+suffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("Scan des alten Etiketts = %d: %s", rec.Code, rec.Body.String())
	}
	if n := erfasstFuer(t, pool, lauf.ID, exemplarID); n != 1 {
		t.Errorf("%d Erfassungen des Exemplars, erwartet eine", n)
	}
	var antwort scanAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if antwort.BarcodeID != nummer {
		t.Errorf("die Antwort nennt %q, erwartet die Nummer %q", antwort.BarcodeID, nummer)
	}
}

// Die Inventur liest dieselben alten Aufdrucke wie die Theke: das Etikett von Littera (im
// Strichcode eine EAN-13, im Bestand die kurze Mediennummer) und ein Code-39-Etikett, dessen
// Prüfzeichen der Scanner mitliefert. Fände sie die Bücher nicht, buchte der Abschluss sie als
// Verlust.
func TestScanVerbuchen_AlteAufdruckeWieAnDerTheke(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	signatur := "INV-ALT-" + suffix
	code39Nummer := "B-" + suffix
	pruefzeichen, ok := code39.Pruefzeichen(code39Nummer)
	if !ok {
		t.Fatalf("kein Prüfzeichen zu %q", code39Nummer)
	}

	lauf, err := repository.NewInventurAbfragen(pool).LegeLaufAn(ctx, "signature",
		repository.InventurBereich{Signatur: &signatur}, "Alte Aufdrucke", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	for _, fall := range []struct {
		name, nummer, scan string
	}{
		// Gemessen an einem Buch der Schule: Aufdruck 58968, Scan 5896800039556.
		{"Etikett von Littera", "58968", "5896800039556"},
		{"Code 39 mit Prüfzeichen", code39Nummer, code39Nummer + string(pruefzeichen)},
	} {
		t.Run(fall.name, func(t *testing.T) {
			exemplarID := testdaten.LegeExemplarMitSignaturAn(t, pool, signatur, fall.nummer)
			rec := inventurScan(t, srv, lauf.ID, fall.scan)
			if rec.Code != http.StatusOK {
				t.Fatalf("Scan %q = %d: %s", fall.scan, rec.Code, rec.Body.String())
			}
			if n := erfasstFuer(t, pool, lauf.ID, exemplarID); n != 1 {
				t.Errorf("%d Erfassungen des Exemplars, erwartet eine", n)
			}
			var antwort scanAntwort
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort: %v", err)
			}
			if antwort.BarcodeID != fall.nummer {
				t.Errorf("die Antwort nennt %q, erwartet die Nummer %q", antwort.BarcodeID, fall.nummer)
			}
		})
	}
}
