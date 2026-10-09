package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Vier Türen hinterlassen einen Eintrag im Protokoll der Verwaltung: Leser in den Papierkorb,
// zurückholen, endgültig löschen, und die Mail-Einstellungen speichern. Der Eintrag nennt den
// Bearbeiter. Bei den drei Türen der Leserdatei steht der Leser unter dem Schlüssel
// schueler_id, an dem Auskunft, Tilgung und das Einspielen einer Sicherung
// (docs/resilience_and_recovery.md, Schritt 5b) ihn finden, und dazu die Adresse des
// Aufrufers; die Mail-Einstellungen speichern keine Adresse und kein Passwort.
func TestVerwaltung_LeserUndMailEinstellungenStehenImProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	adminID, rufe := protokollWelt(t, pool)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion IN
			('DELETE_STUDENT', 'RESTORE_STUDENT', 'PURGE_STUDENT', 'UPDATE_MAIL_SETTINGS')`)
	})

	// eintrag liest den jüngsten Eintrag der Aktion: Bearbeiter, Details und Adresse.
	eintrag := func(t *testing.T, aktion string) (details map[string]any, ip string) {
		t.Helper()
		var wer string
		var roh []byte
		if err := pool.QueryRow(ctx, `
			SELECT coalesce(admin_id::text, ''), details, coalesce(ip_adresse, '') FROM audit_logs
			WHERE aktion = $1 ORDER BY zeitstempel DESC, id DESC LIMIT 1`, aktion).Scan(&wer, &roh, &ip); err != nil {
			t.Fatalf("%s: kein Eintrag im Protokoll: %v", aktion, err)
		}
		if wer != adminID {
			t.Errorf("%s: Bearbeiter %q, erwartet %q", aktion, wer, adminID)
		}
		details = map[string]any{}
		if err := json.Unmarshal(roh, &details); err != nil {
			t.Fatalf("%s: Details unlesbar: %v", aktion, err)
		}
		return details, ip
	}
	erwarteStatus := func(t *testing.T, was string, ist, soll int, rumpf string) {
		t.Helper()
		if ist != soll {
			t.Fatalf("%s: Status %d, erwartet %d: %s", was, ist, soll, rumpf)
		}
	}

	t.Run("Leserdatei", func(t *testing.T) {
		id := seedSchueler(t, pool, "A-91001", "Protokoll", "07A")
		// leserEintrag prüft, dass der jüngste Eintrag der Aktion diesen Leser und eine Adresse nennt.
		leserEintrag := func(t *testing.T, aktion string) {
			t.Helper()
			details, ip := eintrag(t, aktion)
			if details["schueler_id"] != id {
				t.Errorf("%s: Eintrag nennt den Leser %v, erwartet %q", aktion, details["schueler_id"], id)
			}
			if ip == "" {
				t.Errorf("%s: Eintrag ohne Adresse des Aufrufers", aktion)
			}
		}

		rec := rufe(t, http.MethodDelete, "/api/schueler/"+id, "")
		erwarteStatus(t, "in den Papierkorb", rec.Code, http.StatusOK, rec.Body.String())
		leserEintrag(t, "DELETE_STUDENT")

		rec = rufe(t, http.MethodPost, "/api/schueler/"+id+"/restore", "")
		erwarteStatus(t, "zurückholen", rec.Code, http.StatusOK, rec.Body.String())
		leserEintrag(t, "RESTORE_STUDENT")

		rec = rufe(t, http.MethodDelete, "/api/schueler/"+id, "")
		erwarteStatus(t, "wieder in den Papierkorb", rec.Code, http.StatusOK, rec.Body.String())
		rec = rufe(t, http.MethodDelete, "/api/schueler/deleted/"+id, "")
		erwarteStatus(t, "endgültig löschen", rec.Code, http.StatusOK, rec.Body.String())
		leserEintrag(t, "PURGE_STUDENT")
	})

	t.Run("Mail-Einstellungen", func(t *testing.T) {
		// Die eine Zeile der Mail-Konfiguration steht nach dem Test wieder wie vorher. Das
		// Passwort bleibt unberührt: Ein leeres Passwortfeld lässt das gespeicherte stehen.
		var vorHost, vorPort, vorUser, vorAbsender string
		if err := pool.QueryRow(ctx, `SELECT smtp_host, smtp_port, smtp_user, sender_email
			FROM mail_settings_config WHERE id = 1`).Scan(&vorHost, &vorPort, &vorUser, &vorAbsender); err != nil {
			t.Fatalf("Mail-Konfiguration lesen: %v", err)
		}
		t.Cleanup(func() {
			aufraeumen(t, pool, `UPDATE mail_settings_config
				SET smtp_host = $1, smtp_port = $2, smtp_user = $3, sender_email = $4 WHERE id = 1`,
				vorHost, vorPort, vorUser, vorAbsender)
		})

		rec := rufe(t, http.MethodPut, "/api/admin/settings/mail",
			`{"smtp_host":"mail.protokoll.invalid","smtp_port":"587","smtp_user":"probe","smtp_password":"","sender_email":"probe@protokoll.invalid"}`)
		erwarteStatus(t, "Mail-Einstellungen speichern", rec.Code, http.StatusOK, rec.Body.String())

		details, ip := eintrag(t, "UPDATE_MAIL_SETTINGS")
		if details["smtp_host"] != "mail.protokoll.invalid" || details["password_changed"] != false {
			t.Errorf("Eintrag der Mail-Einstellungen: %v", details)
		}
		if _, steht := details["smtp_password"]; steht {
			t.Errorf("der Eintrag trägt ein Passwort-Feld: %v", details)
		}
		if ip != "" {
			t.Errorf("der Eintrag der Mail-Einstellungen speichert eine Adresse: %q", ip)
		}
	})
}
