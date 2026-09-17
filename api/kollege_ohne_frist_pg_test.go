package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Ein Kollege wird nicht überfällig — nirgends.
//
// Entschieden am 16.09.2026: „kollegen haben keine frist bzw werden einfach nie gesperrt!"
//
// Die Regel gab es schon, sie war nur nicht zu Ende geführt. Eine Ausleihe an jemanden,
// der kein Schüler ist, ist eine DAUERLEIHE (`ausleihen.ist_handapparat`, gesetzt in
// erzeugeAusleihe und im Geräte-Pfad), und die Sperr-Automatik zählt seit jeher nur
// Ausleihen mit `ist_handapparat = false`. Die Leserliste und die Übersicht rechneten
// daneben ihr eigenes „überfällig" aus dem blossen Datum: Nach einem Jahr stand ein
// Kollege mit roter Zahl in der Leserdatei und als Mahnfall in der Statistik, während die
// Theke ihn anstandslos bediente — zwei Wahrheiten über dieselbe Ausleihe.
//
// Gemahnt wurde er nie: Der Mahnlauf joint die Sicht `schueler`
// (repository/mahnwesen_queries.go). Das bleibt so.
func TestKollegeWirdNichtUeberfaellig(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	// Das Exemplar wird ANGELEGT und nicht im Bestand gesucht: resetBestandsdaten räumt ihn
	// weg. Ein `LIMIT 1` auf eine leere Tabelle führte hier erst zu einem Skip — und ein
	// übersprungener Test ist grün, ohne etwas zu belegen (gesehen an der Rot-Probe: Der
	// Rückbau der Regel liess den Test unverändert „ok" melden).
	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Frist-Titel') RETURNING id::text`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	// Je Leser ein eigenes Exemplar: `uniq_ausleihen_aktiv_exemplar` lässt nur EINE offene
	// Ausleihe je Exemplar zu.
	neuesExemplar := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id::text`,
			titelID, barcode).Scan(&id); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		return id
	}

	// Zwei Leser mit derselben, ein Jahr alten Ausleihe — der Unterschied ist allein die
	// Art und damit die Dauerleihe.
	leihe := func(t *testing.T, art string, nachname string, dauerleihe bool) string {
		exemplarID := neuesExemplar(t, "B-FRIST-"+nachname)
		t.Helper()
		var leserID string
		if art == "schueler" {
			if err := pool.QueryRow(ctx, `
				INSERT INTO leser (vorname, nachname, art, klasse, abgaenger_jahr, barcode_id, geburtsdatum)
				VALUES ('Frist', $1, 'schueler', '07A', 2031, $2, '2012-05-04') RETURNING id::text`,
				nachname, "A-FRIST-"+nachname).Scan(&leserID); err != nil {
				t.Fatalf("Schüler anlegen: %v", err)
			}
		} else {
			if err := pool.QueryRow(ctx, `
				INSERT INTO leser (vorname, nachname, art) VALUES ('Frist', $1, $2) RETURNING id::text`,
				nachname, art).Scan(&leserID); err != nil {
				t.Fatalf("Kollegen anlegen: %v", err)
			}
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, ist_handapparat)
			VALUES ($1, $2, NOW() - INTERVAL '400 days', NOW() - INTERVAL '35 days', $3)`,
			exemplarID, leserID, dauerleihe); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
		return leserID
	}

	kollegeID := leihe(t, "lehrkraft", "Kollege", true)
	schuelerID := leihe(t, "schueler", "Kind", false)

	profil := func(t *testing.T, id string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.GetStudentProfileHandler(repository.NewStudentRepository(pool))(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Profil %s: %d %s", id, rec.Code, rec.Body.String())
		}
		var antwort map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Profil lesen: %v", err)
		}
		return antwort
	}

	t.Run("die Akte nennt die Ausleihe des Kollegen als Dauerleihe", func(t *testing.T) {
		buecher, ok := profil(t, kollegeID)["entliehene_buecher"].([]any)
		if !ok {
			t.Fatal("die Akte liefert keine Ausleihliste")
		}
		if len(buecher) != 1 {
			t.Fatalf("erwartet ein Buch, gefunden: %d", len(buecher))
		}
		eintrag, ok := buecher[0].(map[string]any)
		if !ok {
			t.Fatal("die Ausleihzeile ist kein Objekt")
		}
		// Ohne dieses Feld kann die Oberfläche die Frist nicht von einer Dauerleihe
		// unterscheiden — sie rechnet dann wieder ihr eigenes „überfällig" aus dem Datum.
		if eintrag["ist_dauerleihe"] != true {
			t.Errorf("ist_dauerleihe fehlt oder ist falsch: %v", eintrag["ist_dauerleihe"])
		}
	})

	t.Run("beim Schüler bleibt es eine Frist", func(t *testing.T) {
		buecher, ok := profil(t, schuelerID)["entliehene_buecher"].([]any)
		if !ok {
			t.Fatal("die Akte liefert keine Ausleihliste")
		}
		if len(buecher) != 1 {
			t.Fatalf("erwartet ein Buch, gefunden: %d", len(buecher))
		}
		eintrag, ok := buecher[0].(map[string]any)
		if !ok {
			t.Fatal("die Ausleihzeile ist kein Objekt")
		}
		if eintrag["ist_dauerleihe"] != false {
			t.Errorf("die Ausleihe des Schülers gilt als Dauerleihe: %v", eintrag["ist_dauerleihe"])
		}
	})

	t.Run("die Leserliste zählt den Kollegen nicht als überfällig, den Schüler schon", func(t *testing.T) {
		zaehleUeberfaellig := func(t *testing.T, leserID string) int {
			t.Helper()
			// ListLeserMitStats ist die Abfrage der LESERDATEI — sie führt Schüler und
			// Kollegium in einer Liste und ist damit die Stelle, an der ein Kollege
			// überhaupt mit einer Zahl auftauchen kann.
			stats, err := repository.NewStudentRepository(pool).ListLeserMitStats(ctx, nil, "Frist", repository.SchuelerSortierung{})
			if err != nil {
				t.Fatalf("Leserliste: %v", err)
			}
			for _, s := range stats {
				if s.ID == leserID {
					return s.UeberfaelligCount
				}
			}
			t.Fatalf("Leser %s steht nicht in der Liste", leserID)
			return -1
		}

		if n := zaehleUeberfaellig(t, kollegeID); n != 0 {
			t.Errorf("der Kollege steht mit %d überfälligen Medien in der Leserdatei", n)
		}
		// Die Gegenprobe: Ohne sie misst der Test nur, dass niemand überfällig wird.
		if n := zaehleUeberfaellig(t, schuelerID); n != 1 {
			t.Errorf("der Schüler ist nicht mehr überfällig (%d) — die Regel greift zu weit", n)
		}
	})
}
