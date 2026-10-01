package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Die Akte nennt den offenen Betrag eines Lesers im Kopf, in jedem Reiter. Er kommt mit dem
// Profil und aus derselben Abfrage, nach der die Theke beim Ausleihen warnt
// (repository.OffeneSchaeden). Bezahlte und stornierte Forderungen zählen nicht.
func TestProfilNenntOffeneForderungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Forderungs-Titel') RETURNING id::text`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	leser := func(t *testing.T, nachname string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, klasse, abgaenger_jahr, barcode_id, geburtsdatum)
			VALUES ('Forderung', $1, 'schueler', '07A', 2031, $2, '2012-05-04') RETURNING id::text`,
			nachname, "A-FORD-"+nachname).Scan(&id); err != nil {
			t.Fatalf("Leser anlegen: %v", err)
		}
		return id
	}
	// Jede Forderung hängt an einem eigenen Exemplar (check_damage_item).
	forderung := func(t *testing.T, leserID, barcode string, betrag float64, bezahlt, storniert bool) {
		t.Helper()
		var exemplarID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id::text`,
			titelID, barcode).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (schueler_id, exemplar_id, beschreibung, betrag, ist_bezahlt, storniert_am)
			VALUES ($1, $2, 'Probe', $3, $4, CASE WHEN $5 THEN NOW() END)`,
			leserID, exemplarID, betrag, bezahlt, storniert); err != nil {
			t.Fatalf("Forderung anlegen: %v", err)
		}
	}

	mitID := leser(t, "Mit")
	forderung(t, mitID, "B-FORD-1", 12.00, false, false)
	forderung(t, mitID, "B-FORD-2", 7.50, false, false)
	forderung(t, mitID, "B-FORD-3", 30.00, true, false)
	// Ein Storno setzt ist_bezahlt mit (repository/audit_system.go).
	forderung(t, mitID, "B-FORD-4", 4.00, true, true)
	ohneID := leser(t, "Ohne")

	profil := func(t *testing.T, id string) StudentProfileResponse {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.GetStudentProfileHandler(repository.NewStudentRepository(pool))(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Profil %s: %d %s", id, rec.Code, rec.Body.String())
		}
		var antwort StudentProfileResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Profil lesen: %v", err)
		}
		return antwort
	}

	t.Run("zwei offene Forderungen, Summe ohne Bezahltes und Storniertes", func(t *testing.T) {
		p := profil(t, mitID)
		if p.OffeneForderungen.Anzahl != 2 || p.OffeneForderungen.Summe != 19.50 {
			t.Errorf("offene Forderungen = %+v, erwartet 2 über 19,50", p.OffeneForderungen)
		}
		if !p.HasOpenDamages {
			t.Error("has_open_damages ist false, obwohl zwei Forderungen offen sind")
		}
	})

	// Ohne Zeile liefert SUM in Postgres NULL — die Akte muss trotzdem laden.
	t.Run("ohne Forderung steht null über null", func(t *testing.T) {
		p := profil(t, ohneID)
		if p.OffeneForderungen.Anzahl != 0 || p.OffeneForderungen.Summe != 0 {
			t.Errorf("offene Forderungen = %+v, erwartet 0 über 0", p.OffeneForderungen)
		}
		if p.HasOpenDamages {
			t.Error("has_open_damages ist true ohne Forderung")
		}
	})
}
