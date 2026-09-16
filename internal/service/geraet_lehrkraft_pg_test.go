package service

// Wer ein Gerät bekommt: ein aktiver LESER — Schüler wie Kollegium (Migration 125).
//
// Bis zum 13.09.2026 übernahm die Geräte-Ausleihe die Kennung des Ausleihers ungeprüft:
// Eine unbekannte endete als Fremdschlüssel-Verletzung und damit als 500 „…da verknüpfte
// Daten existieren" (am Stack nachgestellt). Bis zum 16.09.2026 entschied danach die
// Personenart des KONTOS, wer als Lehrkraft gilt — ein Admin ohne dieses Feld bekam kein
// Gerät, obwohl er vor der Theke stand. Beides ist hier abgesichert.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

func TestGeraetAusleiheNurAnAktivenLeser(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	legeLeserAn := func(kuerzel, art string, gesperrt bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, ist_gesperrt, block_reason)
			VALUES ('Geraete', $1, $2, $3, CASE WHEN $3 THEN 'Testsperre' END) RETURNING id
		`, kuerzel, art, gesperrt).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", kuerzel, err)
		}
		return id
	}

	var mitarbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Geraete', 'Theke', $1, 'mitarbeiter', true) RETURNING id
	`, "geraete-theke-"+suffix+"@schule.invalid").Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}

	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id LIKE 'G-LK-%-' || $1)`,
			`DELETE FROM geraete WHERE barcode_id LIKE 'G-LK-%-' || $1`,
			`DELETE FROM benutzer WHERE email LIKE '%-' || $1 || '@schule.invalid'`,
			`DELETE FROM leser WHERE vorname = 'Geraete' AND $1 = $1`,
		} {
			if _, err := pool.Exec(ctx, sql, suffix); err != nil {
				t.Errorf("Aufräumen (%s): %v", sql, err)
			}
		}
	})

	svc := NewDeviceService(pool, repository.NewStudentRepository(pool), repository.NewLoanRepository(pool),
		repository.NewAuditRepository(pool))

	faelle := []struct {
		name, kennung string
		verliehen     bool
		fehler        error
	}{
		{"Lehrkraft bekommt das Gerät", legeLeserAn("LK", "lehrkraft", false), true, nil},
		{"LiV bekommt das Gerät", legeLeserAn("LIV", "liv", false), true, nil},
		{"unbekannte Kennung", "3f2504e0-4f89-11d3-9a0c-0305e82c3301", false, ErrNotFound},
		{"gesperrter Leser bekommt nichts", legeLeserAn("GES", "lehrkraft", true), false, ErrBlocked},
	}
	for i, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			barcode := fmt.Sprintf("G-LK-%d-%s", i, suffix)
			if _, err := pool.Exec(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1)`, barcode); err != nil {
				t.Fatalf("Gerät anlegen: %v", err)
			}
			kennung := f.kennung
			res, err := svc.HandleDeviceAction(ctx, barcode, &kennung, true, mitarbeiterID)

			var offen int
			if qerr := pool.QueryRow(ctx, `
				SELECT count(*) FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id
				WHERE g.barcode_id = $1 AND a.rueckgabe_am IS NULL
			`, barcode).Scan(&offen); qerr != nil {
				t.Fatalf("offene Ausleihen zählen: %v", qerr)
			}

			if f.verliehen {
				if err != nil || res == nil || res.Type != "ausleihe" || offen != 1 {
					t.Fatalf("erwartet Ausleihe, war err=%v, offen=%d", err, offen)
				}
				return
			}
			if !errors.Is(err, f.fehler) {
				t.Errorf("erwartet %v (Bedienfehler statt 500), war %v", f.fehler, err)
			}
			if offen != 0 {
				t.Errorf("Gerät wurde trotzdem verliehen (%d offen)", offen)
			}
		})
	}
}
