package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

// Befund F4: Lehrkräfte lagen als Schein-Schüler (klasse='lehrer') in der
// Schüler-Tabelle — mit dem LUSD-Abgleich als tickender Löschfalle. Migration
// 072 zieht sie auf Personal-Konten um; pruefeKlassenname sperrt beide
// Eingabetüren. Hier werden Umzug UND Sperren am echten Postgres bewiesen.

// Der Nachspiel-Test der Migration 072 ist am 16.09.2026 entfallen.
//
// Er ließ 072 gegen das HEUTIGE Schema laufen und prüfte das Ergebnis von damals: Die
// Lehrkraft zieht aus der Schülertabelle in ein Konto, ihre Ausleihe hängt danach an
// `ausleihen.ausleiher_benutzer_id`. Beides gibt es nicht mehr — Migration 125 hat die
// Spalte und die Ausweisnummer am Konto entfernt, und sie hat das Ziel von 072 bewusst
// umgekehrt: Eine Lehrkraft steht wieder in derselben Tabelle wie die Schüler, nur mit
// einer anderen ART. Die Migration selbst bleibt unverändert im Ordner (eingespielte
// Migrationen werden nicht angefasst); sie ist auf einer gewachsenen Anlage längst
// gelaufen, lange vor 125.
//
// Was von F4 bleibt und weiter geprüft wird, steht darunter: Eine Lehrkraft darf nicht
// über die Klasse „Lehrer" in die Schülerdatei zurückkommen — an BEIDEN Eingabetüren.

// TestKlasseLehrerIstGesperrt beweist beide Eingabetüren (Zwei-Türen-Regel):
// POST /api/schueler und PATCH-baueSchuelerUpdate lehnen den Spezialwert ab,
// bevor eine Datenbank berührt wird.
func TestKlasseLehrerIstGesperrt(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest("POST", "/api/schueler",
		strings.NewReader(`{"vorname":"Neu","nachname":"Lehrkraft","klasse":" Lehrer ","barcode_id":"X-1","geburtsdatum":"1980-01-01"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.CreateStudentHandler()(w, req)
	// Der Wegweiser nennt die Rolle, unter der eine Lehrkraft angelegt wird (Wortlaut:
	// student_klasse_regel_test.go). „Benutzer & Rechte" steht im JSON als &.
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Rolle Kollegium") {
		t.Errorf("POST mit klasse='Lehrer' muss 400 mit Wegweiser liefern, got %d: %s", w.Code, w.Body.String())
	}

	klasse := "LEHRER"
	w2 := httptest.NewRecorder()
	if _, ok := baueSchuelerUpdate(w2, &patchStudentRequest{Klasse: &klasse}); ok {
		t.Error("PATCH mit klasse='LEHRER' muss abgelehnt werden")
	}
	if w2.Code != 400 {
		t.Errorf("PATCH-Ablehnung muss 400 sein, got %d", w2.Code)
	}
}

// TestZwillingeAusDerLusd (Befund F8): Zwei ECHTE Schüler mit gleichem Namen
// und Geburtstag existieren nebeneinander, wenn beide aus der LUSD kommen —
// während die Handeingabe eines Namens-Datums-Doppels hart abgewiesen bleibt.
func TestZwillingeAusDerLusd(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)

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
