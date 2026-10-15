package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Spalte der Verfügbarkeit in der Exemplarliste: `ist_verfuegbar` ist ein bool unter fünf
// anderen Feldern, und ein gekipptes bool sieht in der Antwort aus wie ein richtiges. Wer die
// Bedingung umschreibt, etwa `(SELECT COUNT(*) …) = 0` zu `NOT EXISTS (…)`, merkt einen Fehler
// nur an diesem Test.
//
// Warum ein Test an der Datenbank: Die ganze Aussage steckt in SQL. Ein Mock prüfte die
// nachgespielte Antwort, nicht die Frage, ob Postgres bei einer zurückgegebenen Ausleihe
// wirklich wieder „verfügbar" sagt.
//
// Der dritte Fall ist der eigentliche Grund für diese Datei: Ein Exemplar, das einmal
// ausgeliehen war, muss nach der Rückgabe wieder verfügbar sein. Wer die Bedingung
// `a.rueckgabe_am IS NULL` beim nächsten Umbau verliert, baut eine Bibliothek, in der
// jedes je ausgeliehene Buch für immer vergriffen ist — und die ersten beiden Fälle
// blieben dabei grün.

// exemplarZeilen ruft den Live-Pfad der Exemplarliste auf
// (GET /api/buecher/titel/{id}/exemplare) und liefert die Antwortzeilen.
func exemplarZeilen(t *testing.T, srv *Tueren, titelID string) []struct {
	BarcodeID     string `json:"barcode_id"`
	IstVerfuegbar bool   `json:"ist_verfuegbar"`
} {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/exemplare", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()

	srv.TitelExemplare(repository.NewBescheidAbfragen(srv.DB.Pool))(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Exemplarliste: Status %d, Body %s", rec.Code, rec.Body.String())
	}

	var zeilen []struct {
		BarcodeID     string `json:"barcode_id"`
		IstVerfuegbar bool   `json:"ist_verfuegbar"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &zeilen); err != nil {
		t.Fatalf("Antwort lesen: %v (Body: %s)", err, rec.Body.String())
	}
	return zeilen
}

// verfuegbarkeitNach sucht die Zeile eines Barcodes heraus. Fehlt sie, ist das ein
// Fehlschlag und keine stille Auslassung — sonst wäre der Test auch grün, wenn die
// Abfrage gar nichts geliefert hat.
func verfuegbarkeitNach(t *testing.T, srv *Tueren, titelID, barcode string) bool {
	t.Helper()
	for _, z := range exemplarZeilen(t, srv, titelID) {
		if z.BarcodeID == barcode {
			return z.IstVerfuegbar
		}
	}
	t.Fatalf("Exemplar %q kam in der Liste gar nicht vor", barcode)
	return false
}

func TestTitelExemplare_VerfuegbarkeitFolgtDerOffenenAusleihe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ctx := context.Background()

	titel := testdaten.LegeTitelAn(t, pool, "Verfuegbarkeit", 1)
	schueler := testdaten.LegeSchuelerAn(t, pool, "S-VERF-1", "Verf", "7a")

	testdaten.LegeExemplarAn(t, pool, titel, "BC-FREI", true, "") // nie verliehen
	verliehen := testdaten.LegeExemplarAn(t, pool, titel, "BC-VERLIEHEN", true, "")
	zurueck := testdaten.LegeExemplarAn(t, pool, titel, "BC-ZURUECK", true, "")

	// Alle Zeitstempel entstehen in SQL, keiner in Go.
	//
	// Der erste Anlauf setzte `rueckgabe_am` auf ein Go-seitiges time.Now(), während
	// `ausgeliehen_am` aus der DEFAULT-Klausel der Spalte kommt (CURRENT_TIMESTAMP, also
	// die Uhr der Datenbank). Ob `check_return_date` (rueckgabe_am >= ausgeliehen_am) dann
	// hält, entscheidet die Differenz zweier Uhren: Lokal lief der Test grün, in CI fiel er
	// zuverlässig — dort laufen Go und Postgres auf derselben Uhr, und der in Go gebildete
	// Wert ist um die Laufzeit der Anweisung älter als der beim Einfügen erzeugte.
	//
	// Ein Test, dessen Ergebnis an einer Uhrendifferenz hängt, prüft nicht die Sache,
	// für die er geschrieben wurde.
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		 VALUES ($1, $2, NOW() - INTERVAL '3 days', NOW() + INTERVAL '11 days')`,
		verliehen, schueler); err != nil {
		t.Fatalf("offene Ausleihe anlegen: %v", err)
	}

	// Abgeschlossene Ausleihe: dasselbe Exemplar war verliehen und ist zurück.
	// Die Reihenfolge ausgeliehen → zurück ist hier per Konstruktion wahr.
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		 VALUES ($1, $2, NOW() - INTERVAL '30 days', NOW() - INTERVAL '16 days', NOW() - INTERVAL '2 days')`,
		zurueck, schueler); err != nil {
		t.Fatalf("zurückgegebene Ausleihe anlegen: %v", err)
	}

	// Gegenprobe gegen einen stillen Nulllauf.
	if anzahl := len(exemplarZeilen(t, srv, titel)); anzahl != 3 {
		t.Fatalf("Die Liste lieferte %d Exemplare statt 3 — der Test misst sonst nichts", anzahl)
	}

	if !verfuegbarkeitNach(t, srv, titel, "BC-FREI") {
		t.Error("Ein nie verliehenes Exemplar muss verfügbar sein")
	}
	if verfuegbarkeitNach(t, srv, titel, "BC-VERLIEHEN") {
		t.Error("Ein Exemplar mit offener Ausleihe darf nicht als verfügbar gelten — " +
			"an der Theke wäre es zugesagt und stünde nicht im Regal")
	}
	if !verfuegbarkeitNach(t, srv, titel, "BC-ZURUECK") {
		t.Error("Ein zurückgegebenes Exemplar muss WIEDER verfügbar sein — sonst ist jedes " +
			"je ausgeliehene Buch dauerhaft vergriffen (Bedingung rueckgabe_am IS NULL verloren)")
	}
}

// Ein Titel ohne Exemplare liefert eine leere Liste und keinen Fehler — die
// Exemplarkarte im Katalog ruft dieselbe Route auch für frisch angelegte Titel auf.
func TestTitelExemplare_TitelOhneExemplare(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	titel := testdaten.LegeTitelAn(t, pool, "Ohne Exemplare", 0)

	if zeilen := exemplarZeilen(t, srv, titel); len(zeilen) != 0 {
		t.Errorf("erwartet: leere Liste, bekommen: %d Zeilen", len(zeilen))
	}
}
