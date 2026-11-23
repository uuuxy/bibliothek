package leser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Die Art eines Lesers (Migration 123) über die Akte ändern — und die eine Grenze, die
// dabei zu halten ist.
//
// Die Art steht in der Datenbank, im Anlege-Dialog und in der Akte; kennte der PATCH sie
// nicht, ließe sie sich nirgends berichtigen, und ein Kollege, der als Lehrkraft angelegt
// wurde und in Wahrheit LiV ist, bliebe es für immer.
//
// Die Grenze verläuft nicht zwischen Lehrkraft und LiV, sondern am Schüler:
//
//   - Schüler -> Kollege nimmt der Zeile ihre LUSD-Bindung. Mit lusd_id bricht
//     chk_leser_nur_schueler_werden_abgaenger; ohne sie liefe der Schüler beim nächsten
//     Import als unbekannter Name durch die Abgänger-Behandlung samt Anonymisierung.
//   - Kollege -> Schüler bricht chk_leser_schueler_pflichtfelder: Klasse, Abgängerjahr
//     und Ausweisnummer sind für einen Schüler Pflicht, und ein Kollege hat sie nicht.
//
// Beide Richtungen müssen als Auskunft zurückkommen (400), nicht als CHECK-Verletzung.
// Eine 500 macht der Sanitizer zu „Ein interner Serverfehler ist aufgetreten"; der Satz
// sagt niemandem, was zu tun ist (siehe internal/api/leser/leser_aendern.go, Kommentar an den
// Sperrfeldern).
func TestLeserArtAendern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	patch := func(t *testing.T, id, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: id, Rolle: auth.RolleAdmin}))
		rec := httptest.NewRecorder()
		srv.LeserAendern(repository.NewProtokollAbfragen(pool))(rec, req)
		return rec
	}

	artVon := func(t *testing.T, id string) string {
		t.Helper()
		var art string
		if err := pool.QueryRow(ctx, "SELECT art FROM leser WHERE id = $1", id).Scan(&art); err != nil {
			t.Fatalf("Art lesen: %v", err)
		}
		return art
	}

	legeKollegen := func(t *testing.T, name, art string) string {
		t.Helper()
		id := testdaten.LegeLeserAn(t, pool, "", "Art", name, art)
		return id
	}

	legeSchueler := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (vorname, nachname, art, klasse, barcode_id, abgaenger_jahr)
			 VALUES ('Art','Schueler','schueler','07a',$1,2030) RETURNING id`,
			barcode).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		return id
	}

	t.Run("Lehrkraft wird LiV", func(t *testing.T) {
		id := legeKollegen(t, "Wechsel", "lehrkraft")
		rec := patch(t, id, `{"art":"liv"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
		}
		if got := artVon(t, id); got != "liv" {
			t.Fatalf("Art ist %q, erwartet %q", got, "liv")
		}
	})

	t.Run("LiV wird Lehrkraft", func(t *testing.T) {
		id := legeKollegen(t, "Zurueck", "liv")
		if rec := patch(t, id, `{"art":"lehrkraft"}`); rec.Code != http.StatusOK {
			t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
		}
		if got := artVon(t, id); got != "lehrkraft" {
			t.Fatalf("Art ist %q, erwartet %q", got, "lehrkraft")
		}
	})

	// Der Kern: keine 500. Die Tür muss die Grenze selbst kennen, statt sie dem
	// CHECK zu überlassen.
	t.Run("Schueler wird nicht zur Lehrkraft", func(t *testing.T) {
		id := legeSchueler(t, "ART-S1")
		rec := patch(t, id, `{"art":"lehrkraft"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PATCH gab %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if got := artVon(t, id); got != "schueler" {
			t.Fatalf("Art wurde trotz Ablehnung auf %q geaendert", got)
		}
	})

	t.Run("Kollege wird nicht zum Schueler", func(t *testing.T) {
		id := legeKollegen(t, "BleibtKollege", "lehrkraft")
		rec := patch(t, id, `{"art":"schueler"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PATCH gab %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if got := artVon(t, id); got != "lehrkraft" {
			t.Fatalf("Art wurde trotz Ablehnung auf %q geaendert", got)
		}
	})

	// Ein Leser, den es nicht gibt, ist ein 404 mit seinem Satz und kein Serverfehler.
	t.Run("unbekannter Leser", func(t *testing.T) {
		rec := patch(t, "00000000-0000-4000-8000-00000000a47e", `{"art":"liv"}`)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Leser nicht gefunden") {
			t.Fatalf("PATCH gab %d, erwartet 404 mit dem Satz zum fehlenden Leser: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("eine vierte Art ist ein Tippfehler", func(t *testing.T) {
		id := legeKollegen(t, "Tippfehler", "lehrkraft")
		if rec := patch(t, id, `{"art":"referendar"}`); rec.Code != http.StatusBadRequest {
			t.Fatalf("PATCH gab %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
	})

	// Das Formular schickt die Art bei jedem Speichern mit, auch unverändert. Wäre das keine
	// leere Änderung, liefe jedes Speichern eines Schülers in die Ablehnung oben, und die Akte
	// ließe sich gar nicht bearbeiten.
	t.Run("dieselbe Art ist ein No-op und kein Fehler", func(t *testing.T) {
		id := legeSchueler(t, "ART-S2")
		rec := patch(t, id, `{"art":"schueler","vorname":"Neu"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
		}
		var vorname string
		if err := pool.QueryRow(ctx, "SELECT vorname FROM leser WHERE id = $1", id).Scan(&vorname); err != nil {
			t.Fatalf("Vorname lesen: %v", err)
		}
		if vorname != "Neu" {
			t.Fatalf("Vorname ist %q — das uebrige PATCH ist nicht durchgelaufen", vorname)
		}
	})

	// Der eigentliche Fall: Wer eine Rolle hat, steht seit Migration 125 mit einer Leserzeile
	// in der Liste, und an ihr muss sich etwas nachtragen lassen. Der Rumpf hier ist genau
	// der, den das Formular für einen Kollegen baut: ohne klasse, ohne abgaenger_jahr, ohne
	// lusd_id, ohne Anschrift und ohne eltern_email. Mit ihnen (leer) antwortete der Server
	// „Bitte das Feld „Klasse“ ausfüllen".
	t.Run("Geburtsdatum eines Kollegen nachtragen", func(t *testing.T) {
		id := legeKollegen(t, "Nachtrag", "lehrkraft")
		rec := patch(t, id, `{"vorname":"Art","nachname":"Nachtrag","art":"lehrkraft",
			"geburtsdatum":"1980-04-12"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
		}
		var geburtsdatum string
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE(TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), '') FROM leser WHERE id = $1`, id).Scan(&geburtsdatum); err != nil {
			t.Fatalf("Geburtsdatum lesen: %v", err)
		}
		if geburtsdatum != "1980-04-12" {
			t.Fatalf("Geburtsdatum steht als %q in der Datenbank", geburtsdatum)
		}
	})
}
