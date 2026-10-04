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
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
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

	// mahnungenInDerListe liest aus der Mahnliste, wie oft und wann zuletzt zu einer
	// Ausleihe gemahnt wurde.
	mahnungenInDerListe := func(t *testing.T, ausleiheID string) (int, string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/mahnwesen", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnliste: Status %d — %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Klassen []repository.MahnwesenKlasse `json:"klassen"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Mahnliste lesen: %v", err)
		}
		for _, kl := range antwort.Klassen {
			for _, sch := range kl.Schueler {
				for _, m := range sch.Medien {
					if m.AusleiheID == ausleiheID {
						return m.Mahnstufe, m.LetztesMahndatum
					}
				}
			}
		}
		t.Fatalf("Ausleihe %s steht nicht in der Mahnliste", ausleiheID)
		return 0, ""
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

	// Aus der Auswahl kommt ein Papier: der Brief an die Eltern mit Anschrift für das
	// Fensterkuvert. Die Klasse steht daneben, weil ein Brief ohne Anschrift über das Kind
	// mitgeht.
	t.Run("der Druck aus der Auswahl ist der Brief an die Eltern", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-6", "Ida", "07H2")
		if _, err := pool.Exec(ctx, `
			UPDATE schueler SET strasse = 'Lindenweg', hausnummer = '4', plz = '61169', ort = 'Friedberg'
			WHERE id = $1`, kind); err != nil {
			t.Fatalf("Anschrift setzen: %v", err)
		}
		buch := seedAusleihe(t, pool, kind, "Band Elternbrief", abgelaufen)

		rec := drucke(t, buch)
		if rec.Code != http.StatusOK {
			t.Fatalf("Druck: Status %d — %s", rec.Code, rec.Body.String())
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		for _, soll := range []string{"Eltern von Ida Test", "Lindenweg 4", "61169 Friedberg", "Klasse: 07H2", "Band Elternbrief"} {
			if !strings.Contains(blatt, soll) {
				t.Errorf("auf dem Brief fehlt %q:\n%s", soll, blatt)
			}
		}
		// Das frühere Blatt in Du-Form sprach das Kind an.
		if strings.Contains(blatt, "Bitte gib") {
			t.Errorf("aus der Auswahl kommt das Blatt an das Kind statt des Briefs:\n%s", blatt)
		}
		if stufe, datum := mahnState(t, pool, buch); stufe != 1 || datum == nil {
			t.Errorf("der Brief zählt die Mahnung: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 1 und gesetzt",
				stufe, datum != nil)
		}
	})

	// Die Mahnliste zeigt je Kind, wie oft und wann zuletzt gemahnt wurde; die Zahl dazu
	// liefert die Liste je Buch.
	t.Run("nach dem Druck nennt die Mahnliste die Mahnung", func(t *testing.T) {
		kind := seedSchueler(t, pool, "MBF-S-9", "Rosa", "06G1")
		buch := seedAusleihe(t, pool, kind, "Band Gezaehlt", abgelaufen)

		if stufe, datum := mahnungenInDerListe(t, buch); stufe != 0 || datum != "" {
			t.Errorf("vor dem Druck: Liste nennt %d Mahnungen und das Datum %q, erwartet 0 und keins", stufe, datum)
		}
		if rec := drucke(t, buch); rec.Code != http.StatusOK {
			t.Fatalf("Druck: Status %d — %s", rec.Code, rec.Body.String())
		}
		heute := schulzeit.Jetzt().Format(dateFormatISO)
		if stufe, datum := mahnungenInDerListe(t, buch); stufe != 1 || datum != heute {
			t.Errorf("nach dem Druck: Liste nennt %d Mahnungen und das Datum %q, erwartet 1 und %s", stufe, datum, heute)
		}
	})

	t.Run("ab 18 geht der Brief an die Person selbst", func(t *testing.T) {
		erwachsen := seedSchueler(t, pool, "MBF-S-7", "Jonas", "13T1")
		if _, err := pool.Exec(ctx, `
			UPDATE schueler SET geburtsdatum = CURRENT_DATE - INTERVAL '19 years' WHERE id = $1`, erwachsen); err != nil {
			t.Fatalf("Geburtsdatum setzen: %v", err)
		}
		buch := seedAusleihe(t, pool, erwachsen, "Band Oberstufe", abgelaufen)

		rec := drucke(t, buch)
		if rec.Code != http.StatusOK {
			t.Fatalf("Druck: Status %d — %s", rec.Code, rec.Body.String())
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		if !strings.Contains(blatt, "Jonas Test") || !strings.Contains(blatt, "Band Oberstufe") {
			t.Fatalf("Name oder Buch fehlen auf dem Brief — der Leser sieht das Blatt nicht:\n%s", blatt)
		}
		if strings.Contains(blatt, "Eltern") {
			t.Errorf("der Brief an einen Volljährigen nennt Eltern:\n%s", blatt)
		}
	})

	// Ein Ehemaliger steht in der Mahnliste unter „Ehemalige" und geht an keine
	// Klassenleitung; den Brief bekommt er wie jeder andere über die Auswahl.
	t.Run("ein Ehemaliger bekommt den Brief", func(t *testing.T) {
		ehemalig := seedSchueler(t, pool, "MBF-S-8", "Erik", "08H3")
		if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = true WHERE id = $1`, ehemalig); err != nil {
			t.Fatalf("als Ehemaligen kennzeichnen: %v", err)
		}
		buch := seedAusleihe(t, pool, ehemalig, "Band Ehemalig", abgelaufen)

		rec := drucke(t, buch)
		if rec.Code != http.StatusOK {
			t.Fatalf("Druck für einen Ehemaligen: Status %d, erwartet 200 mit dem Brief — %s",
				rec.Code, firstBytes(rec.Body.Bytes(), 160))
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		for _, soll := range []string{"Eltern von Erik Test", "Band Ehemalig"} {
			if !strings.Contains(blatt, soll) {
				t.Errorf("auf dem Brief des Ehemaligen fehlt %q:\n%s", soll, blatt)
			}
		}
		// Der Klassenname gehört nach der Versetzung einem anderen Jahrgang.
		if strings.Contains(blatt, "Klasse:") {
			t.Errorf("der Brief des Ehemaligen nennt eine Klasse:\n%s", blatt)
		}
		if stufe, _ := mahnState(t, pool, buch); stufe != 1 {
			t.Errorf("Buch des Ehemaligen: Mahnstufe %d, erwartet 1", stufe)
		}
	})
}
