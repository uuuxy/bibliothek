package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Die Art eines Lesers (Migration 123) über die Akte ändern — und die eine Grenze, die
// dabei zu halten ist.
//
// Peter am 16.09.2026: „hier steht nirgends ob jemand ein Schüler, LiV, oder lehrer ist."
// Die Art stand in der Datenbank, im Anlege-Dialog und in der halben Akte, aber der PATCH
// kannte sie nicht — sie liess sich nirgends berichtigen. Ein Kollege, der als Lehrkraft
// angelegt wurde und in Wahrheit LiV ist, blieb es für immer.
//
// Die Grenze verläuft NICHT zwischen Lehrkraft und LiV, sondern am Schüler:
//
//   - Schüler -> Kollege nimmt der Zeile ihre LUSD-Bindung. Mit lusd_id bricht
//     chk_leser_nur_schueler_werden_abgaenger; ohne sie liefe der Schüler beim nächsten
//     Import als unbekannter Name durch die Abgänger-Behandlung samt Anonymisierung.
//   - Kollege -> Schüler bricht chk_leser_schueler_pflichtfelder: Klasse, Abgängerjahr
//     und Ausweisnummer sind für einen Schüler Pflicht, und ein Kollege hat sie nicht.
//
// Beide Richtungen müssen als AUSKUNFT zurückkommen (400), nicht als CHECK-Verletzung.
// Eine 500 macht der Sanitizer zu „Ein interner Datenbankfehler ist aufgetreten" — der
// Satz sagt niemandem, was zu tun ist, und derselbe Fehler kostete am 22.08.2026 schon
// einmal die Diagnose (siehe api/student_update.go, Kommentar an den Sperrfeldern).
func TestLeserArtAendern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	patch := func(t *testing.T, id, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
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
		var id string
		// In die TABELLE, nicht in die Sicht: `schueler` ist auf art='schueler'
		// eingeschraenkt und weist einen Kollegen mit "check option for view" ab.
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (vorname, nachname, art) VALUES ('Art', $1, $2) RETURNING id`,
			name, art).Scan(&id); err != nil {
			t.Fatalf("Kollegen anlegen: %v", err)
		}
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

	// Der Kern: KEINE 500. Der Handler muss die Grenze selbst kennen, statt sie dem
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

	t.Run("eine vierte Art ist ein Tippfehler", func(t *testing.T) {
		id := legeKollegen(t, "Tippfehler", "lehrkraft")
		if rec := patch(t, id, `{"art":"referendar"}`); rec.Code != http.StatusBadRequest {
			t.Fatalf("PATCH gab %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
	})

	// Das Formular schickt die Art bei JEDEM Speichern mit, auch unveraendert. Waere das
	// kein No-op, liefe jedes Speichern eines Schuelers in die Ablehnung oben und die
	// Akte liesse sich gar nicht mehr bearbeiten.
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

	// Peters eigentlicher Befund: Wer eine Rolle hat, steht seit Migration 125 mit einer
	// Leserzeile in der Liste — und an ihr liess sich nichts nachtragen. Der Rumpf hier
	// ist GENAU der, den das Formular fuer einen Kollegen baut: ohne klasse, ohne
	// abgaenger_jahr, ohne lusd_id, ohne eltern_email. Mit ihnen (leer) antwortete der
	// Server „Klasse darf nicht leer sein."
	t.Run("Adresse eines Kollegen nachtragen", func(t *testing.T) {
		id := legeKollegen(t, "Adresse", "lehrkraft")
		rec := patch(t, id, `{"vorname":"Art","nachname":"Adresse","art":"lehrkraft",
			"geburtsdatum":null,"strasse":"Kleegartenstr.","hausnummer":"8",
			"plz":"61381","ort":"Friedrichsdorf"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
		}
		var strasse, plz, ort string
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE(strasse,''), COALESCE(plz,''), COALESCE(ort,'')
			   FROM leser WHERE id = $1`, id).Scan(&strasse, &plz, &ort); err != nil {
			t.Fatalf("Adresse lesen: %v", err)
		}
		if strasse != "Kleegartenstr." || plz != "61381" || ort != "Friedrichsdorf" {
			t.Fatalf("Adresse steht als %q %q %q in der Datenbank", strasse, plz, ort)
		}
	})
}
