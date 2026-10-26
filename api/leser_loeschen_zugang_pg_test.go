package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// Wer in der Leserdatei einen Kollegen löscht, hält die Regeln der Benutzerverwaltung.
//
// Mit der Leserzeile eines Kollegen fällt sein Konto (repository.LegeLeserInPapierkorb). Die Route
// verlangt delete_students, das ab Werk auch Mitarbeiter und Leitung haben. Ein Konto löscht die
// Anwendung sonst nur mit manage_users, das eines Administrators nur ein Administrator
// (DELETE /api/benutzer/{id}, api/benutzer/admin_schutz.go). Jedes Konto hat eine Leserzeile, auch
// das des Administrators: Über die Leserdatei konnte ein Mitarbeiter es löschen, und einen neuen
// Administrator legt danach niemand mehr im Programm an.
//
// Gegenproben: Ohne Zugang genügt delete_students weiter (Praktikum, Schüler, das Konto einer aus
// Littera übernommenen Lehrkraft unter der Ersatzadresse), und ein Administrator löscht weiter
// alles in einem Schritt. Die Akte sagt der Oberfläche, ob ein Zugang da ist (hat_zugang).
func TestLeserLoeschen_ZugangVerlangtKontenrecht(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	t.Setenv("RATE_LIMIT", "100000")
	ctx := context.Background()

	_, adminSitzung, anmeldung := tuertest.AdminSitzung(t, pool, "ada@loesch-zugang.example", "Ada", "Admin")
	// Routes legt das Verzeichnis uploads an.
	t.Chdir(t.TempDir())
	srv := NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), ServerKonfig{})
	router := srv.Routes()

	// Der Mitarbeiter hat nur das Recht der Route, die Leitung dazu manage_users und ist trotzdem
	// kein Administrator. Danach stehen beide Rollen wieder auf der Vorgabe.
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM role_permissions WHERE role IN ('MITARBEITER', 'LEITUNG')`); err != nil {
			t.Errorf("Rechte zurücksetzen: %v", err)
			return
		}
		if err := (&db.Verbindung{Pool: pool}).SchreibeRechteVorgabe(context.Background()); err != nil {
			t.Errorf("SchreibeRechteVorgabe() = %v", err)
		}
	})
	tuertest.SetzeGenauEinRecht(t, pool, &srv.Rechte, "delete_students")
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = 'LEITUNG'`); err != nil {
		t.Fatalf("Rechte der Leitung leeren: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO role_permissions (role, permission, allowed)
		VALUES ('LEITUNG', 'delete_students', true), ('LEITUNG', 'manage_users', true)`); err != nil {
		t.Fatalf("Rechte der Leitung setzen: %v", err)
	}
	srv.Rechte.Leere()

	sitzungFuer := func(t *testing.T, email string, rolle auth.Rolle) string {
		t.Helper()
		kontoID := testdaten.LegeKontoAn(t, pool, "Bedien", "Person", email, strings.ToLower(string(rolle)), true)
		sitzung, err := anmeldung.ErzeugeToken(kontoID, rolle, "")
		if err != nil {
			t.Fatalf("ErzeugeToken(%s) = %v", email, err)
		}
		return sitzung
	}
	alsAdmin := tuertest.RouterAufruf(router, adminSitzung)
	alsMitarbeiter := tuertest.RouterAufruf(router, sitzungFuer(t, "mia@loesch-zugang.example", auth.RolleMitarbeiter))
	alsLeitung := tuertest.RouterAufruf(router, sitzungFuer(t, "lea@loesch-zugang.example", auth.RolleLeitung))

	// Ein Konto bekommt seine Leserzeile vom Trigger trg_benutzer_hat_leserzeile.
	leserMitKonto := func(t *testing.T, email, rolle string, aktiv bool) string {
		t.Helper()
		kontoID := testdaten.LegeKontoAn(t, pool, "Kora", "Kollegin", email, rolle, aktiv)
		var leserID string
		if err := pool.QueryRow(ctx, `SELECT leser_id::text FROM benutzer WHERE id = $1`, kontoID).Scan(&leserID); err != nil {
			t.Fatalf("Leserzeile zu %s lesen: %v", email, err)
		}
		return leserID
	}
	zustand := func(t *testing.T, leserID string) (imPapierkorb bool, konten int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT (SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1),
			       (SELECT count(*) FROM benutzer WHERE leser_id = $1)`, leserID).Scan(&imPapierkorb, &konten); err != nil {
			t.Fatalf("Zustand von %s lesen: %v", leserID, err)
		}
		return imPapierkorb, konten
	}
	hatZugang := func(t *testing.T, leserID string) bool {
		t.Helper()
		rec := alsAdmin(t, http.MethodGet, "/api/schueler/"+leserID, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/schueler/%s = %d %s, erwartet 200", leserID, rec.Code, rec.Body.String())
		}
		var akte struct {
			HatZugang *bool `json:"hat_zugang"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &akte); err != nil {
			t.Fatalf("Akte lesen: %v", err)
		}
		if akte.HatZugang == nil {
			t.Fatalf("GET /api/schueler/%s = %s, erwartet ein Feld hat_zugang", leserID, rec.Body.String())
		}
		return *akte.HatZugang
	}
	// abgewiesen prüft eine Ablehnung der Tür: der Grund in der Antwort, und nichts ist angefasst.
	abgewiesen := func(t *testing.T, rufe func(*testing.T, string, string, string) *httptest.ResponseRecorder, leserID, grund string) {
		t.Helper()
		rec := rufe(t, http.MethodDelete, "/api/schueler/"+leserID, "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("DELETE /api/schueler/%s = %d %s, erwartet 403", leserID, rec.Code, rec.Body.String())
		}
		if meldung := tuertest.Fehlermeldung(t, rec); !strings.Contains(meldung, grund) {
			t.Errorf("DELETE /api/schueler/%s meldet %q, erwartet den Grund %q", leserID, meldung, grund)
		}
		if imPapierkorb, konten := zustand(t, leserID); imPapierkorb || konten != 1 {
			t.Errorf("nach der Ablehnung: im Papierkorb %v, Konten %d, erwartet false und 1", imPapierkorb, konten)
		}
	}
	geloescht := func(t *testing.T, rufe func(*testing.T, string, string, string) *httptest.ResponseRecorder, leserID string) {
		t.Helper()
		if rec := rufe(t, http.MethodDelete, "/api/schueler/"+leserID, ""); rec.Code != http.StatusOK {
			t.Fatalf("DELETE /api/schueler/%s = %d %s, erwartet 200", leserID, rec.Code, rec.Body.String())
		}
		if imPapierkorb, konten := zustand(t, leserID); !imPapierkorb || konten != 0 {
			t.Errorf("nach dem Löschen: im Papierkorb %v, Konten %d, erwartet true und 0", imPapierkorb, konten)
		}
	}
	const ohneKontenrecht = "Löschen kann sie nur, wer „Benutzer & Rechte verwalten“ darf"
	const nurAdministrator = "Das Konto eines Administrators löscht nur ein Administrator"

	t.Run("ohne manage_users bleibt ein Kollege mit Zugang stehen", func(t *testing.T) {
		kora := leserMitKonto(t, "kora@loesch-zugang.example", "kollegium", true)
		if !hatZugang(t, kora) {
			t.Errorf("hat_zugang einer Lehrkraft mit Schuladresse = false, erwartet true")
		}
		abgewiesen(t, alsMitarbeiter, kora, ohneKontenrecht)
	})

	t.Run("ein Antrag, der noch nicht freigeschaltet ist, zählt als Zugang", func(t *testing.T) {
		kim := leserMitKonto(t, "kim@loesch-zugang.example", "kollegium", false)
		if !hatZugang(t, kim) {
			t.Errorf("hat_zugang eines inaktiven Kontos = false, erwartet true")
		}
		abgewiesen(t, alsMitarbeiter, kim, ohneKontenrecht)
	})

	t.Run("das Konto eines Administrators löscht nur ein Administrator", func(t *testing.T) {
		anna := leserMitKonto(t, "anna@loesch-zugang.example", "admin", true)
		abgewiesen(t, alsMitarbeiter, anna, ohneKontenrecht)
		abgewiesen(t, alsLeitung, anna, nurAdministrator)
		geloescht(t, alsAdmin, anna)
	})

	t.Run("mit manage_users fällt der Zugang mit der Leserzeile", func(t *testing.T) {
		karl := leserMitKonto(t, "karl@loesch-zugang.example", "kollegium", true)
		geloescht(t, alsLeitung, karl)
	})

	// Ein Zusammenführen kann das Konto eines Administrators an eine Zeile hängen, während sie in
	// den Papierkorb geht. Der Trigger stellt genau diesen Augenblick nach: Er hängt das Konto um,
	// sobald das Löschen die Zeile schreibt. Eine Prüfung vor der Transaktion sähe an der Zeile
	// noch kein Konto und ließe es fallen; geprüft wird deshalb an den Konten, die fallen.
	t.Run("ein Konto, das während des Löschens an die Zeile wandert, fällt nicht ohne Recht", func(t *testing.T) {
		ziel := testdaten.LegeLeserAn(t, pool, "", "Otto", "Wanderziel", "lehrkraft")
		bea := leserMitKonto(t, "bea@loesch-zugang.example", "admin", true)
		if _, err := pool.Exec(ctx, `
			CREATE OR REPLACE FUNCTION test_konto_wandert() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				UPDATE benutzer SET leser_id = NEW.id WHERE email = 'bea@loesch-zugang.example';
				RETURN NEW;
			END $$;
			CREATE TRIGGER test_konto_wandert AFTER UPDATE OF deleted_at ON leser FOR EACH ROW
				WHEN (NEW.nachname = 'Wanderziel' AND NEW.deleted_at IS NOT NULL)
				EXECUTE FUNCTION test_konto_wandert();`); err != nil {
			t.Fatalf("Trigger anlegen: %v", err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS test_konto_wandert ON leser;
				DROP FUNCTION IF EXISTS test_konto_wandert();`); err != nil {
				t.Errorf("Trigger entfernen: %v", err)
			}
		})

		rec := alsMitarbeiter(t, http.MethodDelete, "/api/schueler/"+ziel, "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("DELETE /api/schueler/%s = %d %s, erwartet 403", ziel, rec.Code, rec.Body.String())
		}
		if meldung := tuertest.Fehlermeldung(t, rec); !strings.Contains(meldung, ohneKontenrecht) {
			t.Errorf("DELETE /api/schueler/%s meldet %q, erwartet den Grund %q", ziel, meldung, ohneKontenrecht)
		}
		if imPapierkorb, konten := zustand(t, ziel); imPapierkorb || konten != 0 {
			t.Errorf("Zielzeile nach der Ablehnung: im Papierkorb %v, Konten %d, erwartet false und 0", imPapierkorb, konten)
		}
		if _, konten := zustand(t, bea); konten != 1 {
			t.Errorf("Konto des Administrators an seiner Zeile: %d, erwartet 1", konten)
		}
	})

	t.Run("ohne Zugang genügt das Recht der Route", func(t *testing.T) {
		for name, leserID := range map[string]string{
			"Praktikum": testdaten.LegeLeserAn(t, pool, "", "Pia", "Praktikum", "praktikum"),
			"Schüler":   testdaten.LegeSchuelerAn(t, pool, "S-LOESCH-ZUGANG", "Sami", "07A"),
			"Littera":   leserMitKonto(t, "littera-4711@littera.invalid", "kollegium", true),
		} {
			if hatZugang(t, leserID) {
				t.Errorf("%s: hat_zugang = true, erwartet false", name)
			}
			rec := alsMitarbeiter(t, http.MethodDelete, "/api/schueler/"+leserID, "")
			if rec.Code != http.StatusOK {
				t.Errorf("%s: DELETE /api/schueler/{id} als Mitarbeiter = %d %s, erwartet 200", name, rec.Code, rec.Body.String())
				continue
			}
			if imPapierkorb, konten := zustand(t, leserID); !imPapierkorb || konten != 0 {
				t.Errorf("%s: im Papierkorb %v, Konten %d, erwartet true und 0", name, imPapierkorb, konten)
			}
		}
	})
}
