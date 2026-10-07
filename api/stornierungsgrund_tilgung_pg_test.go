package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
	"bibliothek/sse"
)

// Der getippte Grund einer Stornierung fällt mit der Person.
//
// Wer in der Leserakte eine Gebühr storniert, muss einen Grund tippen. Er steht an der
// Forderung (stornierungsgrund) und im Eintrag STORNIERUNG der Datensatz-Historie (grund) und
// kann die Person oder ihre Familie nennen. Der Eintrag trägt die Kennung des Lesers, damit
// die Tilgung ihn findet, auch wenn die Forderung schon gelöscht ist. Geprüft über die Tür
// der Stornierung und vier Wege: Anonymisierung, endgültiges Löschen, Forderung vorher mit
// ihrem Titel gelöscht, Leserzeile vorher zusammengeführt.
func TestStornierungsgrund_FaelltMitDerPerson(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"storno-tilgung-testgeheimnis-32-bytes!!!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Stella', 'Storno', 'storno-tilgung@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "STORNO-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	// storniere legt dem Leser eine offene Forderung an und storniert sie über die Tür.
	storniere := func(t *testing.T, leserID, kennung, grund string) (forderungID, titelID string) {
		t.Helper()
		titelID = titelMitMeldebestand(t, pool, "Stornotitel "+kennung, 1)
		exemplarID := exemplar(t, pool, titelID, "EX-STORNO-"+kennung, true, "")
		if err := pool.QueryRow(ctx,
			`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag)
			 VALUES ($1, $2, 'Wasserschaden', 12.50) RETURNING id`, exemplarID, leserID).Scan(&forderungID); err != nil {
			t.Fatalf("Forderung anlegen: %v", err)
		}
		req := jsonPost("/api/schadensfaelle/"+forderungID+"/storno", `{"grund":"`+grund+`"}`)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Stornieren: Status %d — %s", rec.Code, rec.Body.String())
		}
		return forderungID, titelID
	}
	anForderung := func(t *testing.T, grund string) int {
		t.Helper()
		return zaehleZeilen(t, pool, `SELECT count(*) FROM schadensfaelle WHERE stornierungsgrund = $1`, grund)
	}
	imProtokoll := func(t *testing.T, grund string) int {
		t.Helper()
		return zaehleZeilen(t, pool, `SELECT count(*) FROM audit_log WHERE details::text LIKE '%' || $1 || '%'`, grund)
	}
	// Positivkontrolle: Ohne den Grund an beiden Stellen wäre die Prüfung danach grün, ohne
	// etwas zu prüfen.
	stehtAnBeidenStellen := func(t *testing.T, grund string) {
		t.Helper()
		if a, p := anForderung(t, grund), imProtokoll(t, grund); a != 1 || p != 1 {
			t.Fatalf("nach dem Stornieren steht der Grund an %d Forderungen und in %d Einträgen, erwartet je 1", a, p)
		}
	}
	anonymisiere := func(t *testing.T, leserID string) {
		t.Helper()
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
	}

	t.Run("Anonymisierung", func(t *testing.T) {
		const grund = "STORNO-PROBE-A Mutter von Max Muster hat angerufen"
		leser := seedSchueler(t, pool, "S-STORNO-A", "Anna", "07A")
		forderung, _ := storniere(t, leser, "A", grund)
		stehtAnBeidenStellen(t, grund)

		anonymisiere(t, leser)

		if n := anForderung(t, grund); n != 0 {
			t.Errorf("der getippte Grund steht nach der Anonymisierung noch an der Forderung")
		}
		if n := imProtokoll(t, grund); n != 0 {
			t.Errorf("der getippte Grund steht nach der Anonymisierung noch in %d Einträgen der Datensatz-Historie", n)
		}
		// Forderung und Eintrag bleiben als Beleg: storniert, wann, welcher Betrag.
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM schadensfaelle WHERE id = $1 AND storniert_am IS NOT NULL AND betrag = 12.50`, forderung); n != 1 {
			t.Errorf("die stornierte Forderung steht nach der Anonymisierung nicht mehr als Beleg da")
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_log WHERE tabelle = 'schadensfaelle' AND aktion = 'STORNIERUNG'
			   AND datensatz_id = $1::uuid AND details->>'betrag' = '12.5' AND details->>'schueler_id' = $2`,
			forderung, leser); n != 1 {
			t.Errorf("der Eintrag STORNIERUNG steht nach der Anonymisierung nicht mehr mit Betrag und Kennung da")
		}
	})

	t.Run("endgültiges Löschen", func(t *testing.T) {
		const grund = "STORNO-PROBE-B Vater von Ben Beispiel war am Telefon"
		leser := seedSchueler(t, pool, "S-STORNO-B", "Ben", "07B")
		storniere(t, leser, "B", grund)
		stehtAnBeidenStellen(t, grund)

		if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = NOW() WHERE id = $1`, leser); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		if err := repository.NewAuditRepository(pool).PurgeStudent(ctx, leser, kontoID); err != nil {
			t.Fatalf("endgültig löschen: %v", err)
		}

		if n := imProtokoll(t, grund); n != 0 {
			t.Errorf("der getippte Grund steht nach dem endgültigen Löschen noch in %d Einträgen der Datensatz-Historie", n)
		}
	})

	t.Run("Forderung vorher mit ihrem Titel gelöscht", func(t *testing.T) {
		const grund = "STORNO-PROBE-C Oma von Cem Probe bringt das Buch"
		leser := seedSchueler(t, pool, "S-STORNO-C", "Cem", "07C")
		_, titelID := storniere(t, leser, "C", grund)
		stehtAnBeidenStellen(t, grund)

		if err := repository.NewAuditRepository(pool).DeleteTitle(ctx, titelID, kontoID); err != nil {
			t.Fatalf("Titel löschen: %v", err)
		}
		if n := imProtokoll(t, grund); n != 1 {
			t.Fatalf("nach dem Löschen des Titels steht der Grund in %d Einträgen, erwartet 1", n)
		}

		anonymisiere(t, leser)

		if n := imProtokoll(t, grund); n != 0 {
			t.Errorf("der getippte Grund einer Forderung, die mit ihrem Titel gelöscht wurde, überlebt die Anonymisierung")
		}
	})

	t.Run("Leserzeile vorher zusammengeführt", func(t *testing.T) {
		const grund = "STORNO-PROBE-D Schwester von Dana Doppelt hat bezahlt"
		ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Dana", nachname: "Doppelt", klasse: "07A", barcode: "S-STORNO-D1", geb: datum(2012, 6, 6)})
		quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Dana", nachname: "Doppelt-Alt", klasse: "07A", barcode: "S-STORNO-D2", geb: datum(2012, 6, 6)})
		storniere(t, quelle, "D", grund)
		stehtAnBeidenStellen(t, grund)

		if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
			t.Fatalf("Zusammenführen: %v", err)
		}
		anonymisiere(t, ziel)

		if a, p := anForderung(t, grund), imProtokoll(t, grund); a != 0 || p != 0 {
			t.Errorf("nach Zusammenführen und Anonymisierung steht der Grund an %d Forderungen und in %d Einträgen", a, p)
		}
	})
}
