package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
	"bibliothek/sse"
)

// jahrgangsFall ist ein Kind mit einem offenen Buch: seine Klasse, die Spanne des Titels
// und was die Ansicht „Jahrgang" dazu sagen muss.
type jahrgangsFall struct {
	vorname, klasse string
	ehemalig        bool
	bisKlasse       int
	steht           bool
	darueber        int
}

// jahrgangsZeile ist, was die Ansicht zu einem Kind nennt.
type jahrgangsZeile struct {
	klasse, frist string
	darueber      int
}

// Die Ansicht „Jahrgang" des Mahnwesens am Router des Betriebs. Sie nennt ein Buch, wenn
// der Jahrgang des Kindes über der Spanne des Titels liegt. Die Klassen der Schule tragen
// hinter dem Jahrgang Zweig und Zug („05F1"), und die Einführungsphase heißt „ET": Wer
// alle Ziffern der Klasse liest, hält die 05F1 für Jahrgang 51 und die ET1 für Jahrgang 1.
func TestMahnwesenJahrgang_LiestDenJahrgangNichtDenZug(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"mahnwesen-jahrgang-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Jana', 'Jahrgang', 'mahnwesen-jahrgang@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "MJG-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	titelBis := map[int]string{}
	for _, bis := range []int{6, 10} {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, jahrgang_von, jahrgang_bis)
			VALUES ($1, 5, $2) RETURNING id`, fmt.Sprintf("Jahrgangsband bis %d", bis), bis).Scan(&id); err != nil {
			t.Fatalf("Titel bis Klasse %d anlegen: %v", bis, err)
		}
		titelBis[bis] = id
	}

	// leihe legt das Kind an und gibt ihm ein eigenes Exemplar des Titels mit der Spanne.
	leihe := func(nr int, f jahrgangsFall) {
		t.Helper()
		var schuelerID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
			VALUES ($1, $2, 'Jahrgangstest', $3, 2031, $4) RETURNING id`,
			fmt.Sprintf("MJG-S-%d", nr), f.vorname, f.klasse, f.ehemalig).Scan(&schuelerID); err != nil {
			t.Fatalf("%s in %s anlegen: %v", f.vorname, f.klasse, err)
		}
		exemplarID := exemplar(t, pool, titelBis[f.bisKlasse], fmt.Sprintf("MJG-B-%d", nr), false, "")
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id)
			VALUES ($1, $2, now() - interval '30 days', now() + interval '30 days', $3)`,
			exemplarID, schuelerID, kontoID); err != nil {
			t.Fatalf("Ausleihe für %s anlegen: %v", f.vorname, err)
		}
	}

	// hole ruft die Ansicht über den Router und liefert je Vorname die genannte Zeile.
	hole := func(t *testing.T) map[string]jahrgangsZeile {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/mahnwesen/ueberfaellig_jahrgang", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/mahnwesen/ueberfaellig_jahrgang: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Klassen []struct {
				Klasse   string `json:"klasse"`
				Schueler []struct {
					Name   string `json:"name"`
					Medien []struct {
						FaelligAm        string `json:"faellig_am"`
						TageUeberfaellig int    `json:"tage_ueberfaellig"`
					} `json:"medien"`
				} `json:"schueler"`
			} `json:"klassen"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v — %s", err, rec.Body.String())
		}
		zeilen := map[string]jahrgangsZeile{}
		for _, k := range antwort.Klassen {
			for _, s := range k.Schueler {
				vorname := strings.TrimSuffix(s.Name, " Jahrgangstest")
				if len(s.Medien) != 1 {
					t.Fatalf("%s (%s): %d Bücher in der Liste, erwartet 1", vorname, k.Klasse, len(s.Medien))
				}
				zeilen[vorname] = jahrgangsZeile{klasse: k.Klasse, frist: s.Medien[0].FaelligAm, darueber: s.Medien[0].TageUeberfaellig}
			}
		}
		return zeilen
	}

	pruefe := func(t *testing.T, faelle []jahrgangsFall) {
		t.Helper()
		zeilen := hole(t)
		erwartet := 0
		for _, f := range faelle {
			zeile, da := zeilen[f.vorname]
			if da != f.steht {
				t.Errorf("%s in %s, Buch bis Klasse %d: in der Liste = %v, erwartet %v", f.vorname, f.klasse, f.bisKlasse, da, f.steht)
			}
			if !f.steht {
				continue
			}
			erwartet++
			soll := jahrgangsZeile{klasse: f.klasse, frist: fmt.Sprintf("bis Kl. %d", f.bisKlasse), darueber: f.darueber}
			if da && zeile != soll {
				t.Errorf("%s in %s: die Liste nennt %+v, erwartet %+v", f.vorname, f.klasse, zeile, soll)
			}
		}
		if len(zeilen) != erwartet {
			t.Errorf("die Liste nennt %d Kinder, erwartet %d: %v", len(zeilen), erwartet, zeilen)
		}
	}

	faelle := []jahrgangsFall{
		{vorname: "Fuenf", klasse: "05F1", bisKlasse: 10},
		{vorname: "Sechs", klasse: "06G6", bisKlasse: 6},
		{vorname: "Sieben", klasse: "07H2", bisKlasse: 6, steht: true, darueber: 1},
		{vorname: "Zehn", klasse: "10R3", bisKlasse: 10},
		{vorname: "Elf", klasse: "ET1", bisKlasse: 10, steht: true, darueber: 1},
		{vorname: "Dreizehn", klasse: "13T3", bisKlasse: 10, steht: true, darueber: 3},
		{vorname: "Ehemalig", klasse: "ABG", ehemalig: true, bisKlasse: 10, steht: true},
		// Ein Ehemaliger, dessen Klasse stehen blieb: in der Liste, aber nicht „minus eins“.
		{vorname: "Gegangen", klasse: "09H1", ehemalig: true, bisKlasse: 10, steht: true},
	}
	for nr, f := range faelle {
		leihe(nr, f)
	}

	t.Run("der Jahrgang steht vorn in der Klasse", func(t *testing.T) {
		pruefe(t, faelle)
	})

	// Die erste Klasse heißt wie die der Browser-Tests, die zweite ist eine Buchnummer im
	// Feld Klasse. Die Ziffern beider passen in keine ganze Zahl der Datenbank.
	ohneJahrgang := []jahrgangsFall{
		{vorname: "Lang", klasse: "ZZZ-mtlt123476069028", bisKlasse: 6},
		{vorname: "Ziffern", klasse: "9783123456789", bisKlasse: 6},
	}
	for nr, f := range ohneJahrgang {
		leihe(len(faelle)+nr, f)
	}

	t.Run("eine Klasse ohne lesbaren Jahrgang nimmt die Liste der anderen nicht mit", func(t *testing.T) {
		pruefe(t, append(faelle, ohneJahrgang...))
	})

	t.Run("der Klassenfilter grenzt auf eine Klasse ein", func(t *testing.T) {
		klassen, err := repository.NewMahnwesenRepository(pool).QueryUeberfaelligeNachJahrgang(ctx, "ET1")
		if err != nil {
			t.Fatalf("QueryUeberfaelligeNachJahrgang: %v", err)
		}
		if len(klassen) != 1 || klassen[0].Klasse != "ET1" || len(klassen[0].Schueler) != 1 {
			t.Errorf("Filter ET1 liefert %+v, erwartet genau die Klasse ET1 mit einem Kind", klassen)
		}
	})
}
