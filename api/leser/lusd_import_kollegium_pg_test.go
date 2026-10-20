package leser

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der LUSD-Import durch die Vordertür — und zwar der schreibende Lauf, mit Kollegium im
// Bestand.
//
// Die bestehenden Tests prüfen die Einzelteile: die Bestandsabfrage kennt nur Schüler
// (TestLUSDBestandKenntNurSchueler), die Sicht macht einen Kollegen für die Schreibwege
// unsichtbar (TestKollegeIstFuerDenAbgleichUnsichtbar), der CHECK verbietet den Zustand
// (TestKollegeKannAuchDirektKeinAbgaengerWerden). Keiner von ihnen fährt den Import
// selbst: Upload → Parser → Klassifizierung → Anwenden → Commit.
//
// Genau diese Lücke ist die Bugklasse „Unit grün, Funktion unerreichbar". Seit dem
// Zusammenlegen der Leser (Migrationen 123 bis 125) liegt das Kollegium in derselben Tabelle,
// die der Abgleich bearbeitet — die Frage „klappt der Import noch?" ist deshalb keine
// Frage an die Einzelteile, sondern an den ganzen Weg.
//
// Was dieser Test beweist und was nicht (durch Rückbau gemessen):
//
//   - Rot wird er, wenn der Import nicht mehr schreibt (Tür auf uebernehmen=false
//     zurückgebaut: Klasse bleibt 07A, der neue Schüler fehlt ganz). Er misst also
//     wirklich die Kette und nicht sich selbst.
//   - Rot wird er nicht, wenn man allein die Schranke `art = 'schueler'` aus der
//     Bestandsabfrage nimmt. Dann stehen die Kollegen zwar im Bestand, aber der
//     Namensmodus macht nur aus einem schon einmal bestätigten Schüler einen Abgänger
//     (`lusd_bestaetigt_am`, Migration 084) — und ein Kollege ist nie von einem Export
//     bestätigt worden. Diese zweite Schranke fängt den Fehler ab.
//
// Darum steht die erste Schranke trotzdem nicht ohne Wächter da: Sie hat ihren eigenen
// Test (TestLUSDBestandKenntNurSchueler). Vier Schranken hintereinander, jede mit eigenem
// Nachweis — Bestandsabfrage, Bestätigungsstempel, die Sicht beim Schreiben und der CHECK
// in der Datenbank.
func TestLUSDImport_VordertuerMitKollegiumImBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	admin := &auth.Claims{KontoID: "00000000-0000-0000-0000-00000000a130", Rolle: auth.RolleAdmin}

	// ── Bestand vor dem Import ───────────────────────────────────────────────────
	// Zwei Schüler: einer steht im Export (mit neuer Klasse), einer nicht (Abgänger).
	//
	// lusd_bestaetigt_am ist gesetzt: Beide sind schon einmal in einem Export
	// aufgetaucht. Ohne diesen Stempel macht der Namensmodus aus einem fehlenden Schüler
	// keinen Abgänger, sondern meldet ihn als „nicht im Export" (Migration 084) — eine
	// nie bestätigte Handanlage soll nicht an der ersten Liste zugrunde gehen. Der Test
	// braucht den Abgänger-Zweig, also stellt er die Lage her, in der es ihn gibt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum, lusd_bestaetigt_am)
		VALUES ('S-IMP-1', 'Mia', 'Muster',   '7a', 2030, DATE '2012-03-04', NOW()),
		       ('S-IMP-2', 'Tim', 'Tschuess', '9b', 2027, DATE '2010-05-06', NOW())`); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Eine Lehrkraft ohne Konto — so entsteht sie über „Neuen Leser anlegen".
	lehrkraftID := testdaten.LegeLeserAn(t, pool, "L-IMP-1", "Petra", "Kollegin", "lehrkraft")
	// Und ein Kollegiumskonto — seine Leserzeile legt der Trigger an (Migration 125).
	var kontoLeserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ulf', 'Unberuehrt', 'ulf@import.invalid', 'kollegium', true)
		RETURNING leser_id`).Scan(&kontoLeserID); err != nil {
		t.Fatalf("Kollegiumskonto anlegen: %v", err)
	}

	// ── Der Export: Mia wechselt die Klasse, ein Kind kommt neu dazu, Tim fehlt ──
	// Namen, die es im Bestand als Kollegen gibt, stehen bewusst nicht darin: Durch die
	// LUSD kommen keine Lehrkräfte.
	//
	// Ohne LUSD-ID-Spalte — das ist der Namensmodus, und er ist der gefährliche: Im
	// ID-Modus hat ein Kollege gar keine Kennung und fiele schon deshalb nur unter
	// „nicht abgleichbar". Über den Namen dagegen ist er vollwertig abgleichbar, und was
	// abgleichbar ist und im Export fehlt, ist ein Abgänger. Genau diese Datei bekommt
	// die Schule: die LANIS-Klassenliste mit Name, Klasse, Geburtsdatum.
	csv := "vorname,nachname,klasse,geburtsdatum\n" +
		"Mia,Muster,8a,2012-03-04\n" +
		"Nora,Neuling,5c,2015-07-08\n"

	antwort := func(t *testing.T, pfad string, felder map[string]string) (*httptest.ResponseRecorder, *lusd.Vorschau) {
		t.Helper()
		var koerper bytes.Buffer
		mw := multipart.NewWriter(&koerper)
		teil, err := mw.CreateFormFile("csvFile", "lusd_export.csv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := teil.Write([]byte(csv)); err != nil {
			t.Fatal(err)
		}
		for k, v := range felder {
			if err := mw.WriteField(k, v); err != nil {
				t.Fatal(err)
			}
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, pfad, &koerper)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, admin))
		rec := httptest.NewRecorder()
		if pfad == "/api/lusd/import" {
			srv.LUSDImport().ServeHTTP(rec, req)
		} else {
			srv.LUSDVorschau().ServeHTTP(rec, req)
		}
		var res lusd.Vorschau
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
			}
		}
		return rec, &res
	}

	// ── 1. Vorschau: sagt sie das Richtige voraus? ───────────────────────────────
	rec, vorschau := antwort(t, "/api/lusd/preview", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Vorschau: Status %d, %s", rec.Code, rec.Body.String())
	}
	if len(vorschau.Neuzugaenge) != 1 || vorschau.Neuzugaenge[0].Nachname != "Neuling" {
		t.Errorf("Vorschau, Neue: %+v", vorschau.Neuzugaenge)
	}
	if len(vorschau.Klassenwechsel) != 1 || vorschau.Klassenwechsel[0].NeueKlasse != "8a" {
		t.Errorf("Vorschau, Klassenwechsel: %+v", vorschau.Klassenwechsel)
	}
	// Genau ein Abgänger — Tim. Stünden die Kollegen im Bestand des Abgleichs, wären es
	// drei, und zwei Menschen verlören nach der Karenz ihren Namen.
	if len(vorschau.Abgaenger) != 1 || vorschau.Abgaenger[0].Nachname != "Tschuess" {
		t.Fatalf("Vorschau, Abgänger: %+v — Kollegium gehört NICHT dazu", vorschau.Abgaenger)
	}

	// ── 2. Der schreibende Lauf ──────────────────────────────────────────────────
	// confirm_graduates, weil bei drei Schülern jeder Abgänger über der Massenschwelle
	// liegt: Der Schutz vor dem versehentlichen Jahrgangsabgang gilt auch hier.
	rec, ergebnis := antwort(t, "/api/lusd/import", map[string]string{"confirm_graduates": "true"})
	if rec.Code != http.StatusOK {
		t.Fatalf("Import: Status %d, %s", rec.Code, rec.Body.String())
	}
	if len(ergebnis.Neuzugaenge) != 1 || len(ergebnis.Klassenwechsel) != 1 || len(ergebnis.Abgaenger) != 1 {
		t.Errorf("Import-Zahlen: neu=%d wechsel=%d abgänger=%d",
			len(ergebnis.Neuzugaenge), len(ergebnis.Klassenwechsel), len(ergebnis.Abgaenger))
	}

	// ── 3. Was wirklich in der Datenbank steht ───────────────────────────────────
	lies := func(t *testing.T, where string, arg any) (vorname, nachname, klasse, art string, abgaenger, gesperrt bool, anonym *string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT vorname, nachname, coalesce(klasse, ''), art, ist_abgaenger, ist_gesperrt, anonymized_at::text
			FROM leser WHERE `+where, arg).Scan(&vorname, &nachname, &klasse, &art, &abgaenger, &gesperrt, &anonym); err != nil {
			t.Fatalf("Zeile lesen (%s): %v", where, err)
		}
		return
	}

	// Der Schüler im Export: neue Klasse, aktiv.
	if _, _, klasse, _, abg, _, _ := lies(t, "nachname = $1", "Muster"); klasse != "08A" || abg {
		t.Errorf("Mia nach dem Import: Klasse %q, Abgänger %v — erwartet 08A und aktiv", klasse, abg)
	}
	// Der neue Schüler ist angelegt.
	if _, _, klasse, art, _, _, _ := lies(t, "nachname = $1", "Neuling"); klasse != "05C" || art != "schueler" {
		t.Errorf("Nora nach dem Import: Klasse %q, Art %q", klasse, art)
	}
	// Der fehlende Schüler ist Abgänger und gesperrt — der Import tut also wirklich etwas.
	if _, _, _, _, abg, gesperrt, _ := lies(t, "nachname = $1", "Tschuess"); !abg || !gesperrt {
		t.Errorf("Tim nach dem Import: Abgänger %v, gesperrt %v — erwartet beides", abg, gesperrt)
	}

	// Und das ist der Punkt: Beide Kollegen stehen unverändert da.
	for _, fall := range []struct{ name, id string }{
		{"Lehrkraft ohne Konto", lehrkraftID},
		{"Kollegiumskonto", kontoLeserID},
	} {
		vorname, nachname, klasse, art, abg, gesperrt, anonym := lies(t, "id = $1", fall.id)
		if abg || gesperrt || anonym != nil || klasse != "" || art == "schueler" {
			t.Errorf("%s (%s %s) wurde vom Import angefasst: Art=%q Klasse=%q Abgänger=%v gesperrt=%v anonymisiert=%v",
				fall.name, vorname, nachname, art, klasse, abg, gesperrt, anonym)
		}
	}

	// Gegenprobe gegen einen stillen Nulllauf: Es standen wirklich fünf Leser da.
	var gesamt int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM leser WHERE deleted_at IS NULL`).Scan(&gesamt); err != nil {
		t.Fatal(err)
	}
	if gesamt != 5 {
		t.Errorf("%d Leser nach dem Import, erwartet 5 (3 Schüler + 2 Kollegen)", gesamt)
	}
}
