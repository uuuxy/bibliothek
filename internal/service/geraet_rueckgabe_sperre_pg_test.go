package service

// Rückgabe trotz Sperre. Bis zum 13.09.2026 prüfte HandleDeviceAction die Sperren des
// Geräts (ist_ausleihbar = false, etwa nach der Defekt-Meldung) und des aktiven Schülers,
// BEVOR feststand, ob ausgeliehen oder zurückgegeben wird. Ein verliehenes Gerät, das
// danach als defekt gemeldet wurde, ließ sich an der Theke nicht mehr zurücknehmen: 403
// „Gerät ist aktuell gesperrt", die Ausleihe blieb offen (am Stack nachgestellt). Die
// Sperren gelten der Ausleihe, nicht der Rückgabe. Echtes Postgres über den echten
// Service: Die Rückgabe sperrt die Ausleihe per FOR UPDATE in einer Transaktion.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

func TestGeraetRueckgabeTrotzSperre(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var mitarbeiterID, lehrkraftID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Geraete', 'Theke', $1, 'mitarbeiter', true) RETURNING id
	`, "geraete-theke-"+suffix+"@schule.invalid").Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	// Die Lehrkraft ist ein LESER (Migration 125) — das Gerät hängt an ihrer Leserzeile,
	// nicht an ihrem Konto.
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Geraete', 'Lehrkraft', $1, 'kollegium', true) RETURNING leser_id
	`, "geraete-lehrkraft-"+suffix+"@schule.invalid").Scan(&lehrkraftID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id LIKE 'G-SP-%-' || $1)`,
			`DELETE FROM geraete WHERE barcode_id LIKE 'G-SP-%-' || $1`,
			`DELETE FROM schueler WHERE barcode_id LIKE 'GR-S-%-' || $1`,
			`DELETE FROM benutzer WHERE email LIKE '%-' || $1 || '@schule.invalid'`,
			`DELETE FROM leser WHERE vorname = 'Geraete' AND $1 = $1`,
		} {
			if _, err := pool.Exec(ctx, sql, suffix); err != nil {
				t.Errorf("Aufräumen (%s): %v", sql, err)
			}
		}
	})

	studentRepo := repository.NewStudentRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	svc := NewDeviceService(pool, studentRepo, loanRepo, auditRepo)

	legeGeraetAn := func(t *testing.T, name string) string {
		t.Helper()
		barcode := "G-SP-" + name + "-" + suffix
		if _, err := pool.Exec(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ($1, $2)`,
			"Tablet "+name, barcode); err != nil {
			t.Fatalf("Gerät anlegen: %v", err)
		}
		return barcode
	}
	legeSchuelerAn := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Geraete', $2, '05A', 2031) RETURNING id
		`, "GR-S-"+name+"-"+suffix, name).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		return id
	}
	offeneAusleihen := func(t *testing.T, barcode string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id
			WHERE g.barcode_id = $1 AND a.rueckgabe_am IS NULL
		`, barcode).Scan(&n); err != nil {
			t.Fatalf("offene Ausleihen zählen: %v", err)
		}
		return n
	}
	sperreGeraet := func(t *testing.T, barcode string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE geraete SET ist_ausleihbar = false WHERE barcode_id = $1`, barcode); err != nil {
			t.Fatalf("Gerät als defekt sperren: %v", err)
		}
	}
	sperreSchueler := func(t *testing.T, schuelerID string) {
		t.Helper()
		// chk_schueler_block_reason: jede Sperre braucht einen Grund.
		if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test: Sperre' WHERE id = $1`, schuelerID); err != nil {
			t.Fatalf("Schüler sperren: %v", err)
		}
	}

	t.Run("defektes Gerät kommt von der Lehrkraft zurück", func(t *testing.T) {
		barcode := legeGeraetAn(t, "defekt")
		if _, err := svc.HandleDeviceAction(ctx, barcode, &lehrkraftID, true, mitarbeiterID); err != nil {
			t.Fatalf("Ausleihe: %v", err)
		}
		sperreGeraet(t, barcode)

		res, err := svc.HandleDeviceAction(ctx, barcode, nil, true, mitarbeiterID)
		if err != nil {
			t.Fatalf("Rückgabe eines verliehenen, danach defekt gemeldeten Geräts abgewiesen: %v", err)
		}
		if res.Type != "rueckgabe" || offeneAusleihen(t, barcode) != 0 {
			t.Fatalf("erwartet Rückgabe ohne offene Ausleihe, war Typ %q, offen %d", res.Type, offeneAusleihen(t, barcode))
		}
	})

	t.Run("gesperrter Schüler gibt sein Gerät zurück", func(t *testing.T) {
		barcode := legeGeraetAn(t, "schueler")
		schuelerID := legeSchuelerAn(t, "Zurueck")
		if _, err := svc.HandleDeviceAction(ctx, barcode, &schuelerID, true, mitarbeiterID); err != nil {
			t.Fatalf("Ausleihe: %v", err)
		}
		sperreSchueler(t, schuelerID)

		res, err := svc.HandleDeviceAction(ctx, barcode, &schuelerID, true, mitarbeiterID)
		if err != nil {
			t.Fatalf("Rückgabe durch den gesperrten Schüler abgewiesen: %v", err)
		}
		if res.Type != "rueckgabe" || offeneAusleihen(t, barcode) != 0 {
			t.Fatalf("erwartet Rückgabe ohne offene Ausleihe, war Typ %q, offen %d", res.Type, offeneAusleihen(t, barcode))
		}
	})

	t.Run("Gegenprobe: freies defektes Gerät wird nicht verliehen", func(t *testing.T) {
		barcode := legeGeraetAn(t, "frei")
		sperreGeraet(t, barcode)

		if _, err := svc.HandleDeviceAction(ctx, barcode, &lehrkraftID, true, mitarbeiterID); !errors.Is(err, ErrBlocked) {
			t.Fatalf("erwartet ErrBlocked, war %v", err)
		}
		if n := offeneAusleihen(t, barcode); n != 0 {
			t.Fatalf("gesperrtes Gerät wurde trotzdem verliehen (%d offen)", n)
		}
	})

	t.Run("Gegenprobe: gesperrter Schüler bekommt kein Gerät", func(t *testing.T) {
		barcode := legeGeraetAn(t, "sperre")
		schuelerID := legeSchuelerAn(t, "Gesperrt")
		sperreSchueler(t, schuelerID)

		if _, err := svc.HandleDeviceAction(ctx, barcode, &schuelerID, true, mitarbeiterID); !errors.Is(err, ErrBlocked) {
			t.Fatalf("erwartet ErrBlocked, war %v", err)
		}
		if n := offeneAusleihen(t, barcode); n != 0 {
			t.Fatalf("gesperrter Schüler bekam trotzdem ein Gerät (%d offen)", n)
		}
	})
}
