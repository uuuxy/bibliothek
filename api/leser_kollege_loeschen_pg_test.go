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

// Einen Kollegen löschen — und mit ihm seinen Zugang.
//
// Entschieden am 16.09.2026: „wenn ein kollege gelöscht wird dann wird alles gelöscht
// oder nicht das wäre doch das logischste oder nicht." Bis dahin ging das gar nicht: Der ganze
// Löschweg (Prüfung, Soft-Delete, Papierkorb, Wiederherstellen, endgültiges Entfernen)
// schrieb gegen die SICHT `schueler`, die nur Schüler zeigt. Bei einem Kollegen traf er
// null Zeilen und antwortete „nicht gefunden" — deshalb stand der Löschknopf in seiner
// Akte gar nicht erst (docs/OFFEN.md 5.16 C).
//
// Das Konto bleibt nicht als abgeschaltete Hülle stehen: Es hielte die Adresse besetzt
// (benutzer_email_unique), und wer die Person danach neu anlegt, bekäme „steht bereits ein
// Zugang" — über einen Eintrag, der im Papierkorb liegt und nirgends zu sehen ist.
func TestKollegeLoeschen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)

	loesche := func(t *testing.T, id, alsKonto string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: alsKonto, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.DeleteStudentHandler(auditRepo)(rec, req)
		return rec
	}

	// Ein Kollege mit Konto, wie ihn „Neuer Leser" seit dem 16.09.2026 anlegt.
	kollegeMitKonto := func(t *testing.T, vorname, nachname, email string) (leserID, kontoID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art) VALUES ($1, $2, 'lehrkraft') RETURNING id::text`,
			vorname, nachname).Scan(&leserID); err != nil {
			t.Fatalf("Leserzeile anlegen: %v", err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
			VALUES ($1, $2, $3, 'kollegium', true, $4) RETURNING id::text`,
			vorname, nachname, email, leserID).Scan(&kontoID); err != nil {
			t.Fatalf("Konto anlegen: %v", err)
		}
		return leserID, kontoID
	}

	zaehle := func(t *testing.T, sql string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("zählen: %v", err)
		}
		return n
	}

	// Ein ECHTES Admin-Konto: `audit_log.bearbeiter_id` trägt einen Fremdschlüssel auf
	// benutzer(id). Eine erfundene UUID liess den Löschweg mit 500 enden — am Protokoll,
	// nicht an der Sache.
	var admin string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Lösch', 'Admin', 'loesch.admin@schule.invalid', 'admin', true)
		RETURNING id::text`).Scan(&admin); err != nil {
		t.Fatalf("Admin anlegen: %v", err)
	}

	t.Run("die Leserzeile wandert in den Papierkorb, das Konto ist weg", func(t *testing.T) {
		leserID, _ := kollegeMitKonto(t, "Karla", "Loeschbar", "karla.loeschbar@schule.invalid")

		if rec := loesche(t, leserID, admin); rec.Code != http.StatusOK {
			t.Fatalf("Antwort %d: %s", rec.Code, rec.Body.String())
		}

		var geloescht bool
		if err := pool.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1`, leserID).Scan(&geloescht); err != nil {
			t.Fatalf("Leserzeile lesen: %v", err)
		}
		if !geloescht {
			t.Error("die Leserzeile liegt nicht im Papierkorb")
		}
		if n := zaehle(t, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, leserID); n != 0 {
			t.Errorf("das Konto steht noch (%d)", n)
		}
		// Die Adresse ist wieder frei — sonst scheiterte die Neuanlage an einem Eintrag,
		// den niemand sieht.
		if n := zaehle(t, `SELECT count(*) FROM benutzer WHERE lower(email) = 'karla.loeschbar@schule.invalid'`); n != 0 {
			t.Errorf("die Adresse ist weiter belegt (%d)", n)
		}
	})

	t.Run("der gelöschte Kollege steht im Papierkorb", func(t *testing.T) {
		leserID, _ := kollegeMitKonto(t, "Paul", "Papierkorb", "paul.papierkorb@schule.invalid")
		if rec := loesche(t, leserID, admin); rec.Code != http.StatusOK {
			t.Fatalf("löschen: %d %s", rec.Code, rec.Body.String())
		}

		req := httptest.NewRequest(http.MethodGet, "/api/schueler/deleted", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.GetDeletedStudentsHandler()(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Papierkorb: %d %s", rec.Code, rec.Body.String())
		}
		// Ohne die Zeile im Papierkorb wäre das Löschen eine Einbahnstraße: Wer sich
		// vertippt hat, käme an den Eintrag nicht mehr heran.
		if !strings.Contains(rec.Body.String(), "Papierkorb") {
			t.Errorf("der gelöschte Kollege fehlt im Papierkorb: %s", rec.Body.String())
		}
	})

	t.Run("wiederhergestellt kommt er ohne Zugang zurück", func(t *testing.T) {
		leserID, _ := kollegeMitKonto(t, "Rita", "Rueckkehr", "rita.rueckkehr@schule.invalid")
		if rec := loesche(t, leserID, admin); rec.Code != http.StatusOK {
			t.Fatalf("löschen: %d %s", rec.Code, rec.Body.String())
		}

		req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+leserID+"/restore", nil)
		req.SetPathValue("id", leserID)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.RestoreStudentHandler()(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiederherstellen: %d %s", rec.Code, rec.Body.String())
		}

		var imPapierkorb, gesperrt bool
		if err := pool.QueryRow(ctx,
			`SELECT deleted_at IS NOT NULL, ist_gesperrt FROM leser WHERE id = $1`, leserID).Scan(&imPapierkorb, &gesperrt); err != nil {
			t.Fatalf("Leserzeile lesen: %v", err)
		}
		if imPapierkorb {
			t.Error("die Zeile liegt weiter im Papierkorb")
		}
		// Die Lösch-Sperre muss fallen, sonst steht ein Kollege zurück, der nichts ausleihen
		// kann (Zombie-Sperre).
		if gesperrt {
			t.Error("der wiederhergestellte Kollege bleibt gesperrt")
		}
		// Der Zugang kommt NICHT von allein zurück — er wird über die Schul-E-Mail in der
		// Akte neu angelegt.
		if n := zaehle(t, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, leserID); n != 0 {
			t.Errorf("ein Konto ist wieder aufgetaucht (%d)", n)
		}
	})

	t.Run("der eigene Eintrag lässt sich nicht löschen", func(t *testing.T) {
		leserID, kontoID := kollegeMitKonto(t, "Selbst", "Schutz", "selbst.schutz@schule.invalid")
		rec := loesche(t, leserID, kontoID)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Antwort %d (erwartet 403): %s", rec.Code, rec.Body.String())
		}
		if n := zaehle(t, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, leserID); n != 1 {
			t.Errorf("das eigene Konto wurde angetastet (%d)", n)
		}
	})

	t.Run("mit offenen Büchern wird nicht gelöscht", func(t *testing.T) {
		leserID, _ := kollegeMitKonto(t, "Bert", "Buchoffen", "bert.buchoffen@schule.invalid")
		// Selbst angelegt statt im Bestand gesucht: resetBestandsdaten räumt ihn weg, und ein
		// Skip auf leerer Tabelle wäre ein grüner Test ohne Aussage.
		var titelID, exemplarID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel) VALUES ('Loesch-Titel') RETURNING id::text`).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-LOESCH-1') RETURNING id::text`,
			titelID).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, NOW(), NOW() + INTERVAL '30 days')`, exemplarID, leserID); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}

		rec := loesche(t, leserID, admin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		// Und das Konto steht noch: Eine abgewiesene Löschung darf nichts angefasst haben.
		if n := zaehle(t, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, leserID); n != 1 {
			t.Errorf("das Konto ist trotz Abweisung weg (%d)", n)
		}
	})
}
