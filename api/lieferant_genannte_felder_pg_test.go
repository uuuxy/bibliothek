package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Die Tür „Lieferant ändern" schreibt nur, was der Rumpf nennt.
//
// Die Zeile der Lieferantentabelle füllt ihre Maske aus der Liste, die beim Öffnen der Seite
// lädt, und schickte beim Speichern jedes Feld zurück, auch das Merkmal Hauptlieferant. Hatte
// inzwischen ein anderer Platz einen anderen Händler zum Hauptlieferanten gemacht, nahm das
// Korrigieren einer E-Mail ihm das Merkmal wieder. Merkmal und Stammdaten schrieb die Tür in
// zwei Schritten. Geprüft über die Handler am echten Postgres.
func TestLieferantAendern_SchreibtNurDieGenanntenFelder(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	srv := &Server{DB: &db.Database{Pool: pool}}
	// Andere Tests im selben Paketlauf lassen Lieferanten stehen, auch einen Hauptlieferanten.
	aufraeumen(t, pool, `UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant`)
	t.Cleanup(func() { aufraeumen(t, pool, `DELETE FROM lieferanten WHERE name LIKE 'NGF-%'`) })

	anlegen := func(t *testing.T, name string, haupt bool) string {
		t.Helper()
		rumpf := `{"name":"` + name + `","email":"` + strings.ToLower(name) + `@test.invalid","customerNumber":"K-` + name +
			`","kundennummer_schultraeger":"S-` + name + `","ist_hauptlieferant":` + map[bool]string{true: "true", false: "false"}[haupt] + `}`
		rec := httptest.NewRecorder()
		srv.CreateSupplierHandler()(rec, httptest.NewRequest(http.MethodPost, "/api/lieferanten", strings.NewReader(rumpf)))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s anlegen: Status %d — %s", name, rec.Code, rec.Body.String())
		}
		var antwort SupplierResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		return antwort.ID
	}
	aendere := func(t *testing.T, id, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/"+id, strings.NewReader(rumpf))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateSupplierHandler()(rec, req)
		return rec
	}
	type stand struct {
		name, email, nummer, zweitnummer string
		haupt                            bool
	}
	lies := func(t *testing.T, id string) stand {
		t.Helper()
		var s stand
		if err := pool.QueryRow(ctx, `
			SELECT name, email, kundennummer, kundennummer_schultraeger, ist_hauptlieferant
			FROM lieferanten WHERE id = $1`, id).Scan(&s.name, &s.email, &s.nummer, &s.zweitnummer, &s.haupt); err != nil {
			t.Fatalf("Stand lesen: %v", err)
		}
		return s
	}

	alt := anlegen(t, "NGF-Alt", true)
	neu := anlegen(t, "NGF-Neu", false)

	// Ein anderer Platz macht den zweiten Händler zum Hauptlieferanten; er nennt nur das Merkmal.
	if rec := aendere(t, neu, `{"ist_hauptlieferant":true}`); rec.Code != http.StatusOK {
		t.Fatalf("das Merkmal allein setzen: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}
	if a, n := lies(t, alt), lies(t, neu); a.haupt || !n.haupt || n.name != "NGF-Neu" || n.zweitnummer != "S-NGF-Neu" {
		t.Fatalf("nach dem Umhängen: alt %+v, neu %+v — erwartet das Merkmal beim zweiten, sonst nichts geändert", a, n)
	}

	// Die Maske vom Morgen (der erste trug das Merkmal noch) korrigiert nur seine E-Mail.
	rec := aendere(t, alt, `{"email":"korrigiert@test.invalid"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("die E-Mail allein ändern: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}
	if a, n := lies(t, alt), lies(t, neu); a.email != "korrigiert@test.invalid" || a.haupt || !n.haupt ||
		a.name != "NGF-Alt" || a.nummer != "K-NGF-Alt" || a.zweitnummer != "S-NGF-Alt" {
		t.Errorf("nach dem Korrigieren der E-Mail: alt %+v, neu %+v — das Merkmal bleibt beim zweiten, die übrigen Felder stehen", a, n)
	}
	// Die Antwort nennt den gespeicherten Stand, auch für Felder, die der Rumpf nicht nannte.
	var antwort SupplierResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Name != "NGF-Alt" || antwort.Email != "korrigiert@test.invalid" || antwort.CustomerNumber != "K-NGF-Alt" ||
		antwort.KundennummerSchultraeger != "S-NGF-Alt" || antwort.IstHauptlieferant {
		t.Errorf("die Antwort nennt nicht den gespeicherten Stand: %+v", antwort)
	}

	t.Run("ein Rumpf ohne Feld ändert nichts", func(t *testing.T) {
		vorher := lies(t, alt)
		if rec := aendere(t, alt, `{}`); rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200", rec.Code)
		}
		if nachher := lies(t, alt); nachher != vorher {
			t.Errorf("der Stand hat sich geändert: %+v, vorher %+v", nachher, vorher)
		}
	})

	t.Run("ein genanntes Pflichtfeld darf nicht leer sein, ein unbekanntes Feld wird abgelehnt", func(t *testing.T) {
		vorher := lies(t, alt)
		for _, rumpf := range []string{`{"name":""}`, `{"email":""}`, `{"customerNumber":""}`, `{"emial":"vertippt@test.invalid"}`,
			`{"name":"NGF-Anders","hauptlieferant":true}`} {
			if rec := aendere(t, alt, rumpf); rec.Code != http.StatusBadRequest {
				t.Errorf("%s: Status %d, erwartet 400", rumpf, rec.Code)
			}
		}
		if nachher := lies(t, alt); nachher != vorher {
			t.Errorf("eine abgelehnte Anfrage hat geschrieben: %+v, vorher %+v", nachher, vorher)
		}
	})

	t.Run("eine leere zweite Kundennummer heißt dieselbe wie die erste", func(t *testing.T) {
		if rec := aendere(t, alt, `{"kundennummer_schultraeger":" "}`); rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200", rec.Code)
		}
		if s := lies(t, alt); s.zweitnummer != "" || s.nummer != "K-NGF-Alt" {
			t.Errorf("nach dem Leeren: %+v", s)
		}
	})

	// Merkmal und Stammdaten sind eine Transaktion: Der Setzer räumt zuerst den bisherigen
	// Hauptlieferanten. Gibt es den genannten Lieferanten nicht, bleibt das Merkmal, wo es war.
	t.Run("ein unbekannter Lieferant nimmt niemandem das Merkmal", func(t *testing.T) {
		rec := aendere(t, "00000000-0000-4000-8000-000000000000", `{"ist_hauptlieferant":true,"name":"NGF-Niemand"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Status %d, erwartet 404 — %s", rec.Code, rec.Body.String())
		}
		if n := lies(t, neu); !n.haupt {
			t.Errorf("der Hauptlieferant hat sein Merkmal verloren, obwohl die Änderung abgelehnt wurde")
		}
	})

	t.Run("das Merkmal lässt sich abwählen, ohne es jemand anderem zu geben", func(t *testing.T) {
		if rec := aendere(t, neu, `{"ist_hauptlieferant":false}`); rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200", rec.Code)
		}
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM lieferanten WHERE ist_hauptlieferant`); n != 0 {
			t.Errorf("%d Hauptlieferanten nach dem Abwählen, erwartet 0", n)
		}
	})
}
