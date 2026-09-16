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
)

// „Neuen Leser anlegen" fragt zuerst nach der Art. Ein Schüler entsteht wie bisher; eine
// Lehrkraft bekommt eine LESERZEILE und KEIN Konto — das holt sie sich über die
// Selbstanmeldung mit ihrer Schuladresse (16.09.2026).
//
// Vier Dinge unterscheiden die beiden Wege, und jedes davon wäre einzeln ein stiller
// Fehler:
//
//   - Klasse und Abgangsjahr sind Pflicht — aber nur für einen Schüler. Ein Kollege mit
//     Klasse stünde in Klassenlisten und im LUSD-Abgleich.
//   - Das Geburtsdatum ist der Schlüssel des LUSD-Abgleichs. Ein Kollege kommt nie aus
//     der LUSD; es von ihm zu verlangen, wäre eine Angabe ohne Zweck (Datenminimierung).
//   - Die Ausweisnummer kommt aus demselben Nummernkreis wie bei einem Schüler.
//   - Ein zweiter Eintrag für denselben Kollegen teilt seine Ausleihen auf zwei Akten,
//     ohne dass es jemand merkt. Er hat kein Geburtsdatum, an dem die Doppelprüfung
//     der Schüler greift — also prüft der Name.
func TestLeserAnlegen_ArtEntscheidet(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	claims := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a128", Rolle: auth.RoleAdmin}

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname IN ('Anlegetest', 'Anlegetest2')`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	anlegen := func(t *testing.T, koerper string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/schueler", strings.NewReader(koerper))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.CreateStudentHandler().ServeHTTP(rec, req)
		return rec
	}

	// 1. Lehrkraft: ohne Klasse, ohne Geburtsdatum — und sie bekommt eine Nummer.
	rec := anlegen(t, `{"art":"lehrkraft","vorname":"Katrin","nachname":"Anlegetest","email":"katrin.anlegetest@schule.invalid"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Lehrkraft anlegen: Status %d, %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		ID        string `json:"id"`
		BarcodeID string `json:"barcode_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
	}
	if antwort.BarcodeID == "" {
		t.Error("ohne Ausweisnummer lässt sich kein Ausweis drucken — sie gehört vergeben")
	}

	var art, klasse string
	var abgang *int
	var hatKonto bool
	if err := pool.QueryRow(ctx, `
		SELECT l.art, coalesce(l.klasse, ''), l.abgaenger_jahr,
		       EXISTS(SELECT 1 FROM benutzer b WHERE b.leser_id = l.id)
		FROM leser l WHERE l.id = $1`, antwort.ID).Scan(&art, &klasse, &abgang, &hatKonto); err != nil {
		t.Fatalf("Leserzeile lesen: %v", err)
	}
	if art != "lehrkraft" {
		t.Errorf("Art: %q, erwartet \"lehrkraft\"", art)
	}
	if klasse != "" || abgang != nil {
		t.Errorf("eine Lehrkraft hat keine Klasse und kein Abgangsjahr: %q / %v", klasse, abgang)
	}
	// UMGEKEHRT seit dem 16.09.2026 : Die Leserdatei legt das Konto MIT an, weil die
	// Schul-E-Mail beim Anlegen Pflicht ist. Nur so findet die spätere Selbstanmeldung über
	// „Mein Portal" diesen Eintrag wieder, statt einen zweiten anzulegen — der Doppeleintrag
	// war sonst unbemerkt und von niemandem zu reparieren. Der Zugang selbst bleibt eine
	// Entscheidung: Freigeschaltet wird nur, wenn der Anlegende `manage_users` hat.
	if !hatKonto {
		t.Error("die Leserdatei legt das Konto mit an — sonst steht die Lehrkraft nach der Selbstanmeldung doppelt da")
	}

	// 2. Derselbe Name ein zweites Mal: Das ist fast immer der Kollege, der sich längst
	//    selbst angemeldet hat. Zwei Akten teilen seine Ausleihen auf.
	// Andere Adresse als oben: Sonst schlüge die eindeutige E-Mail zu, und der Test
	// prüfte nicht mehr die NAMENS-Dublette, die er prüfen will.
	if rec := anlegen(t, `{"art":"lehrkraft","vorname":"Katrin","nachname":"Anlegetest","email":"k.anlegetest2@schule.invalid"}`); rec.Code != http.StatusConflict {
		t.Errorf("zweiter Eintrag mit demselben Namen: Status %d, erwartet 409 — %s", rec.Code, rec.Body.String())
	}

	// 3. Ein Schüler geht weiter nur MIT Klasse und Geburtsdatum durch.
	if rec := anlegen(t, `{"art":"schueler","vorname":"Lena","nachname":"Anlegetest2","klasse":"7a"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Schüler ohne Geburtsdatum: Status %d, erwartet 400 — %s", rec.Code, rec.Body.String())
	}
	rec = anlegen(t, `{"art":"schueler","vorname":"Lena","nachname":"Anlegetest2","klasse":"7a","geburtsdatum":"2012-03-04"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Schüler anlegen: Status %d, %s", rec.Code, rec.Body.String())
	}
	// Ohne Art-Angabe gilt weiter „Schüler" — die Vorgabe der Spalte.
	rec = anlegen(t, `{"vorname":"Leon","nachname":"Anlegetest2","klasse":"7a","geburtsdatum":"2012-05-06"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Schüler ohne Art-Angabe: Status %d, %s", rec.Code, rec.Body.String())
	}
	var artOhneAngabe string
	if err := pool.QueryRow(ctx,
		`SELECT art FROM leser WHERE vorname = 'Leon' AND nachname = 'Anlegetest2'`).Scan(&artOhneAngabe); err != nil {
		t.Fatalf("Art ohne Angabe lesen: %v", err)
	}
	if artOhneAngabe != "schueler" {
		t.Errorf("ohne Art-Angabe: %q, erwartet \"schueler\"", artOhneAngabe)
	}

	// 4. Eine Lehrkraft MIT Klasse ist ein Widerspruch: Sie stünde in Klassenlisten und
	//    fiele beim nächsten LUSD-Abgleich auf.
	if rec := anlegen(t, `{"art":"lehrkraft","vorname":"Falk","nachname":"Anlegetest","klasse":"7a","email":"falk.anlegetest@schule.invalid"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Lehrkraft mit Klasse: Status %d, erwartet 400 — %s", rec.Code, rec.Body.String())
	}

	// 5. Eine unbekannte Art ist ein Tippfehler, kein neuer Personenkreis.
	if rec := anlegen(t, `{"art":"hausmeister","vorname":"Udo","nachname":"Anlegetest"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unbekannte Art: Status %d, erwartet 400 — %s", rec.Code, rec.Body.String())
	}
}
