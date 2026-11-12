package lernmittel

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// Ein Schulbuch, das ab dem ersten Tag des Büchertauschs ausgegeben wird, gilt dem nächsten
// Schuljahr, gleich ob die Klasse im Plan steht. Der Ausleihdienst rechnet das beim Ausleihen;
// hier zieht der Plan es nach, wenn er erst während des Tauschs veröffentlicht wird oder sein
// Beginn nach vorn rückt. Ohne das bekämen diese Bücher den Termin ihrer Klasse und stünden
// danach überfällig da. Die Ausleihen tragen ihr Ausgabedatum von Hand, der Plan liegt in der
// Zukunft.
func TestLMFPlan_VeroeffentlichenWaehrendDesTauschs_SchulbuecherAbBeginnGeltenDemFolgendenSchuljahr(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	ende := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatalf("Testdatum %q: %v", d, err)
		}
		return schulzeit.TagesEnde(x)
	}
	laufend, folgend := ende("2027-07-31"), ende("2028-07-31")
	ausleihe := func(schueler, titel, ausgegeben string, frist time.Time) string {
		t.Helper()
		id := testdaten.LegeAusleiheMitFristAn(t, pool, schueler, titel, frist)
		tag, err := time.ParseInLocation("2006-01-02 15:04", ausgegeben, schulzeit.Zone())
		if err != nil {
			t.Fatalf("Ausgabedatum %q: %v", ausgegeben, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE ausleihen SET ausgeliehen_am = $1 WHERE id = $2`, tag, id); err != nil {
			t.Fatalf("Ausgabedatum setzen: %v", err)
		}
		return id
	}

	anna := testdaten.LegeSchuelerAn(t, pool, "T-1", "Anna", "9H1")
	annaNeu := ausleihe(anna, "LMF Mathe 9 Anna", "2027-06-23 10:00", laufend)
	annaAlt := ausleihe(anna, "LMF Mathe 8 Anna", "2026-09-01 10:00", laufend)
	ben := testdaten.LegeSchuelerAn(t, pool, "T-2", "Ben", "9H2")
	benNeu := ausleihe(ben, "LMF Deutsch 9 Ben", "2027-06-23 10:00", laufend)
	carla := testdaten.LegeSchuelerAn(t, pool, "T-3", "Carla", "7G2")
	carlaNeu := ausleihe(carla, "LMF Bio 7 Carla", "2027-06-23 10:00", laufend)
	carlaAlt := ausleihe(carla, "LMF Bio 6 Carla", "2026-09-01 10:00", laufend)
	carlaRoman := ausleihe(carla, "Roman Carla", "2027-06-23 10:00", ende("2027-07-14"))
	dora := testdaten.LegeSchuelerAn(t, pool, "T-4", "Dora", "8R1")
	doraNeu := ausleihe(dora, "LMF Englisch 8 Dora", "2027-06-23 10:00", laufend)
	if _, err := pool.Exec(ctx, `UPDATE schueler SET is_manually_blocked = true, block_reason = 'Test' WHERE id = $1`, dora); err != nil {
		t.Fatal(err)
	}
	emil := testdaten.LegeSchuelerAn(t, pool, "T-5", "Emil", "")
	emilNeu := ausleihe(emil, "LMF Atlas Emil", "2027-06-23 10:00", laufend)
	frieda := testdaten.LegeSchuelerAn(t, pool, "T-6", "Frieda", "9H1")
	friedaVorher := ausleihe(frieda, "LMF Physik 9 Frieda", "2027-06-22 10:00", laufend)
	gesa := testdaten.LegeLeserAn(t, pool, "T-7", "Gesa", "Test", "lehrkraft")
	if _, err := pool.Exec(ctx, `UPDATE leser SET klasse = '9H1' WHERE id = $1`, gesa); err != nil {
		t.Fatal(err)
	}
	gesaNeu := ausleihe(gesa, "LMF Chemie 9 Gesa", "2027-06-23 10:00", laufend)

	erwarte := func(schritt, was, id string, soll time.Time) {
		t.Helper()
		if ist := testdaten.LadeFrist(t, pool, id); !ist.Equal(soll) {
			t.Errorf("%s, %s: Frist %v, erwartet %v", schritt, was, ist.In(schulzeit.Zone()), soll.In(schulzeit.Zone()))
		}
	}

	// Ende Donnerstag 24.06.2027 4. Stunde, sechs Stunden je Tag: 9H2 liegt auf dem Ende, 9H1
	// sieben Zeilen davor am Mittwoch 23.06. in der 3. Stunde. Der Tausch beginnt am 23.06.
	zeilen := `{"klassen":["9H1"]},{"vermerk":"1"},{"vermerk":"2"},{"vermerk":"3"},{"vermerk":"4"},{"vermerk":"5"},{"vermerk":"6"},{"klassen":["9H2"]}`
	if rec := lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-24","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[`+zeilen+`]}`); rec.Code != http.StatusOK {
		t.Fatalf("Entwurf speichern: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Entwurf", "Annas neues Schulbuch", annaNeu, laufend)

	rec := lmfPlanAufruf(t, srv, http.MethodPost, "rueckgabe", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("veröffentlichen: %d %s", rec.Code, rec.Body.String())
	}
	var antwort lmfPlanSpeicherAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if len(antwort.Zeilen) != 8 || antwort.Zeilen[0].Datum != "2027-06-23" || antwort.Zeilen[7].Datum != "2027-06-24" {
		t.Fatalf("Aufbau: Zeilen des Plans %+v", antwort.Zeilen)
	}
	// Drei Bücher aus dem Tausch, dazu Annas altes und Friedas vor dem Tausch geholtes Buch
	// auf den Termin der 9H1.
	if antwort.FristenAngepasst != 5 {
		t.Errorf("Veröffentlichen meldet %d angepasste Fristen, erwartet 5", antwort.FristenAngepasst)
	}
	schritt := "veröffentlicht am zweiten Tag des Tauschs"
	erwarte(schritt, "Annas neues Schulbuch (am Termintag geholt)", annaNeu, folgend)
	erwarte(schritt, "Annas altes Schulbuch", annaAlt, ende("2027-06-23"))
	erwarte(schritt, "Bens neues Schulbuch (vor seinem Termin geholt)", benNeu, folgend)
	erwarte(schritt, "Carlas neues Schulbuch (Klasse nicht im Plan)", carlaNeu, folgend)
	erwarte(schritt, "Carlas altes Schulbuch (Klasse nicht im Plan)", carlaAlt, laufend)
	erwarte(schritt, "Carlas Roman (kein Lernmittel)", carlaRoman, ende("2027-07-14"))
	erwarte(schritt, "Doras Schulbuch (von Hand gesperrt)", doraNeu, laufend)
	erwarte(schritt, "Emils Schulbuch (ohne Klasse)", emilNeu, laufend)
	erwarte(schritt, "Friedas Schulbuch vom Tag vor dem Tausch", friedaVorher, ende("2027-06-23"))
	erwarte(schritt, "Gesas Schulbuch (Lehrkraft mit Klasse)", gesaNeu, laufend)

	// Der Beginn rückt nach vorn: Drei Zeilen vorn legen die erste auf Dienstag 22.06., 6. Stunde.
	// Friedas Buch vom 22.06. gehört jetzt zum Tausch.
	if rec := lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-24","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"vermerk":"a"},{"vermerk":"b"},{"vermerk":"c"},`+zeilen+`]}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"datum":"2027-06-22"`) {
		t.Fatalf("Beginn nach vorn: %d %s", rec.Code, rec.Body.String())
	}
	schritt = "Beginn nach vorn"
	erwarte(schritt, "Friedas Schulbuch vom 22.06.", friedaVorher, folgend)
	erwarte(schritt, "Annas altes Schulbuch", annaAlt, ende("2027-06-23"))

	// Verwerfen: Die alten Bücher kehren zum Stichtag zurück; die Bücher aus dem Tausch bleiben
	// beim nächsten Schuljahr, sie wurden für das nächste Schuljahr ausgegeben.
	if rec := lmfPlanAufruf(t, srv, http.MethodDelete, "rueckgabe", ""); rec.Code != http.StatusOK {
		t.Fatalf("verwerfen: %d %s", rec.Code, rec.Body.String())
	}
	schritt = "verworfen"
	erwarte(schritt, "Annas altes Schulbuch", annaAlt, laufend)
	erwarte(schritt, "Annas neues Schulbuch", annaNeu, folgend)
	erwarte(schritt, "Carlas neues Schulbuch", carlaNeu, folgend)
}
