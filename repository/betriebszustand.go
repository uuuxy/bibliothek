package repository

// betriebszustand.go — Abfragen, die die Selbstprüfung (api/betriebsbereitschaft.go)
// braucht, um den Zustand der Anlage zu beurteilen.
//
// Eigene Datei statt einer Zeile im Handler: Handler lesen und schreiben über repository/,
// dort steht jede Regel genau einmal. Das ist im Projekt eine geprüfte Invariante
// (api/schichtung_test.go) — und sie hat mich hier zu Recht erwischt.

import (
	"context"
	"time"

	"bibliothek/db"
)

// demoBarcodePraefix ist das Präfix, das scripts/seed_demo.sql jedem angelegten Schüler
// gibt. Es ist zugleich der Schlüssel, an dem der CLEANUP-Block desselben Skripts die
// Demo-Daten wieder entfernt — beide Seiten müssen dasselbe meinen.
const demoBarcodePraefix = "DEMO-S-%"

// BetriebszustandRepository beantwortet Fragen über den Zustand des Bestandes.
type BetriebszustandRepository struct {
	pool db.PgxPoolIface
}

// NewBetriebszustandRepository bindet das Repository an einen Pool.
func NewBetriebszustandRepository(pool db.PgxPoolIface) *BetriebszustandRepository {
	return &BetriebszustandRepository{pool: pool}
}

// ZaehleDemoSchueler liefert die Anzahl der Datensätze aus scripts/seed_demo.sql.
//
// Eigenes kurzes Zeitlimit: Die Selbstprüfung ist eine Auskunft, kein Arbeitsschritt — sie
// darf keine Seite blockieren. Bleibt die Antwort aus, meldet der Aufrufer lieber „keine
// Demo-Daten gefunden" als gar nichts; der Bereich ist ohnehin nur eine Warnung.
func (r *BetriebszustandRepository) ZaehleDemoSchueler(ctx context.Context) (int, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	var anzahl int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE barcode_id LIKE $1`, demoBarcodePraefix).Scan(&anzahl)
	return anzahl, err
}

// AdminKonto ist ein aktives Konto mit Vollzugriff — Empfänger der Alarm-Mails
// und Gegenstand des Sichtbarkeits-Befunds der Selbstprüfung.
type AdminKonto struct {
	Name  string
	Email string
}

// AktiveAdmins liefert alle aktiven Admin-Konten (Name + Adresse) — EINE Quelle
// für den Alarm-Versand UND den Betriebsbereitschafts-Befund. Der Vorfall vom
// 16.08.2026 (Alarm-Mail an ein dem Betreiber unbekanntes Admin-Konto) hat
// gezeigt: Die Admin-Liste muss sichtbar sein, nicht nur benutzt werden.
func (r *BetriebszustandRepository) AktiveAdmins(ctx context.Context) ([]AdminKonto, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := r.pool.Query(ctx, `
		SELECT btrim(vorname || ' ' || nachname), email
		FROM benutzer WHERE rolle = 'admin' AND aktiv = true AND email <> ''
		ORDER BY erstellt_am`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	konten := []AdminKonto{}
	for rows.Next() {
		var k AdminKonto
		if err := rows.Scan(&k.Name, &k.Email); err != nil {
			return nil, err
		}
		konten = append(konten, k)
	}
	return konten, rows.Err()
}

// LadeRollenRechte liefert die Live-Rechte (role_permissions) als Rolle→Recht→erlaubt.
// Für den Vorgabe-Abgleich der Selbstprüfung; gleiche Zeitlimit-Begründung wie oben.
func (r *BetriebszustandRepository) LadeRollenRechte(ctx context.Context) (map[string]map[string]bool, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := r.pool.Query(ctx, `SELECT role, permission, allowed FROM role_permissions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	live := map[string]map[string]bool{}
	for rows.Next() {
		var rolle, recht string
		var erlaubt bool
		if err := rows.Scan(&rolle, &recht, &erlaubt); err != nil {
			return nil, err
		}
		if live[rolle] == nil {
			live[rolle] = map[string]bool{}
		}
		live[rolle][recht] = erlaubt
	}
	return live, rows.Err()
}
