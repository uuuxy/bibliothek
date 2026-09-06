package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/pkg/schulzeit"
)

// Veröffentlichen, Korrigieren und Verwerfen eines Rückgabe-Plans schreiben die
// Rückgabefristen ALLER Klassen des Plans um und löschen dabei Mahnstufe und Mahndatum
// der betroffenen Ausleihen (repository.SetzeLernmittelFristFuerKlassen). Für die Frist
// EINER Ausleihe verlangt dieses Projekt seit dem 18.08.2026 einen Protokolleintrag
// („FRIST_OVERRIDE", api/ausleihe.go); die tausendfache Fassung desselben Eingriffs
// hinterließ bis zum Rasterdurchgang am 06.09.2026 keine Spur.
//
// Geprüft wird am ERGEBNIS in audit_logs, nicht am Aufruf: Wer hat wann welchen Plan
// bewegt, und wie viele Fristen hingen daran. Ein ENTWURF fasst keine Frist an und
// schreibt deshalb auch nichts — sonst stünde im Protokoll jeder Zwischenstand.
func TestLmfPlan_MassenaenderungDerFristenWirdProtokolliert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	// Echter Benutzer für den Audit-FK (audit_logs.admin_id).
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('LMF', 'Planer', 'lmf-planer@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen Pläne: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM audit_logs WHERE admin_id = $1`, adminID); err != nil {
			t.Logf("Aufräumen Audit: %v", err)
		}
	})

	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatalf("Testdatum %q: %v", d, err)
		}
		return service.TagesEndeInSchulzeitzone(x)
	}
	anna := seedSchueler(t, pool, "AUD-1", "Anna", "9H1")
	seedAusleihe(t, pool, anna, "LMF Mathe 9 Audit", tag("2027-07-31"))

	// Wie lmfPlanAufruf, aber mit angemeldetem Benutzer im Kontext — ohne ihn schreibt
	// kein Audit, und der Test wäre grün, ohne etwas zu belegen.
	ruf := func(methode, art, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, "/api/lmf-plan/"+art, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("art", art)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		switch methode {
		case http.MethodPut:
			srv.PutLmfPlanHandler()(rec, req)
		case http.MethodPost:
			srv.PostLmfPlanVeroeffentlichenHandler()(rec, req)
		default:
			srv.DeleteLmfPlanHandler()(rec, req)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: %d %s", methode, art, rec.Code, rec.Body.String())
		}
		return rec
	}
	// eintraege liefert die Protokollzeilen dieses Admins, älteste zuerst, als
	// „AKTION/fristen_angepasst".
	eintraege := func() []string {
		t.Helper()
		zeilen, err := pool.Query(ctx, `
			SELECT aktion, COALESCE(details->>'fristen_angepasst', '?'), COALESCE(details->>'art', '?')
			FROM audit_logs WHERE admin_id = $1 AND aktion LIKE 'LMF_PLAN%'
			ORDER BY zeitstempel, aktion`, adminID)
		if err != nil {
			t.Fatalf("Audit lesen: %v", err)
		}
		defer zeilen.Close()
		var alle []string
		for zeilen.Next() {
			var aktion, fristen, art string
			if err := zeilen.Scan(&aktion, &fristen, &art); err != nil {
				t.Fatalf("Audit-Zeile: %v", err)
			}
			alle = append(alle, aktion+"/"+fristen+"/"+art)
		}
		if err := zeilen.Err(); err != nil {
			t.Fatalf("Audit-Zeilen: %v", err)
		}
		return alle
	}

	// 1. Entwurf speichern: keine Frist bewegt, keine Spur.
	ruf(http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-28","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`)
	if got := eintraege(); len(got) != 0 {
		t.Errorf("ein Entwurf gehört nicht ins Protokoll: %v", got)
	}

	// 2. Veröffentlichen: Annas Schulbuch folgt dem Termin — mit Zahl im Protokoll.
	rec := ruf(http.MethodPost, "rueckgabe", "")
	var antwort LmfPlanSpeicherAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if antwort.FristenAngepasst != 1 {
		t.Fatalf("Veröffentlichen sollte genau eine Frist bewegen: %d", antwort.FristenAngepasst)
	}
	if got := eintraege(); len(got) != 1 || got[0] != "LMF_PLAN_VEROEFFENTLICHT/1/rueckgabe" {
		t.Errorf("Veröffentlichen im Protokoll: %v", got)
	}

	// 3. Korrektur eines veröffentlichten Plans: gilt sofort, gehört also ins Protokoll.
	ruf(http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-30","letzte_stunde":1,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`)
	if got := eintraege(); len(got) != 2 || got[1] != "LMF_PLAN_GESPEICHERT/1/rueckgabe" {
		t.Errorf("Korrektur im Protokoll: %v", got)
	}

	// 4. Verwerfen: die Fristen kehren zum Stichtag zurück — auch das steht drin.
	ruf(http.MethodDelete, "rueckgabe", "")
	got := eintraege()
	if len(got) != 3 || got[2] != "LMF_PLAN_VERWORFEN/1/rueckgabe" {
		t.Errorf("Verwerfen im Protokoll: %v", got)
	}
}
