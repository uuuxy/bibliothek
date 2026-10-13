package repository_test

import (
	"context"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ein Teilindex gilt nur, wenn die Bedingung der Abfrage seine Bedingung einschließt. Wer
// repository.EtikettOffenBedingung lockert, macht idx_buecher_exemplare_etikett_offen
// unbrauchbar, ohne Fehlermeldung und ohne rote Prüfung, nur langsamer; das fiele erst auf,
// wenn der Bestand gewachsen ist. Der Test liest deshalb den Ausführungsplan und verlangt den
// Namen des Index darin.
//
// enable_seqscan=off macht ihn unabhängig von der Größe der Test-Datenbank: Bei drei Zeilen
// nähme Postgres sonst immer den vollen Durchlauf. Erzwungen wird nur der Vorzug; passt der
// Index nicht zur Abfrage, holt auch enable_seqscan=off ihn nicht in den Plan.
func TestEtikettenOffenNutztDenTeilindex(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Verbindung holen: %v", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SET enable_seqscan = off"); err != nil {
		t.Fatalf("enable_seqscan: %v", err)
	}

	plan := erklaere(t, conn, `
		SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = '00000000-0000-0000-0000-000000000001' AND `+repository.EtikettOffenBedingung)

	if !strings.Contains(plan, "idx_buecher_exemplare_etikett_offen") {
		t.Fatalf("Der Teil-Index taucht im Plan nicht auf — EtikettOffenBedingung passt nicht "+
			"mehr zu seiner WHERE-Bedingung (Migration 064).\nPlan:\n%s", plan)
	}
}

// Die Bestellhistorie darf für ihre 200 neuesten Zeilen nicht die ganze Tabelle lesen.
func TestBestellhistorieNutztDenDatumsindex(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Verbindung holen: %v", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SET enable_seqscan = off"); err != nil {
		t.Fatalf("enable_seqscan: %v", err)
	}

	plan := erklaere(t, conn, `
		SELECT b.id FROM bestellungen_verlauf b
		ORDER BY b.bestelldatum DESC
		LIMIT 200`)

	if !strings.Contains(plan, "idx_bestellungen_verlauf_datum") {
		t.Fatalf("Der Datumsindex taucht im Plan nicht auf — die Sortierrichtung der Abfrage "+
			"passt nicht mehr zum Index (Migration 064).\nPlan:\n%s", plan)
	}
}

// erklaere liefert den Ausführungsplan als Text, ohne die Abfrage auszuführen: Es geht um die
// Wahl des Zugriffspfads, nicht um Laufzeiten auf einer leeren Test-Datenbank.
func erklaere(t *testing.T, conn *pgxpool.Conn, sql string) string {
	t.Helper()
	rows, err := conn.Query(context.Background(), "EXPLAIN "+sql)
	if err != nil {
		t.Fatalf("EXPLAIN: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var zeile string
		if err := rows.Scan(&zeile); err != nil {
			t.Fatalf("Plan lesen: %v", err)
		}
		plan.WriteString(zeile)
		plan.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Plan lesen: %v", err)
	}
	return plan.String()
}
