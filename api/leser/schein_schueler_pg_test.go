package leser

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Eine Lehrkraft als Schein-Schüler (klasse='lehrer') in der Schülerdatei wäre eine tickende
// Löschfalle: Der LUSD-Abgleich kennt sie nicht und machte sie zum Abgänger. Migration 072 zog
// solche Zeilen in Konten um; Migration 125 hat das bewusst umgekehrt: Eine Lehrkraft steht
// wieder in derselben Tabelle wie die Schüler, nur mit einer anderen Art. Die Migration selbst
// bleibt unverändert im Ordner (eingespielte Migrationen werden nicht angefasst).
//
// Was von 072 bleibt und hier geprüft wird: Eine Lehrkraft darf nicht über die Klasse „Lehrer"
// in die Schülerdatei zurückkommen, an beiden Eingabetüren (pruefeKlassenname).

// TestKlasseLehrerIstGesperrt beweist beide Eingabetüren (Zwei-Türen-Regel):
// POST /api/schueler und PATCH-baueLeserAenderung lehnen den Spezialwert ab,
// bevor eine Datenbank berührt wird.
func TestKlasseLehrerIstGesperrt(t *testing.T) {
	s := &Tueren{Basis: &tuer.Basis{}}

	req := httptest.NewRequest("POST", "/api/schueler",
		strings.NewReader(`{"vorname":"Neu","nachname":"Lehrkraft","klasse":" Lehrer ","barcode_id":"X-1","geburtsdatum":"1980-01-01"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.LeserAnlegen()(w, req)
	// Der Wegweiser nennt die Rolle, unter der eine Lehrkraft angelegt wird (Wortlaut:
	// leser_klasse_regel_test.go). „Benutzer & Rechte" steht im JSON als &.
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Rolle Kollegium") {
		t.Errorf("POST mit klasse='Lehrer': Status %d, erwartet 400 mit Wegweiser: %s", w.Code, w.Body.String())
	}

	klasse := "LEHRER"
	w2 := httptest.NewRecorder()
	if _, ok := baueLeserAenderung(w2, &leserAendernAnfrage{Klasse: &klasse}); ok {
		t.Error("PATCH mit klasse='LEHRER' muss abgelehnt werden")
	}
	if w2.Code != 400 {
		t.Errorf("PATCH-Ablehnung: Status %d, erwartet 400", w2.Code)
	}
}

// TestZwillingeAusDerLUSD: Zwei echte Schüler mit gleichem Namen
// und Geburtstag existieren nebeneinander, wenn beide aus der LUSD kommen —
// während die Handeingabe eines Namens-Datums-Doppels hart abgewiesen bleibt.
func TestZwillingeAusDerLUSD(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)

	for i, lusd := range []string{"LUSD-F8-1", "LUSD-F8-2"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum, lusd_id)
			VALUES ($1, 'Kim', 'Zwilling', '5a', 2033, '2014-03-03', $2)`,
			"F8-K-"+lusd, lusd); err != nil {
			t.Fatalf("LUSD-Zwilling %d: %v", i+1, err)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
		VALUES ('F8-HAND-1', 'Mira', 'Handeingabe', '6b', 2032, '2013-07-07')`); err != nil {
		t.Fatalf("Handeingabe 1: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
		VALUES ('F8-HAND-2', 'Mira', 'Handeingabe', '6b', 2032, '2013-07-07')`); err == nil {
		t.Fatal("Handeingabe-Doppel wurde angenommen — der Schutz für manuelle Anlagen ist weg")
	} else if !strings.Contains(err.Error(), "unique_schueler_name_gebdatum") {
		t.Fatalf("falscher Fehler: %v", err)
	}
}
