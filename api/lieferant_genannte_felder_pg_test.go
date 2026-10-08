package api

import (
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

	// Anlegen und Ändern über die Helfer des Pakets (lieferant_zweitnummer_pg_test.go); beide
	// brechen bei einer Ablehnung ab.
	anlegen := func(t *testing.T, name string, haupt bool) string {
		t.Helper()
		return lieferantAnlegenUeberHandler(t, srv, `{"name":"`+name+`","email":"`+strings.ToLower(name)+
			`@test.invalid","customerNumber":"K-`+name+`","kundennummer_schultraeger":"S-`+name+
			`","ist_hauptlieferant":`+map[bool]string{true: "true", false: "false"}[haupt]+`}`).ID
	}
	// abgelehnt schickt eine Änderung, die die Tür ablehnen soll, und liefert ihren Status.
	abgelehnt := func(t *testing.T, id, rumpf string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/"+id, strings.NewReader(rumpf))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateSupplierHandler()(rec, req)
		return rec.Code
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
	lieferantAendernUeberHandler(t, srv, neu, `{"ist_hauptlieferant":true}`)
	if a, n := lies(t, alt), lies(t, neu); a.haupt || !n.haupt || n.name != "NGF-Neu" || n.zweitnummer != "S-NGF-Neu" {
		t.Fatalf("nach dem Umhängen: alt %+v, neu %+v — erwartet das Merkmal beim zweiten, sonst nichts geändert", a, n)
	}

	// Die Maske vom Morgen (der erste trug das Merkmal noch) korrigiert nur seine E-Mail.
	antwort := lieferantAendernUeberHandler(t, srv, alt, `{"email":"korrigiert@test.invalid"}`)
	if a, n := lies(t, alt), lies(t, neu); a.email != "korrigiert@test.invalid" || a.haupt || !n.haupt ||
		a.name != "NGF-Alt" || a.nummer != "K-NGF-Alt" || a.zweitnummer != "S-NGF-Alt" {
		t.Errorf("nach dem Korrigieren der E-Mail: alt %+v, neu %+v — das Merkmal bleibt beim zweiten, die übrigen Felder stehen", a, n)
	}
	// Die Antwort nennt den gespeicherten Stand, auch für Felder, die der Rumpf nicht nannte.
	if antwort.Name != "NGF-Alt" || antwort.Email != "korrigiert@test.invalid" || antwort.CustomerNumber != "K-NGF-Alt" ||
		antwort.KundennummerSchultraeger != "S-NGF-Alt" || antwort.IstHauptlieferant {
		t.Errorf("die Antwort nennt nicht den gespeicherten Stand: %+v", antwort)
	}

	t.Run("ein Rumpf ohne Feld ändert nichts", func(t *testing.T) {
		vorher := lies(t, alt)
		lieferantAendernUeberHandler(t, srv, alt, `{}`)
		if nachher := lies(t, alt); nachher != vorher {
			t.Errorf("der Stand hat sich geändert: %+v, vorher %+v", nachher, vorher)
		}
	})

	t.Run("ein genanntes Pflichtfeld darf nicht leer sein, ein unbekanntes Feld wird abgelehnt", func(t *testing.T) {
		vorher := lies(t, alt)
		for _, rumpf := range []string{`{"name":""}`, `{"email":""}`, `{"customerNumber":""}`, `{"emial":"vertippt@test.invalid"}`,
			`{"name":"NGF-Anders","hauptlieferant":true}`} {
			if code := abgelehnt(t, alt, rumpf); code != http.StatusBadRequest {
				t.Errorf("%s: Status %d, erwartet 400", rumpf, code)
			}
		}
		if nachher := lies(t, alt); nachher != vorher {
			t.Errorf("eine abgelehnte Anfrage hat geschrieben: %+v, vorher %+v", nachher, vorher)
		}
	})

	t.Run("eine leere zweite Kundennummer heißt dieselbe wie die erste", func(t *testing.T) {
		lieferantAendernUeberHandler(t, srv, alt, `{"kundennummer_schultraeger":" "}`)
		if s := lies(t, alt); s.zweitnummer != "" || s.nummer != "K-NGF-Alt" {
			t.Errorf("nach dem Leeren: %+v", s)
		}
	})

	// Merkmal und Stammdaten sind eine Transaktion: Der Setzer räumt zuerst den bisherigen
	// Hauptlieferanten. Gibt es den genannten Lieferanten nicht, bleibt das Merkmal, wo es war.
	t.Run("ein unbekannter Lieferant nimmt niemandem das Merkmal", func(t *testing.T) {
		if code := abgelehnt(t, "00000000-0000-4000-8000-000000000000", `{"ist_hauptlieferant":true,"name":"NGF-Niemand"}`); code != http.StatusNotFound {
			t.Errorf("Status %d, erwartet 404", code)
		}
		if n := lies(t, neu); !n.haupt {
			t.Errorf("der Hauptlieferant hat sein Merkmal verloren, obwohl die Änderung abgelehnt wurde")
		}
	})

	t.Run("das Merkmal lässt sich abwählen, ohne es jemand anderem zu geben", func(t *testing.T) {
		lieferantAendernUeberHandler(t, srv, neu, `{"ist_hauptlieferant":false}`)
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM lieferanten WHERE ist_hauptlieferant`); n != 0 {
			t.Errorf("%d Hauptlieferanten nach dem Abwählen, erwartet 0", n)
		}
	})
}
