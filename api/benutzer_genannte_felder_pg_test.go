package api

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Tür „Benutzer ändern" schreibt nur, was der Rumpf nennt.
//
// Die Maske füllt sich aus der Zeile der Liste, die beim Öffnen der Seite lädt. Schickte sie
// beim Speichern jedes Feld zurück, schrieb sie den alten Stand über alles, was ein anderer
// Platz inzwischen geändert hat: die Rolle, „aktiv", die Ausweisnummer aus der Leserakte.
// Geprüft über die Handler der Benutzerverwaltung am echten Postgres.
func TestBenutzerAendern_SchreibtNurDieGenanntenFelder(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	admin := adminFuerAudit(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM benutzer WHERE email LIKE '%@genannte-felder.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	if rec := fkAlsAdmin(t, admin, srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer",
		`{"barcode_id":"GF-AUSWEIS-1","vorname":"Gerda","nachname":"Genannt","email":"gerda@genannte-felder.invalid","rolle":"mitarbeiter"}`,
		nil); rec.Code != http.StatusOK {
		t.Fatalf("Konto anlegen: Status %d — %s", rec.Code, rec.Body.String())
	}
	var kontoID, leserID string
	if err := pool.QueryRow(ctx, `SELECT id::text, leser_id::text FROM benutzer WHERE email = 'gerda@genannte-felder.invalid'`).
		Scan(&kontoID, &leserID); err != nil {
		t.Fatalf("Konto lesen: %v", err)
	}
	aendere := func(t *testing.T, rumpf string) int {
		t.Helper()
		rec := fkAlsAdmin(t, admin, srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+kontoID, rumpf,
			map[string]string{"id": kontoID})
		if rec.Code != http.StatusOK {
			t.Logf("Antwort auf %s: %s", rumpf, rec.Body.String())
		}
		return rec.Code
	}
	type stand struct {
		vorname, nachname, email, rolle string
		aktiv, beantragt                bool
		leserVorname, ausweis           string
	}
	lies := func(t *testing.T) stand {
		t.Helper()
		var s stand
		if err := pool.QueryRow(ctx, `
			SELECT b.vorname, b.nachname, b.email, b.rolle::text, b.aktiv, b.zugang_beantragt_am IS NOT NULL,
			       l.vorname, coalesce(l.barcode_id, '')
			FROM benutzer b JOIN leser l ON l.id = b.leser_id WHERE b.id = $1`, kontoID).
			Scan(&s.vorname, &s.nachname, &s.email, &s.rolle, &s.aktiv, &s.beantragt, &s.leserVorname, &s.ausweis); err != nil {
			t.Fatalf("Stand lesen: %v", err)
		}
		return s
	}
	eintraege := func(t *testing.T) int {
		t.Helper()
		return zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'USER_UPDATE' AND details->>'ziel_id' = $1`, kontoID)
	}

	// Ein anderer Platz stuft das Konto herab und schaltet es ab; die Leserakte trägt eine
	// neue Ausweisnummer ein.
	if code := aendere(t, `{"rolle":"helfer","aktiv":false}`); code != http.StatusOK {
		t.Fatalf("Rolle und „aktiv“ allein ändern: Status %d, erwartet 200", code)
	}
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = 'GF-AUSWEIS-2' WHERE id = $1`, leserID); err != nil {
		t.Fatalf("Ausweisnummer in der Leserakte eintragen: %v", err)
	}

	// Die Maske vom Morgen speichert nur den Vornamen.
	if code := aendere(t, `{"vorname":"Gerdi"}`); code != http.StatusOK {
		t.Fatalf("den Vornamen allein ändern: Status %d, erwartet 200", code)
	}
	s := lies(t)
	if s.vorname != "Gerdi" || s.leserVorname != "Gerdi" {
		t.Errorf("der Vorname steht am Konto als %q und an der Leserzeile als %q, erwartet beide „Gerdi“", s.vorname, s.leserVorname)
	}
	if s.rolle != "helfer" || s.aktiv {
		t.Errorf("das Speichern des Vornamens hat Rolle und „aktiv“ zurückgeschrieben: rolle=%q aktiv=%v", s.rolle, s.aktiv)
	}
	if s.ausweis != "GF-AUSWEIS-2" {
		t.Errorf("das Speichern des Vornamens hat die Ausweisnummer der Leserakte überschrieben: %q", s.ausweis)
	}
	if s.nachname != "Genannt" || s.email != "gerda@genannte-felder.invalid" {
		t.Errorf("nicht genannte Felder haben sich geändert: nachname=%q email=%q", s.nachname, s.email)
	}
	// Der Eintrag nennt den Stand nach der Änderung, auch für Felder, die der Rumpf nicht nannte.
	if n := zaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_logs
		WHERE id = (SELECT id FROM audit_logs WHERE aktion = 'USER_UPDATE' AND details->>'ziel_id' = $1
		            ORDER BY zeitstempel DESC LIMIT 1)
		  AND details->>'rolle' = 'helfer' AND details->>'aktiv' = 'false'
		  AND details->>'email' = 'gerda@genannte-felder.invalid'`, kontoID); n != 1 {
		t.Errorf("der jüngste Eintrag USER_UPDATE nennt nicht den Stand nach der Änderung (helfer, nicht aktiv, Adresse)")
	}

	t.Run("ein Rumpf ohne Feld ändert nichts und schreibt keinen Eintrag", func(t *testing.T) {
		vorher, eintraegeVorher := lies(t), eintraege(t)
		if code := aendere(t, `{}`); code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200", code)
		}
		if nachher := lies(t); nachher != vorher {
			t.Errorf("der Stand hat sich geändert: %+v, vorher %+v", nachher, vorher)
		}
		if n := eintraege(t); n != eintraegeVorher {
			t.Errorf("%d Einträge USER_UPDATE, vorher %d", n, eintraegeVorher)
		}
	})

	t.Run("ein genanntes Pflichtfeld darf nicht leer sein", func(t *testing.T) {
		vorher := lies(t)
		for _, rumpf := range []string{`{"vorname":""}`, `{"nachname":" "}`, `{"email":""}`, `{"email":"keine-adresse"}`, `{"rolle":""}`} {
			if code := aendere(t, rumpf); code != http.StatusBadRequest {
				t.Errorf("%s: Status %d, erwartet 400", rumpf, code)
			}
		}
		if nachher := lies(t); nachher != vorher {
			t.Errorf("eine abgelehnte Anfrage hat geschrieben: %+v, vorher %+v", nachher, vorher)
		}
	})

	t.Run("der Antrag auf Zugang fällt mit dem Freischalten, nicht mit einem anderen Feld", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE benutzer SET zugang_beantragt_am = NOW() WHERE id = $1`, kontoID); err != nil {
			t.Fatal(err)
		}
		if code := aendere(t, `{"nachname":"Genannter"}`); code != http.StatusOK {
			t.Fatalf("Nachnamen ändern: Status %d", code)
		}
		if s := lies(t); !s.beantragt || s.nachname != "Genannter" {
			t.Errorf("nach dem Ändern des Nachnamens: beantragt=%v nachname=%q, erwartet true und „Genannter“", s.beantragt, s.nachname)
		}
		if code := aendere(t, `{"aktiv":true}`); code != http.StatusOK {
			t.Fatalf("freischalten: Status %d", code)
		}
		if s := lies(t); s.beantragt || !s.aktiv {
			t.Errorf("nach dem Freischalten: beantragt=%v aktiv=%v, erwartet false und true", s.beantragt, s.aktiv)
		}
	})

	// Die Maske schickt beim Freischalten nur „aktiv". Die Ausweisnummer zieht die Datenbank
	// (trg_aktives_konto_hat_ausweis), ohne dass die Tür die Leserzeile anfasst.
	t.Run("ein freigeschaltetes Konto bekommt seine Ausweisnummer", func(t *testing.T) {
		var antrag, antragLeser string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
			VALUES ('Anton', 'Antrag', 'anton@genannte-felder.invalid', 'kollegium', false, NOW())
			RETURNING id::text, leser_id::text`).Scan(&antrag, &antragLeser); err != nil {
			t.Fatalf("Zugangsanfrage anlegen: %v", err)
		}
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM leser WHERE id = $1 AND barcode_id IS NULL`, antragLeser); n != 1 {
			t.Fatalf("die Leserzeile der Anfrage trägt schon eine Ausweisnummer")
		}
		rec := fkAlsAdmin(t, admin, srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+antrag,
			`{"aktiv":true}`, map[string]string{"id": antrag})
		if rec.Code != http.StatusOK {
			t.Fatalf("freischalten: Status %d — %s", rec.Code, rec.Body.String())
		}
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM leser WHERE id = $1 AND barcode_id LIKE 'A-%'`, antragLeser); n != 1 {
			t.Errorf("das freigeschaltete Konto hat keine Ausweisnummer bekommen")
		}
	})

	t.Run("die Ausweisnummer ändert sich nur, wenn der Rumpf sie nennt", func(t *testing.T) {
		if code := aendere(t, `{"barcode_id":"GF-AUSWEIS-3"}`); code != http.StatusOK {
			t.Fatalf("Ausweisnummer ändern: Status %d", code)
		}
		if s := lies(t); s.ausweis != "GF-AUSWEIS-3" || s.vorname != "Gerdi" {
			t.Errorf("ausweis=%q vorname=%q, erwartet GF-AUSWEIS-3 und „Gerdi“", s.ausweis, s.vorname)
		}
	})
}
