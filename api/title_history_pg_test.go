package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
)

// Die Titel-Historie ist der sichtbarste Konsument der Lesehistorie-Befristung
// (jobs/cron_dsgvo_lesehistorie.go): Nach der Trennung steht dort KEIN Schüler mehr —
// und vor allem kein falscher. Bis 22.08.2026 machte COALESCE(s.klasse, 'Lehrer') aus
// jeder Ausleihe ohne Schüler eine Lehrer-Ausleihe; mit der Befristung wäre das die Regel
// gewesen. Erwartung: getrennte Schüler-Ausleihe → "Anonym" ohne Klasse; echte
// Lehrer-Ausleihe → Klasse "Lehrer"; zugeordnete Schüler-Ausleihe → Name und Klasse.
func TestTitleHistory_GetrennteAusleiheIstAnonymNichtLehrer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID, exemplarID, schuelerID, lehrerID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Historie-Titel') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-HIST-1') RETURNING id`, titelID).Scan(&exemplarID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-HIST', 'Mia', 'Muster', '7a', 2030) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('L-HIST', 'Lena', 'Lehr', 'lehr-hist@example.org', 'kollegium', true) RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatal(err)
	}
	// Drei abgeschlossene Vorgänge desselben Exemplars, in dieser Reihenfolge (neueste zuerst
	// in der Antwort): zugeordnet (vor 3 Tagen), getrennt (vor 200 Tagen), Lehrer (vor 400).
	for _, v := range []struct {
		schueler, lehrer *string
		tage             int
	}{
		{&schuelerID, nil, 3},
		{nil, nil, 200},
		{nil, &lehrerID, 400},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, $3, now() - make_interval(days => $4 + 20), now() - make_interval(days => $4 + 1), now() - make_interval(days => $4))`,
			exemplarID, v.schueler, v.lehrer, v.tage); err != nil {
			t.Fatalf("Ausleihe (%d Tage): %v", v.tage, err)
		}
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/historie", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.handleGetTitleHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	var got []TitleHistory
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("erwartet 3 Einträge, bekam %d: %+v", len(got), got)
	}
	if got[0].Vorname != "Mia" || got[0].Klasse != "7a" {
		t.Errorf("zugeordnete Ausleihe: %+v", got[0])
	}
	if got[1].Vorname != "Anonym" || got[1].Nachname != "" || got[1].Klasse != "" {
		t.Errorf("getrennte Ausleihe muss anonym OHNE Klasse sein (nicht 'Lehrer'): %+v", got[1])
	}
	if got[2].Vorname != "Lena" || got[2].Klasse != "Lehrer" {
		t.Errorf("Lehrer-Ausleihe: %+v", got[2])
	}
}
