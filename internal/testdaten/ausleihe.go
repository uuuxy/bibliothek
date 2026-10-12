package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/pkg/schulzeit"
)

// SeedAusleihe legt Titel, Exemplar und eine offene Ausleihe mit der Frist an und liefert die
// Kennung der Ausleihe.
func SeedAusleihe(t *testing.T, pool *pgxpool.Pool, schuelerID, titel string, frist time.Time) string {
	t.Helper()
	ctx := context.Background()
	tid := TitelMitMeldebestand(t, pool, titel, 1)
	eid := Exemplar(t, pool, tid, "BC-"+titel, true, "")
	var aid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, $3) RETURNING id`,
		eid, schuelerID, frist).Scan(&aid); err != nil {
		t.Fatalf("Ausleihe für %q anlegen: %v", titel, err)
	}
	return aid
}

// SeedLeserAusleihe bucht eine Ausleihe von vor zwei Tagen: innerhalb jedes Zeitfensters des
// Monitors. Die Zeit kommt aus der Datenbank, weil die Fenster dort gerechnet werden.
func SeedLeserAusleihe(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '21 days')`,
		exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

// SeedOffeneAusleihe hängt dem Schüler eine noch nicht zurückgegebene Ausleihe an.
func SeedOffeneAusleihe(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := TitelMitMeldebestand(t, pool, "Titel-"+barcodePraefix, 1)
	exID := Exemplar(t, pool, titelID, "EX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, now() + interval '14 days')`, exID, schuelerID); err != nil {
		t.Fatalf("offene Ausleihe anlegen: %v", err)
	}
}

// FristVon liest die Rückgabefrist einer Ausleihe.
func FristVon(t *testing.T, pool *pgxpool.Pool, ausleiheID string) time.Time {
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

// MahnState liest Mahnstufe und letztes Mahndatum einer Ausleihe.
func MahnState(t *testing.T, pool *pgxpool.Pool, ausleiheID string) (int, *time.Time) {
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
