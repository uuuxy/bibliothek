package pgtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Test ändert, was schema.sql anlegt, und legt Neues an. Die Prüfung meldet er vor dem
// Pool an: t.Cleanup ruft in umgekehrter Reihenfolge, die Prüfung läuft also nach dem
// Zurücksetzen, das Pool angemeldet hat.
func TestPool_SetztAmEndeDesTestsZurueck(t *testing.T) {
	var p *pgxpool.Pool
	t.Cleanup(func() {
		if p == nil {
			return // übersprungen, ohne Testdatenbank
		}
		if n := ZaehleZeilen(t, p, `SELECT count(*) FROM systematik_kategorien`); n != 0 {
			t.Errorf("nach dem Test %d Zeilen in systematik_kategorien, erwartet 0", n)
		}
		if n := ZaehleZeilen(t, p, `SELECT count(*) FROM role_permissions`); n != 0 {
			t.Errorf("nach dem Test %d Zeilen in role_permissions, erwartet 0", n)
		}
		if n := ZaehleZeilen(t, p, `SELECT count(*) FROM system_einstellungen
			WHERE (schluessel = 'max_ausleihen_schueler' AND wert = '5') OR (schluessel = 'schule_ort' AND wert = '')`); n != 2 {
			t.Errorf("nach dem Test %d der beiden Einstellungen aus schema.sql unverändert, erwartet 2", n)
		}
		// schema.sql legt die Zeile mit id 1 an, ohne die Folge zu rufen.
		var letzter int64
		var gerufen bool
		if err := p.QueryRow(context.Background(),
			`SELECT last_value, is_called FROM mail_settings_config_id_seq`).Scan(&letzter, &gerufen); err != nil {
			t.Fatalf("Folge lesen: %v", err)
		}
		if letzter != 1 || gerufen {
			t.Errorf("Folge der Mail-Einstellungen nach dem Test bei %d (gerufen %v), erwartet 1 (ungerufen)", letzter, gerufen)
		}
	})
	p = Pool(t)
	for _, sql := range []string{
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('ZR', 'Zurücksetzen')`,
		`INSERT INTO role_permissions (role, permission, allowed) VALUES ('MITARBEITER', 'merge_students', true)`,
		`UPDATE system_einstellungen SET wert = '9' WHERE schluessel = 'max_ausleihen_schueler'`,
		`DELETE FROM system_einstellungen WHERE schluessel = 'schule_ort'`,
		`SELECT nextval('mail_settings_config_id_seq')`,
	} {
		if _, err := p.Exec(context.Background(), sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

// Ein Untertest teilt, was sein Test angelegt hat: Holt er den Pool, setzt sein Ende nichts
// zurück.
func TestPool_UntertestSetztNichtZurueck(t *testing.T) {
	p := Pool(t)
	if _, err := p.Exec(context.Background(),
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('ZR', 'Zurücksetzen')`); err != nil {
		t.Fatalf("Kategorie anlegen: %v", err)
	}
	t.Run("holt den Pool", func(t *testing.T) { Pool(t) })
	if n := ZaehleZeilen(t, p, `SELECT count(*) FROM systematik_kategorien`); n != 1 {
		t.Errorf("nach dem Untertest %d Zeilen in systematik_kategorien, erwartet 1", n)
	}
}
