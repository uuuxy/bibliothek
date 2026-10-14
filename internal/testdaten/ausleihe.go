package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/pkg/schulzeit"
)

// LegeAusleiheMitFristAn legt Titel, Exemplar und eine offene Ausleihe mit der Frist an und
// liefert die Kennung der Ausleihe.
func LegeAusleiheMitFristAn(t *testing.T, pool *pgxpool.Pool, schuelerID, titel string, frist time.Time) string {
	t.Helper()
	ctx := context.Background()
	titelID := LegeTitelAn(t, pool, titel, 1)
	exemplarID := LegeExemplarAn(t, pool, titelID, "BC-"+titel, true, "")
	var ausleiheID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, $3) RETURNING id`,
		exemplarID, schuelerID, frist).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe für %q anlegen: %v", titel, err)
	}
	return ausleiheID
}

// LegeAusleiheVorZweiTagenAn legt am Exemplar eine Ausleihe von vor zwei Tagen an: Sie liegt in
// jedem Zeitfenster des Monitors. Die Zeit kommt aus der Datenbank, weil die Fenster dort
// gerechnet werden.
func LegeAusleiheVorZweiTagenAn(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '21 days')`,
		exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

// LegeOffeneAusleiheAn legt Titel und Exemplar an und hängt dem Schüler eine Ausleihe an, deren
// Frist in 14 Tagen endet.
func LegeOffeneAusleiheAn(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := LegeTitelAn(t, pool, "Titel-"+barcodePraefix, 1)
	exemplarID := LegeExemplarAn(t, pool, titelID, "EX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, now() + interval '14 days')`, exemplarID, schuelerID); err != nil {
		t.Fatalf("offene Ausleihe anlegen: %v", err)
	}
}

// LadeFrist liest die Rückgabefrist einer Ausleihe.
func LadeFrist(t *testing.T, pool *pgxpool.Pool, ausleiheID string) time.Time {
	t.Helper()
	var frist time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT rueckgabe_frist FROM ausleihen WHERE id = $1`, ausleiheID).Scan(&frist); err != nil {
		t.Fatalf("Frist lesen: %v", err)
	}
	return frist.UTC()
}

// FristEnde ist das Tagesende eines Kalendertags in der Zeitzone der Schule.
func FristEnde(j int, m time.Month, d int) time.Time {
	return time.Date(j, m, d, 23, 59, 59, 0, schulzeit.Zone())
}

// LadeMahnstufe liest Mahnstufe und letztes Mahndatum einer Ausleihe.
func LadeMahnstufe(t *testing.T, pool *pgxpool.Pool, ausleiheID string) (int, *time.Time) {
	t.Helper()
	var stufe int
	var datum *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT mahnstufe, letztes_mahndatum FROM ausleihen WHERE id = $1`, ausleiheID).
		Scan(&stufe, &datum); err != nil {
		t.Fatalf("Mahn-Status lesen: %v", err)
	}
	return stufe, datum
}

// LegeSchuelerMitAusleiheAn legt einen Schüler mit genau einer offenen Ausleihe an, deren Frist
// vor drei Tagen ablief.
func LegeSchuelerMitAusleiheAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	id := LegeSchuelerAn(t, pool, barcode, vorname, klasse)
	LegeAusleiheMitFristAn(t, pool, id, "Buch "+vorname, time.Now().AddDate(0, 0, -3))
	return id
}
