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

// Ein Ehemaliger im Mahnwesen, am Router des Betriebs. Meldet die LUSD ein Kind mit offenen
// Büchern nicht mehr, wird es Ehemaliger und behält seine Klasse; die Versetzung fasst es
// danach nicht mehr an. Im Jahr darauf trägt ein anderer Jahrgang den Klassennamen. Die Liste
// an dessen Klassenleitung nennt den Ehemaligen nicht: Er steht in der Mahnliste als eigene
// Gruppe, und diese Gruppe geht an keine Klassenleitung.
func TestMahnwesen_EhemaligerGehtAnKeineKlassenleitung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"mahnwesen-ehemalige-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Mara', 'Mahnliste', 'mahnwesen-ehemalige@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "EHM-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()
	rufe := func(req *http.Request) *httptest.ResponseRecorder {
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	abgelaufen := time.Now().AddDate(0, 0, -30)

	// Erik ging mitten im Jahr aus der 08H3 und hat sein Buch noch. So markiert ihn der
	// Abgleich mit der LUSD: Ehemaliger, die Klasse bleibt stehen.
	erik := seedSchueler(t, pool, "EHM-S-1", "Erik", "08H3")
	seedAusleihe(t, pool, erik, "Atlas Alt", abgelaufen)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion: %v", err)
	}
	if err := sperreAbgaenger(ctx, tx, erik, "Abgänger laut LUSD"); err != nil {
		t.Fatalf("Erik als Ehemaligen markieren: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Nora ist in der 07H3 und wird versetzt; ihre Klassenleitung zieht mit.
	nora := seedSchueler(t, pool, "EHM-S-2", "Nora", "07H3")
	seedAusleihe(t, pool, nora, "Atlas Neu", abgelaufen)
	klassenleitung(t, pool, "07H3", "leitung-nora@schule.invalid")

	if rec := rufe(jsonPost("/api/students/promote", `{"confirm":true}`)); rec.Code != http.StatusOK {
		t.Fatalf("Versetzung: Status %d — %s", rec.Code, rec.Body.String())
	}
	var klasseErik, klasseNora, leitung string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE id = $1`, erik).Scan(&klasseErik); err != nil {
		t.Fatalf("Klasse von Erik lesen: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE id = $1`, nora).Scan(&klasseNora); err != nil {
		t.Fatalf("Klasse von Nora lesen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = '08H3'`).Scan(&leitung); err != nil {
		t.Fatalf("Klassenleitung der 08H3 lesen: %v", err)
	}
	if klasseErik != "08H3" || klasseNora != "08H3" || leitung != "leitung-nora@schule.invalid" {
		t.Fatalf("die Ausgangslage steht nicht: Erik in %q, Nora in %q, Klassenleitung der 08H3 %q — "+
			"erwartet beide in 08H3 und die Klassenleitung von Nora", klasseErik, klasseNora, leitung)
	}
	// Eine Zuordnung unter dem Namen der Gruppe macht aus ihr keine Klasse.
	klassenleitung(t, pool, "Ehemalige", "niemand@schule.invalid")

	t.Run("die Mahnliste führt den Ehemaligen als eigene Gruppe ohne Klassenleitung", func(t *testing.T) {
		rec := rufe(httptest.NewRequest(http.MethodGet, "/api/mahnwesen", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnliste: Status %d — %s", rec.Code, rec.Body.String())
		}
		var liste struct {
			Klassen []struct {
				Klasse      string `json:"klasse"`
				Ehemalige   bool   `json:"ehemalige"`
				LehrerEmail string `json:"lehrer_email"`
				Schueler    []struct {
					Name string `json:"name"`
				} `json:"schueler"`
			} `json:"klassen"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
			t.Fatalf("Mahnliste lesen: %v — %s", err, firstBytes(rec.Body.Bytes(), 200))
		}
		gefunden := map[string]bool{}
		for _, g := range liste.Klassen {
			for _, s := range g.Schueler {
				gefunden[s.Name] = true
				switch s.Name {
				case "Erik Test":
					if !g.Ehemalige || g.LehrerEmail != "" {
						t.Errorf("Erik steht in der Gruppe %q (Ehemalige: %v, Klassenleitung %q) — "+
							"erwartet die Gruppe der Ehemaligen ohne Klassenleitung", g.Klasse, g.Ehemalige, g.LehrerEmail)
					}
				case "Nora Test":
					if g.Ehemalige || g.Klasse != "08H3" || g.LehrerEmail != "leitung-nora@schule.invalid" {
						t.Errorf("Nora steht in der Gruppe %q (Ehemalige: %v, Klassenleitung %q) — "+
							"erwartet die 08H3 mit ihrer Klassenleitung", g.Klasse, g.Ehemalige, g.LehrerEmail)
					}
				}
			}
		}
		if !gefunden["Erik Test"] || !gefunden["Nora Test"] {
			t.Errorf("in der Mahnliste fehlt ein Kind mit überfälligem Buch: %v", gefunden)
		}
	})

	t.Run("der Druck der Klasse nennt den Ehemaligen nicht", func(t *testing.T) {
		rec := rufe(httptest.NewRequest(http.MethodGet, "/api/print/mahnung/klasse/08H3", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Druck der 08H3: Status %d — %s", rec.Code, firstBytes(rec.Body.Bytes(), 200))
		}
		blatt := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		if !strings.Contains(blatt, "Nora") {
			t.Fatalf("Nora fehlt auf der Liste ihrer Klasse:\n%s", blatt)
		}
		if strings.Contains(blatt, "Erik") {
			t.Errorf("der Ehemalige steht auf der Liste der 08H3:\n%s", blatt)
		}
	})

	t.Run("die Mail an die Klassenleitung nennt den Ehemaligen nicht", func(t *testing.T) {
		sitzungen := mailAbfangen(t)
		rec := rufe(jsonPost("/api/mail/send-bulk-overdue", ""))
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnlauf: Status %d — %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Sent    int `json:"sent_count"`
			Skipped int `json:"skipped_count"`
			Failed  int `json:"failed_count"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v — %s", err, rec.Body.String())
		}
		// Der Testserver nimmt eine Sitzung an. Ginge die Gruppe der Ehemaligen als zweite
		// Mail hinaus, stünde sie hier als übersprungen oder fehlgeschlagen.
		if antwort.Sent != 1 || antwort.Skipped != 0 || antwort.Failed != 0 {
			t.Errorf("Mahnlauf: %d verschickt, %d übersprungen, %d fehlgeschlagen — erwartet genau die Mail der 08H3",
				antwort.Sent, antwort.Skipped, antwort.Failed)
		}
		nachricht := warteAufMail(t, sitzungen)
		if !strings.Contains(nachricht, "leitung-nora@schule.invalid") {
			t.Fatalf("die Mail ging nicht an die Klassenleitung der 08H3:\n%s", kopf(nachricht))
		}
		liste := strings.Join(pdfTexte(t, pdfAusMail(t, nachricht)), "\n")
		if !strings.Contains(liste, "Nora") {
			t.Fatalf("Nora fehlt in der Liste an ihre Klassenleitung:\n%s", liste)
		}
		if strings.Contains(liste, "Erik") {
			t.Errorf("der Ehemalige steht in der Liste an die Klassenleitung eines anderen Jahrgangs:\n%s", liste)
		}
	})

	t.Run("auch ausdrücklich gewählt geht die Gruppe der Ehemaligen an niemanden", func(t *testing.T) {
		sitzungen := mailAbfangen(t)
		rec := rufe(jsonPost("/api/mail/send-bulk-overdue",
			`{"klassen":["Ehemalige"],"override_email":"vertretung@schule.invalid"}`))
		if rec.Code != http.StatusOK {
			t.Fatalf("Mahnlauf: Status %d — %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Sent int `json:"sent_count"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v — %s", err, rec.Body.String())
		}
		if antwort.Sent != 0 {
			t.Errorf("Mahnlauf für die Gruppe der Ehemaligen: %d Mail verschickt, erwartet keine", antwort.Sent)
		}
		select {
		case s := <-sitzungen:
			t.Errorf("für die Ehemaligen ging eine Mail hinaus:\n%s", kopf(s.Nachricht))
		default:
		}
	})
}
