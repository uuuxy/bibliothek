package api

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Befund F4: Lehrkräfte lagen als Schein-Schüler (klasse='lehrer') in der
// Schüler-Tabelle — mit dem LUSD-Abgleich als tickender Löschfalle. Migration
// 072 zieht sie auf Personal-Konten um; pruefeKlassenname sperrt beide
// Eingabetüren. Hier werden Umzug UND Sperren am echten Postgres bewiesen.

// TestScheinSchuelerUmzug072 spielt die Migrations-Logik an echten Daten durch
// (der DO-Block ist wiederholbar — ein erneuter Lauf über Testdaten ist exakt
// das Produktionsverhalten).
func TestScheinSchuelerUmzug072(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)

	// Lehrkraft als Schein-Schüler mit Ausweis und offener Ausleihe.
	var lehrerID, titelID, exemplarID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('F4-KARTE-1', 'Frieda', 'Fachlehrerin', 'lehrer', 2031) RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Schein-Schüler: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, medientyp) VALUES ('F4-Handbuch', 'Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'F4-EX-1') RETURNING id`, titelID).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, CURRENT_TIMESTAMP + interval '21 days')`, exemplarID, lehrerID); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	// Zweite Schein-Zeile MIT Schadensfall — der Fall wechselt die Spalte und
	// zieht mit auf das Konto um (schadensfaelle kennt beide Seiten).
	var mitSchadenID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('F4-KARTE-2', 'Stefan', 'Schadenslehrer', 'Lehrer', 2031) RETURNING id`).Scan(&mitSchadenID); err != nil {
		t.Fatalf("Schein-Schüler 2: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (schueler_id, exemplar_id, beschreibung)
		VALUES ($1, $2, 'Wasserschaden')`, mitSchadenID, exemplarID); err != nil {
		t.Fatalf("Schadensfall: %v", err)
	}

	sql, err := os.ReadFile("../migrations/072_schein_schueler_zu_benutzer.sql")
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("Umzug ausführen: %v", err)
	}

	// Frieda ist jetzt ein Personal-Konto mit ihrer Karte …
	var benutzerID, rolle, email string
	var aktiv bool
	if err := pool.QueryRow(ctx, `
		SELECT id, rolle::text, email, aktiv FROM benutzer WHERE barcode_id = 'F4-KARTE-1'`).
		Scan(&benutzerID, &rolle, &email, &aktiv); err != nil {
		t.Fatalf("umgezogenes Konto fehlt: %v", err)
	}
	if rolle != "kollegium" || !aktiv || !strings.HasSuffix(email, "@lehrer-umzug.invalid") {
		t.Errorf("Konto falsch: rolle=%s aktiv=%v email=%s", rolle, aktiv, email)
	}
	// … ihre Ausleihe zeigt auf das Konto, die Schein-Zeile ist weg.
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE ausleiher_benutzer_id = $1 AND rueckgabe_am IS NULL`, benutzerID).Scan(&n); err != nil || n != 1 {
		t.Errorf("Ausleihe nicht umgezogen (n=%d, err=%v)", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schueler WHERE id = $1`, lehrerID).Scan(&n); err != nil || n != 0 {
		t.Errorf("Schein-Zeile besteht noch (n=%d)", n)
	}
	// Auch der Schadensfall wechselte die Seite: er hängt jetzt am Konto.
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM schadensfaelle sf
		JOIN benutzer b ON sf.benutzer_id = b.id
		WHERE b.barcode_id = 'F4-KARTE-2' AND sf.schueler_id IS NULL`).Scan(&n); err != nil || n != 1 {
		t.Errorf("Schadensfall nicht umgezogen (n=%d, err=%v)", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schueler WHERE id = $1`, mitSchadenID).Scan(&n); err != nil || n != 0 {
		t.Errorf("Schein-Zeile mit Schadensfall besteht noch (n=%d)", n)
	}
}

// TestKlasseLehrerIstGesperrt beweist beide Eingabetüren (Zwei-Türen-Regel):
// POST /api/schueler und PATCH-baueSchuelerUpdate lehnen den Spezialwert ab,
// bevor eine Datenbank berührt wird.
func TestKlasseLehrerIstGesperrt(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest("POST", "/api/schueler",
		strings.NewReader(`{"vorname":"Neu","nachname":"Lehrkraft","klasse":" Lehrer ","barcode_id":"X-1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.CreateStudentHandler()(w, req)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Benutzerverwaltung") {
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
