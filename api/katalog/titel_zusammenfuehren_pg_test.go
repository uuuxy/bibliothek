package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Tür zum Zusammenführen zweier Titel. Die Regeln selbst prüft
// repository/titel_zusammenfuehren_pg_test.go; hier geht es um das, was nur an der Tür entsteht:
// Vorschau ohne Wirkung, Zusammenführen mit Eintrag im Protokoll der Verwaltung, Kennungen,
// Anmeldung und die Übersetzung der fachlichen Fehler.
func TestTitelZusammenfuehren_Tuer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	protokoll := repository.NewProtokollAbfragen(pool)
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	titel := func(name, isbn string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, NULLIF($2, '')) RETURNING id::text`,
			name, isbn).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	gross := titel("Natur und Technik - Biologie 7/10", "")
	klein := titel("Natur und Technik - Biologie 7/10", "9783060158423")
	fremd := titel("Lambacher Schweizer 7", "9783127331592")
	testdaten.LegeExemplarAn(t, pool, gross, "101602", true, "")
	testdaten.LegeExemplarAn(t, pool, klein, "B-30001", true, "")

	aufruf := func(methode, zielID, quelleID string, angemeldet bool) (int, string) {
		t.Helper()
		pfad := "/api/buecher/titel/" + zielID + "/zusammenfuehren"
		var req *http.Request
		if methode == http.MethodGet {
			req = httptest.NewRequest(methode, pfad+"?quelle_id="+quelleID, nil)
		} else {
			req = httptest.NewRequest(methode, pfad, strings.NewReader(`{"quelle_id": "`+quelleID+`"}`))
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetPathValue("id", zielID)
		if angemeldet {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
				&auth.Claims{KontoID: bearbeiter, Rolle: auth.RolleMitarbeiter}))
		}
		rec := httptest.NewRecorder()
		if methode == http.MethodGet {
			srv.TitelZusammenfuehrungVorschau()(rec, req)
		} else {
			srv.FuehreTitelZusammen(protokoll)(rec, req)
		}
		return rec.Code, rec.Body.String()
	}
	exemplareAm := func(id string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	for _, f := range []struct {
		name, methode, ziel, quelle string
		angemeldet                  bool
		status                      int
		text                        string
	}{
		{"ohne Anmeldung", http.MethodPost, gross, klein, false, http.StatusUnauthorized, ""},
		{"ohne Quelle", http.MethodPost, gross, "", true, http.StatusBadRequest, "quelle_id"},
		{"Quelle ist keine Kennung", http.MethodGet, gross, "abc", true, http.StatusBadRequest, "keine gültige Kennung"},
		{"Vorschau ohne Quelle", http.MethodGet, gross, "", true, http.StatusBadRequest, "Kennung des anderen Titels"},
		{"Ziel ist keine Kennung", http.MethodGet, "abc", klein, true, http.StatusBadRequest, "Kennung des Titels"},
		{"derselbe Titel", http.MethodPost, gross, gross, true, http.StatusBadRequest, "mit sich selbst"},
		{"unbekannter Titel", http.MethodGet, gross, "00000000-0000-0000-0000-000000000001", true, http.StatusNotFound, tuer.MeldungTitelNichtGefunden},
		{"zwei ISBN", http.MethodPost, fremd, klein, true, http.StatusConflict, "Andere Auflage zuordnen"},
	} {
		t.Run(f.name, func(t *testing.T) {
			status, koerper := aufruf(f.methode, f.ziel, f.quelle, f.angemeldet)
			if status != f.status || !strings.Contains(koerper, f.text) {
				t.Errorf("Status %d, Antwort %s; erwartet %d mit %q", status, koerper, f.status, f.text)
			}
		})
	}
	if exemplareAm(klein) != 1 {
		t.Fatalf("eine Ablehnung hat Exemplare bewegt")
	}

	status, koerper := aufruf(http.MethodGet, gross, klein, true)
	var vorschau repository.TitelZusammenfuehrung
	if err := json.Unmarshal([]byte(koerper), &vorschau); status != http.StatusOK || err != nil {
		t.Fatalf("Vorschau: Status %d (%v) — %s", status, err, koerper)
	}
	if vorschau.Exemplare != 1 || vorschau.QuelleISBN != "9783060158423" || exemplareAm(klein) != 1 {
		t.Errorf("Vorschau %+v; am kleinen Titel %d Exemplare, erwartet die Vorschau ohne Wirkung", vorschau, exemplareAm(klein))
	}

	status, koerper = aufruf(http.MethodPost, gross, klein, true)
	if status != http.StatusOK {
		t.Fatalf("Zusammenführen: Status %d — %s", status, koerper)
	}
	if exemplareAm(gross) != 2 {
		t.Errorf("am großen Titel %d Exemplare, erwartet 2", exemplareAm(gross))
	}
	var eintraege int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = 'TITEL_ZUSAMMENGEFUEHRT'
		AND details->>'titel_id' = $1 AND details->>'aufgeloest_id' = $2`, gross, klein).Scan(&eintraege); err != nil {
		t.Fatal(err)
	}
	if eintraege != 1 {
		t.Errorf("%d Einträge im Protokoll der Verwaltung, erwartet 1", eintraege)
	}
}
