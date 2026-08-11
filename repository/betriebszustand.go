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
