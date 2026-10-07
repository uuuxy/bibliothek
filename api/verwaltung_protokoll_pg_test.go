package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Vier Türen der Verwaltung ändern, wohin Mahnlisten und Bestellungen gehen, was in den Mails
// steht und wann eine Klasse ihre Schulbücher abgibt. Wer es geändert hat, steht im Protokoll:
// Bearbeiter, Zeit und Gegenstand. Die Mailadresse selbst steht nicht darin, sie bliebe bis
// zur Aufbewahrungsfrist des Protokolls, auch wenn die Lehrkraft die Schule verlassen hat.
// Eine Anfrage, die nichts ändert, und eine abgelehnte schreiben keinen Eintrag.
func TestVerwaltung_AenderungStehtImProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"verwaltung-protokoll-testgeheimnis-32-b!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Vera', 'Verwaltung', 'verwaltung-protokoll@example.org', 'admin', true)
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(adminID, "VERW-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	const (
		adresseLeitung  = "frau.beispiel@schule.example"
		adresseLeitung2 = "herr.muster@schule.example"
		adresseHaendler = "bestellung@haendler.example"
	)
	aktionen := `('KLASSENLEITUNG_GEAENDERT', 'KLASSENLEITUNG_ENTFERNT', 'MAILVORLAGE_GEAENDERT',
		'LIEFERANT_ANGELEGT', 'LIEFERANT_GEAENDERT', 'LIEFERANT_GELOESCHT', 'FRIST_KLASSE_GEAENDERT')`
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion IN `+aktionen)
		aufraeumen(t, pool, `DELETE FROM klassen_lehrer_mapping WHERE klasse = 'VP7'`)
		aufraeumen(t, pool, `DELETE FROM mail_vorlagen WHERE typ = 'protokoll_probe'`)
		aufraeumen(t, pool, `DELETE FROM lieferanten WHERE name LIKE 'Buchhandlung Protokoll%'`)
	})

	rufe := func(t *testing.T, methode, pfad, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, mitCSRF(req))
		return rec
	}
	eintraege := func(t *testing.T, aktion string) int {
		t.Helper()
		return zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = $1`, aktion)
	}
	// letzter liest den jüngsten Eintrag der Aktion und prüft den Bearbeiter.
	letzter := func(t *testing.T, aktion string) map[string]any {
		t.Helper()
		var wer string
		var roh []byte
		if err := pool.QueryRow(ctx, `
			SELECT coalesce(admin_id::text, ''), details FROM audit_logs
			WHERE aktion = $1 ORDER BY zeitstempel DESC, id DESC LIMIT 1`, aktion).Scan(&wer, &roh); err != nil {
			t.Fatalf("%s: kein Eintrag im Protokoll: %v", aktion, err)
		}
		if wer != adminID {
			t.Errorf("%s: Bearbeiter %q, erwartet %q", aktion, wer, adminID)
		}
		details := map[string]any{}
		if err := json.Unmarshal(roh, &details); err != nil {
			t.Fatalf("%s: Details unlesbar: %v", aktion, err)
		}
		return details
	}
	erwarte := func(t *testing.T, rec *httptest.ResponseRecorder, status int, was string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("%s: Status %d, erwartet %d: %s", was, rec.Code, status, rec.Body.String())
		}
	}

	t.Run("Klassenleitung", func(t *testing.T) {
		erwarte(t, rufe(t, http.MethodPost, "/api/klassen-mapping",
			`{"klasse":"VP7","lehrer_email":"`+adresseLeitung+`"}`), http.StatusOK, "eintragen")
		if d := letzter(t, "KLASSENLEITUNG_GEAENDERT"); d["klasse"] != "VP7" || d["art"] != "eingetragen" {
			t.Errorf("Eintrag nach dem Eintragen: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPost, "/api/klassen-mapping",
			`{"klasse":"VP7","lehrer_email":"`+adresseLeitung+`"}`), http.StatusOK, "dieselbe Adresse")
		if n := eintraege(t, "KLASSENLEITUNG_GEAENDERT"); n != 1 {
			t.Errorf("%d Einträge, nachdem dieselbe Adresse noch einmal gespeichert wurde; erwartet 1", n)
		}

		erwarte(t, rufe(t, http.MethodPost, "/api/klassen-mapping",
			`{"klasse":"VP7","lehrer_email":"`+adresseLeitung2+`"}`), http.StatusOK, "andere Adresse")
		if d := letzter(t, "KLASSENLEITUNG_GEAENDERT"); d["klasse"] != "VP7" || d["art"] != "geaendert" {
			t.Errorf("Eintrag nach dem Ändern: %v", d)
		}
		if n := eintraege(t, "KLASSENLEITUNG_GEAENDERT"); n != 2 {
			t.Errorf("%d Einträge nach Eintragen und Ändern, erwartet 2", n)
		}

		erwarte(t, rufe(t, http.MethodDelete, "/api/klassen-mapping/VP7", ""), http.StatusNoContent, "entfernen")
		if d := letzter(t, "KLASSENLEITUNG_ENTFERNT"); d["klasse"] != "VP7" {
			t.Errorf("Eintrag nach dem Entfernen: %v", d)
		}
		erwarte(t, rufe(t, http.MethodDelete, "/api/klassen-mapping/VP7", ""), http.StatusNotFound, "noch einmal entfernen")
		if n := eintraege(t, "KLASSENLEITUNG_ENTFERNT"); n != 1 {
			t.Errorf("%d Einträge zum Entfernen, erwartet 1", n)
		}
	})

	t.Run("Mail-Vorlage", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO mail_vorlagen (typ, betreff, text_body)
			VALUES ('protokoll_probe', 'Alter Betreff', 'Alter Text') RETURNING id`).Scan(&id); err != nil {
			t.Fatalf("Vorlage anlegen: %v", err)
		}
		erwarte(t, rufe(t, http.MethodPut, "/api/mail-templates/"+id,
			`{"betreff":"Neuer Betreff","text_body":"Alter Text"}`), http.StatusOK, "Betreff ändern")
		d := letzter(t, "MAILVORLAGE_GEAENDERT")
		if d["vorlage"] != "protokoll_probe" || fmtFelder(d["felder"]) != "betreff" {
			t.Errorf("Eintrag nach dem Ändern des Betreffs: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/mail-templates/"+id,
			`{"betreff":"Neuer Betreff","text_body":"Alter Text"}`), http.StatusOK, "unverändert speichern")
		if n := eintraege(t, "MAILVORLAGE_GEAENDERT"); n != 1 {
			t.Errorf("%d Einträge nach einem Speichern ohne Änderung, erwartet 1", n)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/mail-templates/"+id,
			`{"betreff":"Dritter Betreff","text_body":"Neuer Text"}`), http.StatusOK, "beides ändern")
		if d := letzter(t, "MAILVORLAGE_GEAENDERT"); fmtFelder(d["felder"]) != "betreff,text" {
			t.Errorf("Eintrag nach dem Ändern von Betreff und Text: %v", d)
		}

		// Eine Vorlage, die es nicht gibt, ist kein „Erfolgreich gespeichert".
		erwarte(t, rufe(t, http.MethodPut, "/api/mail-templates/00000000-0000-4000-8000-000000000000",
			`{"betreff":"x","text_body":"y"}`), http.StatusNotFound, "unbekannte Vorlage")
		if n := eintraege(t, "MAILVORLAGE_GEAENDERT"); n != 2 {
			t.Errorf("%d Einträge nach einer unbekannten Vorlage, erwartet 2", n)
		}
	})

	t.Run("Lieferant", func(t *testing.T) {
		rec := rufe(t, http.MethodPost, "/api/lieferanten",
			`{"name":"Buchhandlung Protokoll","email":"`+adresseHaendler+`","customerNumber":"K-1"}`)
		erwarte(t, rec, http.StatusCreated, "anlegen")
		var angelegt struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &angelegt); err != nil || angelegt.ID == "" {
			t.Fatalf("Antwort der Anlage ohne Kennung: %s", rec.Body.String())
		}
		if d := letzter(t, "LIEFERANT_ANGELEGT"); d["lieferant_id"] != angelegt.ID || d["name"] != "Buchhandlung Protokoll" {
			t.Errorf("Eintrag nach der Anlage: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/lieferanten/"+angelegt.ID,
			`{"name":"Buchhandlung Protokoll","email":"neu-`+adresseHaendler+`","customerNumber":"K-1"}`),
			http.StatusOK, "Adresse ändern")
		d := letzter(t, "LIEFERANT_GEAENDERT")
		if d["lieferant_id"] != angelegt.ID || d["name"] != "Buchhandlung Protokoll" || fmtFelder(d["felder"]) != "email" {
			t.Errorf("Eintrag nach dem Ändern der Adresse: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/lieferanten/"+angelegt.ID,
			`{"name":"Buchhandlung Protokoll","email":"neu-`+adresseHaendler+`","customerNumber":"K-1"}`),
			http.StatusOK, "unverändert speichern")
		if n := eintraege(t, "LIEFERANT_GEAENDERT"); n != 1 {
			t.Errorf("%d Einträge nach einem Speichern ohne Änderung, erwartet 1", n)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/lieferanten/"+angelegt.ID,
			`{"name":"Buchhandlung Protokoll 2","email":"neu-`+adresseHaendler+`","customerNumber":"K-2","ist_hauptlieferant":true}`),
			http.StatusOK, "Name, Nummer und Hauptlieferant ändern")
		if d := letzter(t, "LIEFERANT_GEAENDERT"); fmtFelder(d["felder"]) != "name,kundennummer,hauptlieferant" || d["name"] != "Buchhandlung Protokoll 2" {
			t.Errorf("Eintrag nach dem Ändern von Name, Nummer und Hauptlieferant: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPut, "/api/lieferanten/00000000-0000-4000-8000-000000000000",
			`{"name":"Niemand","email":"niemand@haendler.example","customerNumber":"K-0"}`),
			http.StatusNotFound, "unbekannten Lieferanten ändern")
		if n := eintraege(t, "LIEFERANT_GEAENDERT"); n != 2 {
			t.Errorf("%d Einträge nach einem unbekannten Lieferanten, erwartet 2", n)
		}

		erwarte(t, rufe(t, http.MethodDelete, "/api/lieferanten/"+angelegt.ID, ""), http.StatusConflict, "Hauptlieferanten löschen")
		if n := eintraege(t, "LIEFERANT_GELOESCHT"); n != 0 {
			t.Errorf("%d Einträge zu einem abgelehnten Löschen", n)
		}
		erwarte(t, rufe(t, http.MethodPut, "/api/lieferanten/"+angelegt.ID,
			`{"name":"Buchhandlung Protokoll 2","email":"neu-`+adresseHaendler+`","customerNumber":"K-2"}`),
			http.StatusOK, "Hauptlieferant abwählen")
		erwarte(t, rufe(t, http.MethodDelete, "/api/lieferanten/"+angelegt.ID, ""), http.StatusNoContent, "löschen")
		if d := letzter(t, "LIEFERANT_GELOESCHT"); d["lieferant_id"] != angelegt.ID || d["name"] != "Buchhandlung Protokoll 2" {
			t.Errorf("Eintrag nach dem Löschen: %v", d)
		}
	})

	t.Run("Lernmittel einer Klasse verlängern", func(t *testing.T) {
		kind := seedSchueler(t, pool, "S-VPR-1", "Protokoll", "5a")
		seedAusleihe(t, pool, kind, "LMF Protokoll 5", fristEnde(2027, time.March, 31))

		rec := rufe(t, http.MethodPost, "/api/ausleihen/global-extend-lmf",
			`{"klasse":"5a","neues_rueckgabe_datum":"2027-06-30"}`)
		erwarte(t, rec, http.StatusOK, "verlängern")
		d := letzter(t, "FRIST_KLASSE_GEAENDERT")
		frist, istText := d["neue_frist"].(string)
		if !istText || d["klasse"] != "5a" || d["fristen_angepasst"] != float64(1) || !strings.HasPrefix(frist, "2027-06-30") {
			t.Errorf("Eintrag nach dem Verlängern: %v", d)
		}

		erwarte(t, rufe(t, http.MethodPost, "/api/ausleihen/global-extend-lmf",
			`{"klasse":"VP9","neues_rueckgabe_datum":"2027-06-30"}`), http.StatusOK, "Klasse ohne Ausleihe")
		if n := eintraege(t, "FRIST_KLASSE_GEAENDERT"); n != 1 {
			t.Errorf("%d Einträge, nachdem für eine Klasse ohne Ausleihe nichts geändert wurde; erwartet 1", n)
		}
	})

	// Keine der Adressen steht in einem der Einträge.
	for _, adresse := range []string{adresseLeitung, adresseLeitung2, adresseHaendler} {
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs
			WHERE aktion IN `+aktionen+` AND position($1 in details::text) > 0`, adresse); n != 0 {
			t.Errorf("Die Adresse %s steht in %d Einträgen des Protokolls", adresse, n)
		}
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion IN `+aktionen); n != 11 {
		t.Errorf("%d Einträge insgesamt, erwartet 11", n)
	}
}

// fmtFelder macht aus der Liste der geänderten Felder eines Eintrags eine Zeile.
func fmtFelder(wert any) string {
	liste, istListe := wert.([]any)
	if !istListe {
		return ""
	}
	namen := make([]string, 0, len(liste))
	for _, f := range liste {
		if s, ok := f.(string); ok {
			namen = append(namen, s)
		}
	}
	return strings.Join(namen, ",")
}
