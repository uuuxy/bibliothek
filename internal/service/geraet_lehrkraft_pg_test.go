package service

// Wer ein Gerät bekommt: ein aktiver Leser — Schüler wie Kollegium.
//
// Eine unbekannte Kennung des Ausleihers wird abgewiesen, bevor sie als
// Fremdschlüssel-Verletzung und damit als 500 „…weil verknüpfte Daten bestehen" endet. Wer als
// Lehrkraft gilt, entscheidet die Art des Lesers, nicht die Personenart des Kontos: Auch ein
// Admin ohne dieses Feld bekommt sein Gerät.
//
// Ein Kollege wird nie gesperrt, auch am Gerät nicht (pruefeAusleihSperren). Den gesperrten
// Schüler hält geraet_rueckgabe_sperre_pg_test.go.
//
// Wer kein Schüler ist, bekommt das Gerät als Dauerleihe (ausleihen.ist_handapparat), ein
// Schüler nicht: dieselbe Regel wie beim Buch. Ohne das Merkmal stünde die Lehrkraft nach
// Ablauf der Frist mit einer überfälligen Ausleihe in der Leserdatei, mit ihm zählte das Gerät
// eines Schülers dort nie als überfällig (LeserAbfragen.ladeListe).

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

	// vonHandGesperrt: eine Sperre von Hand, wie sie noch an einem Kollegen stehen kann.
	legeLeserAn := func(kuerzel, art string, vonHandGesperrt bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, is_manually_blocked, block_reason)
			VALUES ('Geraete', $1, $2, $3, CASE WHEN $3 THEN 'Testsperre' END) RETURNING id
		`, kuerzel, art, vonHandGesperrt).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", kuerzel, err)
		}
		return id
	}

	legeSchuelerAn := func() string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Geraete', 'Schueler', '05A', 2031) RETURNING id
		`, "GL-S-"+suffix).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
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

	svc := NewGeraeteDienst(pool, repository.NewLeserAbfragen(pool), repository.NewAusleihAbfragen(pool),
		repository.NewProtokollAbfragen(pool))

	faelle := []struct {
		name, kennung string
		verliehen     bool
		dauerleihe    bool
		fehler        error
	}{
		{"Lehrkraft bekommt das Gerät", legeLeserAn("LK", "lehrkraft", false), true, true, nil},
		{"LiV bekommt das Gerät", legeLeserAn("LIV", "liv", false), true, true, nil},
		{"unbekannte Kennung", "3f2504e0-4f89-11d3-9a0c-0305e82c3301", false, false, ErrNichtGefunden},
		{"gesperrte Lehrkraft bekommt das Gerät — ein Kollege wird nie gesperrt", legeLeserAn("GES", "lehrkraft", true), true, true, nil},
		{"Schüler bekommt das Gerät mit Frist", legeSchuelerAn(), true, false, nil},
	}
	for i, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			barcode := fmt.Sprintf("G-LK-%d-%s", i, suffix)
			if _, err := pool.Exec(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1)`, barcode); err != nil {
				t.Fatalf("Gerät anlegen: %v", err)
			}
			kennung := f.kennung
			res, err := svc.verarbeiteScan(ctx, barcode, &kennung, true, false, mitarbeiterID)

			var offen int
			if qerr := pool.QueryRow(ctx, `
				SELECT count(*) FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id
				WHERE g.barcode_id = $1 AND a.rueckgabe_am IS NULL
			`, barcode).Scan(&offen); qerr != nil {
				t.Fatalf("offene Ausleihen zählen: %v", qerr)
			}

			if f.verliehen {
				if err != nil || res == nil || res.Art != "ausleihe" || offen != 1 {
					t.Fatalf("erwartet Ausleihe, war err=%v, offen=%d", err, offen)
				}
				var dauerleihe bool
				if qerr := pool.QueryRow(ctx, `
					SELECT a.ist_handapparat FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id
					WHERE g.barcode_id = $1 AND a.rueckgabe_am IS NULL
				`, barcode).Scan(&dauerleihe); qerr != nil {
					t.Fatalf("Merkmal der Ausleihe lesen: %v", qerr)
				}
				if dauerleihe != f.dauerleihe {
					t.Errorf("Dauerleihe = %v, erwartet %v", dauerleihe, f.dauerleihe)
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
