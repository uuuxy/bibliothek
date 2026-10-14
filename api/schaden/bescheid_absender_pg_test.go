package schaden

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Der Nachdruck eines Bescheids ist derselbe Brief, auch bei den Angaben der Schule:
// Schulanschrift, Geschäftszeichen, Bearbeiter, Zahlstelle, Bankverbindung, Aufsicht und
// Schulleitung stehen im Schnappschuss (Migration 141). Läse der Nachdruck sie aus den
// Einstellungen, trüge ein alter Bescheid nach einem Wechsel der Schulleitung die neue
// Unterschrift, nach einem Kontowechsel das neue Konto. Den Empfänger hält der Bescheid ebenso
// fest (TestBescheidNachdruck_BleibtDerselbeBrief).
func TestBescheidNachdruck_AbsenderBleibtWieZumBriefdatum(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.LeereBescheide(t, pool)
	ctx := context.Background()
	testdaten.SetzeBescheidAngaben(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	repo := repository.NewBescheidAbfragen(pool)

	sid := testdaten.LegeSchuelerAn(t, pool, "S-ABSENDER", "Absenderkind", "08G2")
	titelID := testdaten.LegeLernmittelAn(t, pool, "Biologie 8")
	f := testdaten.LegeForderungAn(t, pool, sid, testdaten.LegeExemplarAn(t, pool, titelID, "ABSENDER-1", true, ""),
		"nicht_zurueckgegeben", "Biologie 8 nicht zurück")
	rec := bescheidErstellenUeberTuer(t, srv, pool, sid,
		testdaten.BescheidRumpf(testdaten.In28Tagen(), map[string]float64{f: 21.50}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid: %d %s", rec.Code, rec.Body.String())
	}
	var b repository.Bescheid
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}

	// Die Schule ändert ihre Angaben nach dem Brief.
	for k, v := range map[string]string{
		"bescheid_schulleitung":   "Neue Leitung",
		"schule_name":             "Umbenannte Schule",
		"bescheid_aufsicht":       "Anderes Schulamt, Nebenweg 2, 54321 Anderswo",
		"bescheid_bankverbindung": "IBAN DE00 1111 2222 3333 4444 55",
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, k, v); err != nil {
			t.Fatalf("Einstellung %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel = 'bescheid_bankverbindung'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/bescheide/"+b.ID+"/pdf", nil)
	req.SetPathValue("id", b.ID)
	pdfRec := httptest.NewRecorder()
	srv.BescheidPDF(repo)(pdfRec, req)
	if pdfRec.Code != http.StatusOK {
		t.Fatalf("PDF: %d %s", pdfRec.Code, pdfRec.Body.String())
	}
	text := pdftest.Suchtext(t, pdfRec.Body.Bytes())

	for _, alt := range []string{"Dr. Beispiel", "Testschule", "Staatliches Schulamt", "DE86500500000001002401"} {
		if !strings.Contains(text, alt) {
			t.Errorf("der Nachdruck nennt %q nicht mehr — so stand es im Brief", alt)
		}
	}
	for _, neu := range []string{"Neue Leitung", "Umbenannte Schule", "Anderes Schulamt", "DE00 1111"} {
		if strings.Contains(text, neu) {
			t.Errorf("der Nachdruck trägt die NEUE Angabe %q — dann ist es nicht mehr derselbe Bescheid", neu)
		}
	}
}
