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
	"bibliothek/internal/pdftest"
	"bibliothek/sse"
)

// Der Mahnbrief am Router des Betriebs: Er entsteht nur für ein Buch, dessen Frist
// abgelaufen ist. Die Auswahl kommt aus der Oberfläche; zwischen dem Laden der Liste und
// dem Druck kann eine Frist verlängert worden sein, und eine Auswahl kann Ausleihen nennen,
// die nie überfällig waren. Ein solches Buch steht nicht auf dem Blatt, und seine Mahnung
// wird nicht gezählt.
func TestMahnbriefDruck_NurMitAbgelaufenerFrist(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"mahnbrief-frist-testgeheimnis-32-bytes-lang!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Mara', 'Mahnbrief', 'mahnbrief-frist@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "MBF-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	// drucke schickt die Auswahl an die Tür, wie es „Mahnbriefe drucken" tut.
	drucke := func(t *testing.T, ausleihIDs ...string) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(BulkPrintRequest{AusleihIDs: ausleihIDs})
		if err != nil {
			t.Fatalf("Auswahl serialisieren: %v", err)
		}
		req := jsonPost("/api/admin/mahnungen/bulk-print", string(body))
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	abgelaufen := time.Now().AddDate(0, 0, -30)
	laeuft := time.Now().AddDate(0, 0, 60)

	t.Run("eine Auswahl nur aus laufenden Fristen ergibt keinen Mahnbrief", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-1", "Lena", "12T1")
		nurLaufend := seedAusleihe(t, pool, kind, "Band Laufend Allein", laeuft)

		rec := drucke(t, nurLaufend)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Mahnbrief für ein Buch mit Frist in 60 Tagen: Status %d, erwartet 404 — %s",
				rec.Code, firstBytes(rec.Body.Bytes(), 120))
		}
		if stufe, datum := mahnState(t, pool, nurLaufend); stufe != 0 || datum != nil {
			t.Errorf("Buch mit Frist in 60 Tagen: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 0 und nicht gesetzt",
				stufe, datum != nil)
		}
	})

	t.Run("in einer gemischten Auswahl wird nur das überfällige Buch gemahnt", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-2", "Timo", "07H2")
		ueberfaellig := seedAusleihe(t, pool, kind, "Band Abgelaufen", abgelaufen)
		laufend := seedAusleihe(t, pool, kind, "Band Laufend", laeuft)

		rec := drucke(t, ueberfaellig, laufend)
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnbrief für ein überfälliges und ein laufendes Buch: Status %d, erwartet 200 — %s",
				rec.Code, rec.Body.String())
		}

		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		if !strings.Contains(blatt, "Band Abgelaufen") {
			t.Fatalf("das überfällige Buch fehlt auf dem Blatt — der Leser sieht das Blatt nicht oder der Druck ist leer:\n%s", blatt)
		}
		if strings.Contains(blatt, "Band Laufend") {
			t.Errorf("das Buch mit Frist in 60 Tagen steht auf dem Mahnbrief:\n%s", blatt)
		}

		if stufe, datum := mahnState(t, pool, ueberfaellig); stufe != 1 || datum == nil {
			t.Errorf("überfälliges Buch: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 1 und gesetzt", stufe, datum != nil)
		}
		if stufe, datum := mahnState(t, pool, laufend); stufe != 0 || datum != nil {
			t.Errorf("Buch mit Frist in 60 Tagen: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 0 und nicht gesetzt",
				stufe, datum != nil)
		}
	})

	// Das Blatt nennt nur Schüler außerhalb des Papierkorbs. Wer nicht auf dem Blatt steht,
	// dessen Mahnung wird auch nicht gezählt.
	t.Run("gezählt wird nur, was auf dem Blatt steht", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-3", "Nora", "08H3")
		aufDemBlatt := seedAusleihe(t, pool, kind, "Band Schueler", abgelaufen)

		imPapierkorb := seedSchueler(t, pool, "MBF-S-4", "Paul", "08H3")
		geloescht := seedAusleihe(t, pool, imPapierkorb, "Band Papierkorb", abgelaufen)
		if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() WHERE id = $1`, imPapierkorb); err != nil {
			t.Fatalf("Schüler in den Papierkorb legen: %v", err)
		}

		var kollegeID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art) VALUES ('Karl', 'Kollege', 'lehrkraft')
			RETURNING id::text`).Scan(&kollegeID); err != nil {
			t.Fatalf("Kollegen anlegen: %v", err)
		}
		dauerleihe := seedAusleihe(t, pool, kollegeID, "Band Kollege", abgelaufen)
		if _, err := pool.Exec(ctx, `UPDATE ausleihen SET ist_handapparat = true WHERE id = $1`, dauerleihe); err != nil {
			t.Fatalf("Dauerleihe kennzeichnen: %v", err)
		}

		rec := drucke(t, aufDemBlatt, geloescht, dauerleihe)
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnbrief für die gemischte Auswahl: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		if !strings.Contains(blatt, "Band Schueler") {
			t.Fatalf("das überfällige Buch der Schülerin fehlt auf dem Blatt:\n%s", blatt)
		}
		for _, fremd := range []string{"Band Papierkorb", "Band Kollege"} {
			if strings.Contains(blatt, fremd) {
				t.Errorf("%q steht auf dem Mahnbrief:\n%s", fremd, blatt)
			}
		}

		if stufe, _ := mahnState(t, pool, aufDemBlatt); stufe != 1 {
			t.Errorf("Buch der Schülerin: Mahnstufe %d, erwartet 1", stufe)
		}
		if stufe, datum := mahnState(t, pool, geloescht); stufe != 0 || datum != nil {
			t.Errorf("Buch des Schülers im Papierkorb: Mahnstufe %d, Mahndatum gesetzt %v — es steht nicht auf dem Blatt, erwartet 0 und nicht gesetzt",
				stufe, datum != nil)
		}
		if stufe, datum := mahnState(t, pool, dauerleihe); stufe != 0 || datum != nil {
			t.Errorf("Dauerleihe des Kollegen: Mahnstufe %d, Mahndatum gesetzt %v — sie steht nicht auf dem Blatt, erwartet 0 und nicht gesetzt",
				stufe, datum != nil)
		}
	})

	// Nach einem Papierstau wird dieselbe Auswahl noch einmal gedruckt. Ein Buch steigt
	// höchstens einmal am Tag; das Blatt gibt es trotzdem.
	t.Run("ein zweiter Druck am selben Tag liefert das Blatt und zählt nicht", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-5", "Olga", "06G1")
		buch := seedAusleihe(t, pool, kind, "Band Nachdruck", abgelaufen)

		if rec := drucke(t, buch); rec.Code != http.StatusOK {
			t.Fatalf("erster Druck: Status %d — %s", rec.Code, rec.Body.String())
		}
		stufe, erstesDatum := mahnState(t, pool, buch)
		if stufe != 1 || erstesDatum == nil {
			t.Fatalf("nach dem ersten Druck: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 1 und gesetzt",
				stufe, erstesDatum != nil)
		}

		rec := drucke(t, buch)
		if rec.Code != http.StatusOK {
			t.Fatalf("zweiter Druck am selben Tag: Status %d, erwartet 200 mit dem Blatt — %s",
				rec.Code, firstBytes(rec.Body.Bytes(), 160))
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		if !strings.Contains(blatt, "Band Nachdruck") {
			t.Errorf("das Buch fehlt auf dem zweiten Blatt:\n%s", blatt)
		}
		stufe, datum := mahnState(t, pool, buch)
		if stufe != 1 {
			t.Errorf("nach dem zweiten Druck: Mahnstufe %d, erwartet weiter 1", stufe)
		}
		if datum == nil || !datum.Equal(*erstesDatum) {
			t.Errorf("nach dem zweiten Druck: Mahndatum %v, erwartet unverändert %v", datum, erstesDatum)
		}
	})
}
