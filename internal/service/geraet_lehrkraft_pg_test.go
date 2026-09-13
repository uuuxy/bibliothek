package service

// Wer ein Gerät als Lehrkraft ausleiht. Die Buch-Ausleihe nimmt nur ein aktives Profil mit
// Rolle kollegium (resolveTeacherBorrower), und nur solche Ausweise erkennt die Theke als
// Lehrerausweis (GetLehrerByBarcode). Die Geräte-Ausleihe übernahm active_teacher_id bis
// zum 13.09.2026 ungeprüft: Eine unbekannte Kennung endete als Fremdschlüssel-Verletzung
// und damit als 500 „…da verknüpfte Daten existieren" (am Stack nachgestellt), und ein
// Profil ohne Lehrer-Rolle oder ein deaktiviertes bekam das Gerät.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

func TestGeraetAusleiheNurAnAktiveLehrkraft(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	legeBenutzerAn := func(kuerzel, rolle string, aktiv bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Geraete', $2, $3, $4, $5) RETURNING id
		`, "GL-"+kuerzel+"-"+suffix, kuerzel, "geraete-"+kuerzel+"-"+suffix+"@schule.invalid", rolle, aktiv).Scan(&id); err != nil {
			t.Fatalf("Benutzer %s anlegen: %v", kuerzel, err)
		}
		return id
	}
	mitarbeiterID := legeBenutzerAn("MA", "mitarbeiter", true)
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id LIKE 'G-LK-%-' || $1)`,
			`DELETE FROM geraete WHERE barcode_id LIKE 'G-LK-%-' || $1`,
			`DELETE FROM benutzer WHERE barcode_id LIKE 'GL-%-' || $1`,
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
	}{
		{"aktive Lehrkraft bekommt das Gerät", legeBenutzerAn("LK", "kollegium", true), true},
		{"unbekannte Kennung", "3f2504e0-4f89-11d3-9a0c-0305e82c3301", false},
		{"deaktivierte Lehrkraft", legeBenutzerAn("ALT", "kollegium", false), false},
		{"Mitarbeiterin ist keine Lehrkraft", mitarbeiterID, false},
	}
	for i, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			barcode := fmt.Sprintf("G-LK-%d-%s", i, suffix)
			if _, err := pool.Exec(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1)`, barcode); err != nil {
				t.Fatalf("Gerät anlegen: %v", err)
			}
			kennung := f.kennung
			res, err := svc.HandleDeviceAction(ctx, barcode, nil, &kennung, true, mitarbeiterID)

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
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("erwartet ErrNotFound (Bedienfehler statt 500), war %v", err)
			}
			if offen != 0 {
				t.Errorf("Gerät wurde trotzdem verliehen (%d offen)", offen)
			}
		})
	}
}
