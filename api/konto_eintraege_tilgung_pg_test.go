package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Name und Adresse eines früheren Zugangskontos fallen mit dem Leser, auch aus den Einträgen
// über das Konto.
//
// Anlage und Änderung eines Kontos stehen im Admin-Protokoll unter der Kennung des Kontos
// (ziel_id), mit Adresse und Name. Wird das Konto gelöscht und bleibt die Leserzeile, nennt
// der Löscheintrag den Leser; daran findet die Auskunft das frühere Konto. Nimmt die Tilgung
// Name und Adresse nur aus dem Löscheintrag, führt von der Kennung des getilgten Lesers über
// die Kennung des Kontos weiter ein Weg zu seinem Namen. Geprüft über die Türen der
// Benutzerverwaltung und der Leserdatei, mit endgültigem Löschen und mit Anonymisierung.
func TestKontoEintraege_FallenMitDemLeser(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	admin := adminFuerAudit(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM benutzer WHERE email LIKE '%@konto-eintraege.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	// kontoMitEintraegen legt über die Benutzerverwaltung ein Konto an und ändert es:
	// USER_CREATE mit Adresse und Name, USER_UPDATE mit Adresse.
	kontoMitEintraegen := func(t *testing.T, vorname, nachname, ausweis string) (kontoID, leserID, adresse string) {
		t.Helper()
		adresse = vorname + "@konto-eintraege.invalid"
		if rec := fkAlsAdmin(t, admin, srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer",
			`{"barcode_id":"`+ausweis+`","vorname":"`+vorname+`","nachname":"`+nachname+`","email":"`+adresse+`","rolle":"mitarbeiter"}`,
			nil); rec.Code != http.StatusOK {
			t.Fatalf("Konto anlegen: Status %d — %s", rec.Code, rec.Body.String())
		}
		if err := pool.QueryRow(ctx, `SELECT id::text, leser_id::text FROM benutzer WHERE email = $1`, adresse).
			Scan(&kontoID, &leserID); err != nil {
			t.Fatalf("Konto lesen: %v", err)
		}
		if rec := fkAlsAdmin(t, admin, srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+kontoID,
			`{"rolle":"helfer"}`, map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
			t.Fatalf("Konto ändern: Status %d — %s", rec.Code, rec.Body.String())
		}
		return kontoID, leserID, adresse
	}
	loescheKonto := func(t *testing.T, kontoID string) {
		t.Helper()
		if rec := fkAlsAdmin(t, admin, srv.DeleteUserHandler(auditRepo, userRepo), http.MethodDelete, "/api/benutzer/"+kontoID, "",
			map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
			t.Fatalf("Konto löschen: Status %d — %s", rec.Code, rec.Body.String())
		}
	}
	// imProtokoll zählt die Einträge beider Protokolle, die den Text irgendwo tragen.
	imProtokoll := func(t *testing.T, text string) int {
		t.Helper()
		return zaehleZeilen(t, pool, `
			SELECT (SELECT count(*) FROM audit_logs WHERE details::text LIKE '%' || $1 || '%')
			     + (SELECT count(*) FROM audit_log WHERE details::text LIKE '%' || $1 || '%')`, text)
	}
	// Positivkontrolle: Ohne Adresse und Name im Protokoll wäre die Prüfung danach grün, ohne
	// etwas zu prüfen. Anlage, Änderung und Löscheintrag des Kontos nennen die Adresse; den
	// Namen nennen Anlage und Löscheintrag, über die Leserdatei auch der Eintrag der Leserzeile.
	stehtImProtokoll := func(t *testing.T, adresse, nachname string, mitNamen int) {
		t.Helper()
		if a, n := imProtokoll(t, adresse), imProtokoll(t, nachname); a != 3 || n != mitNamen {
			t.Fatalf("vor der Tilgung steht die Adresse in %d Einträgen und der Name in %d, erwartet 3 und %d", a, n, mitNamen)
		}
	}
	// Die Einträge selbst bleiben: Aktion, Zeit, Bearbeiter, Kennung des Kontos, Rolle.
	eintraegeBleiben := func(t *testing.T, kontoID string) {
		t.Helper()
		if n := zaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_logs
			WHERE aktion IN ('USER_CREATE', 'USER_UPDATE') AND details->>'ziel_id' = $1 AND details ? 'rolle'`, kontoID); n != 2 {
			t.Errorf("%d Einträge über das Konto mit Kennung und Rolle, erwartet 2 (Anlage und Änderung)", n)
		}
	}
	loescheEndgueltig := func(t *testing.T, leserID string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, leserID); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		if err := auditRepo.PurgeStudent(ctx, leserID, admin); err != nil {
			t.Fatalf("endgültig löschen: %v", err)
		}
	}

	// Ein Konto, das besteht: Seine Einträge bleiben, was auch immer daneben getilgt wird.
	bestehend, _, adresseBestehend := kontoMitEintraegen(t, "berta", "Bleibtkonto", "KE-AUSWEIS-0")

	t.Run("Konto in der Benutzerverwaltung gelöscht, Leser endgültig gelöscht", func(t *testing.T) {
		kontoID, leserID, adresse := kontoMitEintraegen(t, "paula", "Purgekonto", "KE-AUSWEIS-1")
		loescheKonto(t, kontoID)
		stehtImProtokoll(t, adresse, "Purgekonto", 2)

		loescheEndgueltig(t, leserID)

		if a, n := imProtokoll(t, adresse), imProtokoll(t, "Purgekonto"); a != 0 || n != 0 {
			t.Errorf("nach dem endgültigen Löschen steht die Adresse in %d Einträgen und der Name in %d", a, n)
		}
		eintraegeBleiben(t, kontoID)
	})

	t.Run("Konto mit der Leserzeile in den Papierkorb gelegt, Leser endgültig gelöscht", func(t *testing.T) {
		kontoID, leserID, adresse := kontoMitEintraegen(t, "karl", "Korbkonto", "KE-AUSWEIS-2")
		req := httptest.NewRequest(http.MethodDelete, "/api/schueler/"+leserID, nil)
		req.SetPathValue("id", leserID)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.DeleteStudentHandler(auditRepo)(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Leserzeile in den Papierkorb: Status %d — %s", rec.Code, rec.Body.String())
		}
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE id = $1`, kontoID); n != 0 {
			t.Fatalf("das Konto steht noch, obwohl die Leserzeile im Papierkorb liegt")
		}
		stehtImProtokoll(t, adresse, "Korbkonto", 3)

		loescheEndgueltig(t, leserID)

		if a, n := imProtokoll(t, adresse), imProtokoll(t, "Korbkonto"); a != 0 || n != 0 {
			t.Errorf("nach dem endgültigen Löschen steht die Adresse in %d Einträgen und der Name in %d", a, n)
		}
		eintraegeBleiben(t, kontoID)
	})

	// Ein Schüler mit Konto (Helfer): Die Anonymisierung beim Abgang fährt dieselbe Liste wie
	// der Nachtlauf. Das Konto hängt an der Leserzeile des Schülers; die Benutzerverwaltung
	// ändert und löscht es.
	t.Run("Konto gelöscht, Leser anonymisiert", func(t *testing.T) {
		leserID := seedSchueler(t, pool, "S-KE-HELFER", "Hanna", "09A")
		const adresse = "hanna@konto-eintraege.invalid"
		var kontoID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
			VALUES ('Hanna', 'Helferkonto', $1, 'helfer', true, $2) RETURNING id::text`, adresse, leserID).Scan(&kontoID); err != nil {
			t.Fatalf("Konto anlegen: %v", err)
		}
		if rec := fkAlsAdmin(t, admin, srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+kontoID,
			`{"aktiv":false}`, map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
			t.Fatalf("Konto ändern: Status %d — %s", rec.Code, rec.Body.String())
		}
		loescheKonto(t, kontoID)
		// Die Änderung und der Löscheintrag nennen die Adresse.
		if a := imProtokoll(t, adresse); a != 2 {
			t.Fatalf("vor der Anonymisierung steht die Adresse in %d Einträgen, erwartet 2", a)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer db.SafeRollback(ctx, tx)
		if err := anonymisiereAbgaenger(ctx, tx, leserID); err != nil {
			t.Fatalf("anonymisieren: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}

		if a := imProtokoll(t, adresse); a != 0 {
			t.Errorf("nach der Anonymisierung steht die Adresse in %d Einträgen", a)
		}
		if n := zaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_logs WHERE aktion = 'USER_UPDATE' AND details->>'ziel_id' = $1`, kontoID); n != 1 {
			t.Errorf("%d Einträge USER_UPDATE über das Konto, erwartet 1", n)
		}
	})

	// Das Konto gehört der Anlage: Solange es besteht, bleiben seine Einträge mit Adresse und
	// Name (Anlage und Änderung).
	if a, n := imProtokoll(t, adresseBestehend), imProtokoll(t, "Bleibtkonto"); a != 2 || n != 1 {
		t.Errorf("die Einträge des bestehenden Kontos nennen die Adresse %d-mal und den Namen %d-mal, erwartet 2 und 1", a, n)
	}
	eintraegeBleiben(t, bestehend)
}
