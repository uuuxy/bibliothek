package katalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Wer markierte Titel löscht, steht im Protokoll wie beim Löschen eines einzelnen Titels: ein
// Eintrag je Titel mit dem Bearbeiter, dazu je Exemplar die Spur mit Barcode. Vorher nannte das
// Sammel-Löschen niemanden, und ein Titel ohne Exemplar verschwand ohne einen einzigen Eintrag
// (docs/invarianten.md, Frage 19).
func TestMarkierteTitelLoeschen_NenntWerGeloeschtHat(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bearbeiter := bearbeiterFuerAussonderung(t, pool)

	mitExemplar := testdaten.LegeTitelAn(t, pool, "Sammel mit Exemplar", 0)
	exemplar := testdaten.LegeExemplarAn(t, pool, mitExemplar, "SAMMEL-SPUR-1", true, "")
	ohneExemplar := testdaten.LegeTitelAn(t, pool, "Sammel ohne Exemplar", 0)

	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	rumpf := fmt.Sprintf(`{"ids":[%q,%q]}`, mitExemplar, ohneExemplar)
	anfrage := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader(rumpf))
	anfrage = anfrage.WithContext(context.WithValue(anfrage.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: bearbeiter}))
	rec := httptest.NewRecorder()
	tueren.MarkierteTitelLoeschen()(rec, anfrage)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/books: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}

	for _, fall := range []struct{ name, tabelle, kennung string }{
		{"Titel mit Exemplar", "buecher_titel", mitExemplar},
		{"Titel ohne Exemplar", "buecher_titel", ohneExemplar},
		{"Exemplar", "buecher_exemplare", exemplar},
	} {
		var akteur, wer string
		err := pool.QueryRow(ctx, `
			SELECT akteur, coalesce(bearbeiter_id::text, '') FROM audit_log
			WHERE tabelle = $1 AND aktion = 'DELETE' AND datensatz_id = $2`,
			fall.tabelle, fall.kennung).Scan(&akteur, &wer)
		if err != nil {
			t.Errorf("%s: kein Eintrag im Protokoll (%v)", fall.name, err)
			continue
		}
		if akteur != "USER" || wer != bearbeiter {
			t.Errorf("%s: akteur %q, Bearbeiter %q, erwartet USER und %q", fall.name, akteur, wer, bearbeiter)
		}
	}
}
