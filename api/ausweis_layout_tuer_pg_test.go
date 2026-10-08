package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GET und PUT /api/ausweis-layout: Das Design der Ausweise liegt am Server, damit alle
// Arbeitsplätze dasselbe drucken. Gemessen am 08.10.2026 führte kein Go-Test das Speichern
// aus (api/ausweis_layout.go 33,3 %, OFFEN.md 5.10).
//
// Über den ganzen Router mit Sitzung. Die Zusagen: Was gespeichert ist, kommt Byte für Byte
// zurück. Was die Tür abweist, lässt das gespeicherte Design stehen. Und „noch kein Design"
// antwortet sie nur, wenn wirklich keines gespeichert ist: Der Designer speichert nach einer
// solchen Antwort seine Vorgabewerte, für alle Arbeitsplätze.

// poolMitLesefehlerBei lässt die Abfrage scheitern, deren SQL das Muster und deren erstes
// Argument den Schlüssel trägt. Alles andere (Sitzung, Rechte) läuft am echten Pool.
type poolMitLesefehlerBei struct {
	db.PgxPoolIface
	muster, schluessel string
}

type zeileMitFehler struct{ err error }

func (z zeileMitFehler) Scan(...any) error { return z.err }

func (p poolMitLesefehlerBei) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, p.muster) && len(args) > 0 && args[0] == p.schluessel {
		return zeileMitFehler{errors.New("Testfehler: Verbindung zur Datenbank abgerissen")}
	}
	return p.PgxPoolIface.QueryRow(ctx, sql, args...)
}

func TestAusweisLayout_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	_, rufe := protokollWelt(t, pool)
	ohneDesign := func() {
		aufraeumen(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, repository.AusweisLayoutSchluessel)
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
		return rec.Body.String()
	}
	speichere := func(t *testing.T, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(t, http.MethodPut, pfad, rumpf)
	}
	// abgewiesen prüft den Status und dass das gespeicherte Design dasselbe geblieben ist.
	abgewiesen := func(t *testing.T, rumpf string, status int) {
		t.Helper()
		vorher := lade(t)
		rec := speichere(t, rumpf)
		if rec.Code != status {
			t.Errorf("Status %d, erwartet %d: %.200s", rec.Code, status, rec.Body.String())
		}
		if nachher := lade(t); nachher != vorher {
			t.Errorf("die abgewiesene Anfrage hat das Design verändert: %.80s → %.80s", vorher, nachher)
		}
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

	t.Run("ein zweites Speichern ersetzt das erste", func(t *testing.T) {
		const zweites = `{"printMode":"etikett","front":[],"back":[]}`
		if rec := speichere(t, zweites); rec.Code != http.StatusOK {
			t.Fatalf("Speichern: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if rumpf := lade(t); rumpf != zweites {
			t.Errorf("geladen %q, gespeichert %q", rumpf, zweites)
		}
		if n := zaehleZeilen(t, pool,
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
		abgewiesen(t, logo(maxAusweisLayoutBytes), http.StatusRequestEntityTooLarge)
	})
}

// Scheitert das Lesen des Designs, darf die Antwort nicht „noch kein Design" heißen. Der
// Designer hielte den Ausfall für den ersten Start und speicherte seine Vorgabewerte über
// das Design der Schule.
func TestAusweisLayout_LesefehlerIstKeinLeeresDesign(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	const design = `{"printMode":"karte","front":[{"id":"e1"}],"back":[]}`
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AusweisLayoutSchluessel, design); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, repository.AusweisLayoutSchluessel)
	})

	gestoert := poolMitLesefehlerBei{PgxPoolIface: pool, muster: "FROM system_einstellungen", schluessel: repository.AusweisLayoutSchluessel}
	_, sitzung, router := routerMitSitzungUeber(t, pool, gestoert,
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
	_, sitzung, router = routerMitSitzung(t, pool,
		"ausweis-layout-lesefehler@example.org", "Lea", "Lesefehler")
	req = httptest.NewRequest(http.MethodGet, "/api/ausweis-layout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != design {
		t.Errorf("ungestört: Status %d, Antwort %q, erwartet das gespeicherte Design", rec.Code, rec.Body.String())
	}
}
