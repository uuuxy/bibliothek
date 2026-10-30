package druck_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"bibliothek/api/druck"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GET und PUT /api/ausweis-layout: Das Design der Ausweise liegt am Server, damit alle
// Arbeitsplätze dasselbe drucken.
//
// Über den ganzen Router mit Sitzung. Die Zusagen: Was gespeichert ist, kommt Byte für Byte
// zurück, hinter dem Kopf der Schule. Was die Tür abweist, lässt das gespeicherte Design stehen.
// Und „noch kein Design" antwortet sie nur, wenn wirklich keines gespeichert ist: Gedruckt würde
// sonst mit den Vorgabewerten.

// kopfDerSchule ist der Teil der Antwort vor dem Design (designMitSchule).
var kopfDerSchule = regexp.MustCompile(`^\{"schule":\{[^{}]*\}`)

// ohneKopf liefert die Antwort ohne den Kopf der Schule: das Design, wie es gespeichert ist.
func ohneKopf(t *testing.T, rumpf string) string {
	t.Helper()
	kopf := kopfDerSchule.FindString(rumpf)
	if kopf == "" {
		t.Fatalf("Antwort ohne Kopf der Schule: %.80s", rumpf)
	}
	rest := strings.TrimPrefix(rumpf[len(kopf):], ",")
	if rest == "}" {
		return "{}"
	}
	return "{" + rest
}

// poolMitLesefehlerBei lässt die Abfrage scheitern, deren SQL das Muster und deren erstes
// Argument den Schlüssel trägt. Alles andere (Sitzung, Rechte) läuft am echten Pool.
type poolMitLesefehlerBei struct {
	db.Pool
	muster, schluessel string
}

type zeileMitFehler struct{ err error }

func (z zeileMitFehler) Scan(...any) error { return z.err }

func (p poolMitLesefehlerBei) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, p.muster) && len(args) > 0 && args[0] == p.schluessel {
		return zeileMitFehler{errors.New("testfehler: verbindung zur datenbank abgerissen")}
	}
	return p.Pool.QueryRow(ctx, sql, args...)
}

func TestAusweisLayout_UeberDieTuer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	_, rufe := routertest.MitAufruf(t, pool)
	ohneDesign := func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, repository.AusweisLayoutSchluessel)
	}
	ohneDesign()
	t.Cleanup(ohneDesign)

	const pfad = "/api/ausweis-layout"
	lade := func(t *testing.T) string {
		t.Helper()
		rec := rufe(t, http.MethodGet, pfad, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("Laden: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		return ohneKopf(t, rec.Body.String())
	}
	speichere := func(t *testing.T, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(t, http.MethodPut, pfad, rumpf)
	}
	// abgewiesen prüft den Status und dass das gespeicherte Design dasselbe geblieben ist.
	abgewiesen := func(t *testing.T, rumpf string, status int) *httptest.ResponseRecorder {
		t.Helper()
		vorher := lade(t)
		rec := speichere(t, rumpf)
		if rec.Code != status {
			t.Errorf("Status %d, erwartet %d: %.200s", rec.Code, status, rec.Body.String())
		}
		if nachher := lade(t); nachher != vorher {
			t.Errorf("die abgewiesene Anfrage hat das Design verändert: %.80s → %.80s", vorher, nachher)
		}
		return rec
	}

	t.Run("nichts gespeichert: leeres Objekt", func(t *testing.T) {
		if rumpf := lade(t); rumpf != "{}" {
			t.Errorf("Antwort %q, erwartet {}", rumpf)
		}
	})

	// Eine Zeile ohne Inhalt ist dasselbe wie keine Zeile: Die Antwort muss JSON bleiben.
	t.Run("leere Zeile: leeres Objekt", func(t *testing.T) {
		if _, err := pool.Exec(t.Context(),
			`INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, '  ')`, repository.AusweisLayoutSchluessel); err != nil {
			t.Fatal(err)
		}
		if rumpf := lade(t); rumpf != "{}" {
			t.Errorf("Antwort %q, erwartet {}", rumpf)
		}
	})

	// Das Design geht als Text in die Zeile; eine andere Reihenfolge der Schlüssel oder ein
	// verlorenes Leerzeichen wäre schon ein anderes Design.
	const design = `{"printMode":"karte", "front":[{"id":"e1","x":3.5,"text":"Büchereiausweis"}],"back":[]}`
	t.Run("gespeichert kommt Byte für Byte zurück", func(t *testing.T) {
		if rec := speichere(t, design); rec.Code != http.StatusOK {
			t.Fatalf("Speichern: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if rumpf := lade(t); rumpf != design {
			t.Errorf("geladen %q, gespeichert %q", rumpf, design)
		}
	})

	// Wer Ausweise druckt, darf die Einstellungen oft nicht lesen; auf die Karte gehört die Schule
	// trotzdem. Kein Design gespeichert heißt dabei nicht: keine Schule.
	t.Run("der Kopf nennt Schule und Anschrift", func(t *testing.T) {
		zuruecksetzen := func() {
			pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = ANY($1)`,
				[]string{"schule_name", "schule_strasse", "schule_plz", "schule_ort"})
		}
		zuruecksetzen()
		t.Cleanup(zuruecksetzen)
		for schluessel, wert := range map[string]string{
			"schule_name": "Musterschule", "schule_strasse": "Schulweg 1", "schule_plz": "12345", "schule_ort": "Neustadt",
		} {
			if _, err := pool.Exec(t.Context(),
				`INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)`, schluessel, wert); err != nil {
				t.Fatal(err)
			}
		}
		rec := rufe(t, http.MethodGet, pfad, "")
		var antwort struct {
			Schule    struct{ Name, Adresse string } `json:"schule"`
			PrintMode string                         `json:"printMode"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v: %.200s", err, rec.Body.String())
		}
		if antwort.Schule.Name != "Musterschule" || antwort.Schule.Adresse != "Schulweg 1, 12345 Neustadt" {
			t.Errorf("Kopf %+v", antwort.Schule)
		}
		if antwort.PrintMode != "karte" {
			t.Errorf("printMode %q: Das Design fehlt hinter dem Kopf", antwort.PrintMode)
		}
	})

	t.Run("ein zweites Speichern ersetzt das erste", func(t *testing.T) {
		const zweites = `{"printMode":"etikett","front":[],"back":[]}`
		if rec := speichere(t, zweites); rec.Code != http.StatusOK {
			t.Fatalf("Speichern: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if rumpf := lade(t); rumpf != zweites {
			t.Errorf("geladen %q, gespeichert %q", rumpf, zweites)
		}
		if n := pgtest.ZaehleZeilen(t, pool,
			`SELECT count(*) FROM system_einstellungen WHERE schluessel = $1`, repository.AusweisLayoutSchluessel); n != 1 {
			t.Errorf("%d Zeilen für das Design, erwartet 1", n)
		}
		if rec := speichere(t, design); rec.Code != http.StatusOK {
			t.Fatalf("Speichern: Status %d", rec.Code)
		}
	})

	t.Run("kein JSON: 400, das Design bleibt", func(t *testing.T) {
		abgewiesen(t, `{"printMode":`, http.StatusBadRequest)
		abgewiesen(t, ``, http.StatusBadRequest)
	})

	// Logos stehen als Base64 im Design; es darf Megabytes groß sein, aber nicht beliebig.
	logo := func(bytes int) string {
		return `{"logo":"` + strings.Repeat("A", bytes) + `"}`
	}
	t.Run("ein Design mit großem Logo wird gespeichert", func(t *testing.T) {
		gross := logo(4 << 20)
		if rec := speichere(t, gross); rec.Code != http.StatusOK {
			t.Fatalf("Speichern: Status %d, erwartet 200: %.200s", rec.Code, rec.Body.String())
		}
		if rumpf := lade(t); rumpf != gross {
			t.Errorf("das große Design kam nicht unverändert zurück (%d statt %d Bytes)", len(rumpf), len(gross))
		}
	})
	t.Run("über der Grenze: 413, das Design bleibt", func(t *testing.T) {
		rec := abgewiesen(t, logo(druck.MaxAusweisLayoutBytes), http.StatusRequestEntityTooLarge)
		if got := tuertest.Fehlermeldung(t, rec); got != "Ausweis-Design ist zu groß" {
			t.Errorf("AusweisLayoutSchreiben: Meldung %q, erwartet %q", got, "Ausweis-Design ist zu groß")
		}
	})
}

// Scheitert das Lesen des Designs, darf die Antwort nicht „noch kein Design" heißen. Der
// Designer hielte den Ausfall für den ersten Start und speicherte seine Vorgabewerte über
// das Design der Schule.
func TestAusweisLayout_LesefehlerIstKeinLeeresDesign(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	const design = `{"printMode":"karte","front":[{"id":"e1"}],"back":[]}`
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AusweisLayoutSchluessel, design); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, repository.AusweisLayoutSchluessel)
	})

	gestoert := poolMitLesefehlerBei{Pool: pool, muster: "FROM system_einstellungen", schluessel: repository.AusweisLayoutSchluessel}
	_, sitzung, router := routertest.MitSitzungUeber(t, pool, gestoert,
		"ausweis-layout-lesefehler@example.org", "Lea", "Lesefehler")
	req := httptest.NewRequest(http.MethodGet, "/api/ausweis-layout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) == "{}" {
		t.Error("ein Lesefehler wird als leeres Design ausgeliefert")
	}

	// Gegenprobe: Derselbe Aufruf am ungestörten Pool liefert das Design. Ohne sie bestünde
	// der Test auch an einer Tür, die immer 500 antwortet.
	_, sitzung, router = routertest.MitSitzung(t, pool,
		"ausweis-layout-lesefehler@example.org", "Lea", "Lesefehler")
	req = httptest.NewRequest(http.MethodGet, "/api/ausweis-layout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || ohneKopf(t, rec.Body.String()) != design {
		t.Errorf("ungestört: Status %d, Antwort %q, erwartet das gespeicherte Design", rec.Code, rec.Body.String())
	}
}
