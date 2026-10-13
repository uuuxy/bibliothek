package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Vier Türen nennen den Ersatzwert eines Exemplars: der Melde-Dialog, die Exemplarliste der
// Buchakte, die Antwort nach dem Speichern des Zustands und der Vorschlag zum Bescheid. Die
// Einstellung „immer der Kaufpreis" gilt an jeder: Läse eine Tür sie nicht, nennten Buchakte und
// Bescheid für dasselbe Buch zwei Beträge. Verglichen werden die vier Antworten miteinander, je
// ohne und mit der Einstellung.
func TestErsatzwert_BerechnungsgrundlageGiltAnJederTuer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()

	testdaten.AusleihbaresExemplar(t, pool, "Erdkunde 8", "B-GRT-1")
	var exID, titelID string
	if err := pool.QueryRow(ctx,
		`SELECT id, titel_id FROM buecher_exemplare WHERE barcode_id = 'B-GRT-1'`,
	).Scan(&exID, &titelID); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	// Lernmittel im dritten Jahr im Bestand, Kaufpreis 20 €, Listenpreis 41,50 €.
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_titel SET ist_lernmittel = true, listenpreis = 41.50 WHERE id = $1`, titelID); err != nil {
		t.Fatalf("Titel vorbereiten: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare
		SET einkaufspreis = 20.00,
		    erworben_am = now() - interval '2 years', zugang_am = now() - interval '2 years'
		WHERE id = $1`, exID); err != nil {
		t.Fatalf("Exemplar vorbereiten: %v", err)
	}
	// Der Vorschlag zum Bescheid nennt überfällige Bücher eines Schülers.
	schuelerID := testdaten.SeedSchueler(t, pool, "S-GRT-1", "Gesa", "8a")
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() - interval '40 days')`, exID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = 'ersatzwert_immer_kaufpreis'`)
	})

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	bescheidRepo := repository.NewBescheidAbfragen(pool)
	admin := testdaten.AdminFuerAudit(t, pool)
	rufe := func(t *testing.T, h http.Handler, methode, pfad, rumpf, kennung string, ziel any) {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.SetPathValue("id", kennung)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, tuertest.AlsBenutzer(req, admin))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: Status %d: %s", methode, pfad, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), ziel); err != nil {
			t.Fatalf("%s %s: Antwort unlesbar (%s): %v", methode, pfad, rec.Body.String(), err)
		}
	}
	type nennung struct {
		tuer       string
		betrag     float64
		herleitung string
	}
	// vierTueren fragt jede Tür nach dem Ersatzwert desselben Exemplars.
	vierTueren := func(t *testing.T) []nennung {
		t.Helper()
		var dialog struct {
			Betrag     float64 `json:"betrag"`
			Herleitung string  `json:"herleitung"`
		}
		rufe(t, srv.schaden().ErsatzwertVorschlagHandler(bescheidRepo), http.MethodGet,
			"/api/buecher/exemplare/"+exID+"/ersatzwert-vorschlag", "", exID, &dialog)

		var liste []struct {
			Ersatzwert float64 `json:"ersatzwert"`
			Herleitung string  `json:"ersatzwert_herleitung"`
		}
		rufe(t, srv.katalog().GetTitleCopiesHandler(bescheidRepo), http.MethodGet,
			"/api/buecher/titel/"+titelID+"/exemplare", "", titelID, &liste)
		if len(liste) != 1 {
			t.Fatalf("Exemplarliste: %d Exemplare, erwartet 1", len(liste))
		}

		var status struct {
			Ersatzwert *float64 `json:"ersatzwert"`
			Herleitung string   `json:"ersatzwert_herleitung"`
		}
		rufe(t, srv.katalog().UpdateCopyStatusHandler(repository.NewBookRepository(pool), bescheidRepo), http.MethodPut,
			"/api/buecher/exemplare/"+exID+"/status",
			`{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":""}`, exID, &status)
		if status.Ersatzwert == nil {
			t.Fatal("die Antwort nach dem Speichern des Zustands nennt keinen Ersatzwert")
		}

		var vorschlag struct {
			Ausleihen []struct {
				Betrag     float64 `json:"betrag"`
				Herleitung string  `json:"herleitung"`
			} `json:"ausleihen"`
		}
		rufe(t, srv.schaden().BescheidVorschlagHandler(bescheidRepo), http.MethodGet,
			"/api/schueler/"+schuelerID+"/bescheid-vorschlag", "", schuelerID, &vorschlag)
		if len(vorschlag.Ausleihen) != 1 {
			t.Fatalf("Vorschlag zum Bescheid: %d überfällige Bücher, erwartet 1", len(vorschlag.Ausleihen))
		}
		return []nennung{
			{"Melde-Dialog", dialog.Betrag, dialog.Herleitung},
			{"Exemplarliste", liste[0].Ersatzwert, liste[0].Herleitung},
			{"Antwort nach dem Speichern des Zustands", *status.Ersatzwert, status.Herleitung},
			{"Vorschlag zum Bescheid", vorschlag.Ausleihen[0].Betrag, vorschlag.Ausleihen[0].Herleitung},
		}
	}
	// pruefe verlangt an jeder Tür denselben Betrag und die Grundlage im Satz.
	pruefe := func(t *testing.T, lage, grundlage string) float64 {
		t.Helper()
		nennungen := vierTueren(t)
		for _, n := range nennungen {
			if n.betrag != nennungen[0].betrag {
				t.Errorf("%s: %s nennt %.2f, %s nennt %.2f", lage, n.tuer, n.betrag, nennungen[0].tuer, nennungen[0].betrag)
			}
			if n.betrag <= 0 {
				t.Errorf("%s: %s nennt keinen Betrag (%q)", lage, n.tuer, n.herleitung)
			}
			if !strings.Contains(n.herleitung, grundlage) {
				t.Errorf("%s: %s begründet mit %q, erwartet darin %q", lage, n.tuer, n.herleitung, grundlage)
			}
		}
		return nennungen[0].betrag
	}

	ohne := pruefe(t, "ohne die Einstellung", "(Listenpreis)")

	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('ersatzwert_immer_kaufpreis', 'true')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	mit := pruefe(t, "mit der Einstellung", "Kaufpreis (so eingestellt)")
	if mit >= ohne {
		t.Errorf("mit dem Kaufpreis von 20 € als Grundlage %.2f, mit dem Listenpreis von 41,50 € %.2f: erwartet einen kleineren Betrag", mit, ohne)
	}
}
