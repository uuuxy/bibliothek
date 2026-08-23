package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Ein defektes Gerät bleibt defekt, wenn jemand nur seine Stammdaten bearbeitet.
//
// Der Bearbeiten-Dialog (GeraeteVerwaltung.svelte) hat fünf Felder — modellname,
// barcode_id, seriennummer, zubehoer, zustand_notiz — und schickt genau diese. Das
// Defekt-Kennzeichen liegt auf einem eigenen Knopf und ist im Formular NICHT enthalten.
//
// Im Handler stand dazu `istAusleihbar := true`, wenn das Feld fehlt. Ein fehlendes
// Feld war damit kein "unverändert", sondern ein Wert: Wer bei einem als defekt
// markierten Gerät einen Tippfehler im Zubehör korrigierte, gab es damit still wieder
// zur Ausleihe frei. Auf dem Bildschirm stand "Gerät gespeichert".
func TestGeraetBearbeiten_HebtDefektNichtAuf(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewGeraeteRepository(pool)

	if _, err := pool.Exec(ctx, `DELETE FROM geraete WHERE barcode_id = 'G-DEFEKT'`); err != nil {
		t.Fatalf("aufräumen: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO geraete (modellname, barcode_id, seriennummer, zubehoer, ist_ausleihbar)
		VALUES ('Tablet 10', 'G-DEFEKT', 'SN-4711', 'Ladekabel', false)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}

	// GENAU der Rumpf des Bearbeiten-Dialogs: fünf Felder, kein ist_ausleihbar.
	// Die Seriennummer wird MIT GEÄNDERT: Das Formular bietet das Feld an, der Handler
	// liess es bis zum 23.08.2026 unter den Tisch fallen — die Korrektur einer falschen
	// Seriennummer war folgenlos, mit "Gerät gespeichert" daneben.
	body := `{"modellname":"Tablet 10","barcode_id":"G-DEFEKT","seriennummer":"SN-9999",
	          "zubehoer":"Ladekabel und Hülle","zustand_notiz":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.UpdateGeraetHandler(repo)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT gab %d: %s", rec.Code, rec.Body.String())
	}

	var ausleihbar bool
	var zubehoer, seriennummer string
	if err := pool.QueryRow(ctx,
		`SELECT ist_ausleihbar, coalesce(zubehoer,''), coalesce(seriennummer,'') FROM geraete WHERE id = $1`, id).
		Scan(&ausleihbar, &zubehoer, &seriennummer); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if zubehoer != "Ladekabel und Hülle" {
		t.Errorf("die gewollte Änderung kam nicht an: zubehoer = %q", zubehoer)
	}
	if seriennummer != "SN-9999" {
		t.Errorf("die Seriennummer wurde verworfen: %q — das Formular bietet das Feld an", seriennummer)
	}
	if ausleihbar {
		t.Error("das defekte Gerät ist wieder ausleihbar — ein fehlendes Feld wurde als " +
			"Wert gelesen statt als \"unverändert\"")
	}
}

// Und der Defekt-Knopf schaltet weiterhin, ohne die Stammdaten anzufassen: Er schickt
// die Seriennummer nicht mit, und nil heisst dort "nicht angefasst" — nicht "leeren".
func TestGeraetDefektKnopf_LaesstStammdatenStehen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewGeraeteRepository(pool)

	if _, err := pool.Exec(ctx, `DELETE FROM geraete WHERE barcode_id = 'G-KNOPF'`); err != nil {
		t.Fatalf("aufräumen: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO geraete (modellname, barcode_id, seriennummer, zubehoer, ist_ausleihbar)
		VALUES ('Tablet 11', 'G-KNOPF', 'SN-0815', 'Ladekabel', true)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}

	body := `{"modellname":"Tablet 11","zubehoer":"Ladekabel","zustand_notiz":"","ist_ausleihbar":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.UpdateGeraetHandler(repo)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT gab %d: %s", rec.Code, rec.Body.String())
	}

	var ausleihbar bool
	var seriennummer string
	if err := pool.QueryRow(ctx,
		`SELECT ist_ausleihbar, coalesce(seriennummer,'') FROM geraete WHERE id = $1`, id).
		Scan(&ausleihbar, &seriennummer); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if ausleihbar {
		t.Error("der Defekt-Knopf hat nicht geschaltet")
	}
	if seriennummer != "SN-0815" {
		t.Errorf("der Defekt-Knopf hat die Seriennummer verändert: %q", seriennummer)
	}
}
