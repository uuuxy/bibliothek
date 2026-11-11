package leser

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Klasse, Abgangsjahr und LUSD-ID gehören nur zu einem Schüler. Über die Akte an einem Kollegen
// gesetzt, stand eine Klasse still in der Zeile, und eine LUSD-ID lief in
// chk_leser_nur_schueler_werden_abgaenger und kam als 500 zurück. Die Anlage weist die Klasse
// schon mit demselben Satz ab.
func TestLeserAendern_SchuelerfelderNurBeimSchueler(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	kollege := testdaten.LegeLeserAn(t, pool, "", "Kollege", "Schuelerfeld", "lehrkraft")
	schueler := testdaten.LegeSchuelerAn(t, pool, "SF-0001", "Kind", "07G1")

	for _, fall := range []struct{ name, rumpf, meldung string }{
		{"Klasse", `{"klasse":"05F1"}`, meldungNurSchuelerHatKlasse},
		{"LUSD-ID", `{"lusd_id":"12345"}`, "Nur ein Schüler hat ein Abgangsjahr und eine LUSD-ID."},
		{"Abgangsjahr", `{"abgaenger_jahr":2030}`, "Nur ein Schüler hat ein Abgangsjahr und eine LUSD-ID."},
		{"Klasse mit Namen", `{"klasse":"05F1","nachname":"Neu"}`, meldungNurSchuelerHatKlasse},
	} {
		t.Run(fall.name, func(t *testing.T) {
			rec := aendereLeserUeberTuer(t, srv, kollege, fall.rumpf)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), fall.meldung) {
				t.Fatalf("PATCH %s an einer Lehrkraft = %d %s, erwartet 400 mit %q", fall.rumpf, rec.Code, rec.Body.String(), fall.meldung)
			}
		})
	}
	var klasse, lusdID, nachname string
	var jahr *int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(klasse, ''), COALESCE(lusd_id, ''), abgaenger_jahr, nachname FROM leser WHERE id = $1`,
		kollege).Scan(&klasse, &lusdID, &jahr, &nachname); err != nil {
		t.Fatalf("Lehrkraft lesen: %v", err)
	}
	if klasse != "" || lusdID != "" || jahr != nil || nachname != "Schuelerfeld" {
		t.Errorf("Lehrkraft geändert: Klasse %q, LUSD-ID %q, Abgangsjahr %v, Nachname %q", klasse, lusdID, jahr, nachname)
	}

	// Gegenprobe: Ohne die Felder bleibt die Akte des Kollegen änderbar, und beim Schüler gehen
	// die Felder durch.
	if rec := aendereLeserUeberTuer(t, srv, kollege, `{"nachname":"Geaendert"}`); rec.Code != http.StatusOK {
		t.Errorf("Nachname einer Lehrkraft = %d %s, erwartet 200", rec.Code, rec.Body.String())
	}
	if rec := aendereLeserUeberTuer(t, srv, schueler, `{"klasse":"08G1","lusd_id":"12345","abgaenger_jahr":2032}`); rec.Code != http.StatusOK {
		t.Errorf("Klasse, LUSD-ID und Abgangsjahr eines Schülers = %d %s, erwartet 200", rec.Code, rec.Body.String())
	}
	if rec := aendereLeserUeberTuer(t, srv, "00000000-0000-0000-0000-000000000000", `{"klasse":"08G1"}`); rec.Code != http.StatusNotFound {
		t.Errorf("Klasse an einem unbekannten Leser = %d %s, erwartet 404", rec.Code, rec.Body.String())
	}
}
