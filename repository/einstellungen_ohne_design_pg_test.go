package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// merkeAbfrage reicht jede Abfrage an den Pool weiter und merkt sich ihren Text.
type merkeAbfrage struct {
	pool *pgxpool.Pool
	sql  string
}

func (m *merkeAbfrage) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.sql = sql
	return m.pool.Query(ctx, sql, args...) //nolint:rowserrcheck // reicht die Zeilen weiter, der Aufrufer prüft sie
}

// Das Ausweis-Design liegt in system_einstellungen und darf mit seinen Bildern mehrere
// Megabyte groß sein. LadeEinstellungen läuft bei jeder Ausleihe und holte die Zeile mit, nur
// um sie zu verwerfen. Geprüft wird an der Abfrage, die die Funktion wirklich schickt: Sie
// liefert die Einstellung, aber nicht das Design.
func TestLadeEinstellungen_LiestDasAusweisDesignNichtMit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	design := `{"front":{"elements":[{"content":"data:image/png;base64,` + strings.Repeat("A", 2<<20) + `"}]}}`
	if err := SchreibeAusweisLayout(ctx, pool, design); err != nil {
		t.Fatalf("SchreibeAusweisLayout: %v", err)
	}
	schule := "Schule der Probe"
	if err := NewEinstellungsAbfragen(pool).Schreibe(ctx, &EinstellungenPatch{SchuleName: &schule}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel = $1`, AusweisLayoutSchluessel); err != nil {
			t.Errorf("Design entfernen: %v", err)
		}
	})

	zugang := &merkeAbfrage{pool: pool}
	e, err := LadeEinstellungen(ctx, zugang)
	if err != nil {
		t.Fatalf("LadeEinstellungen: %v", err)
	}
	if e.SchuleName != schule {
		t.Errorf("SchuleName = %q, erwartet %q: Die Einstellungen selbst müssen weiter ankommen", e.SchuleName, schule)
	}

	rows, err := pool.Query(ctx, zugang.sql)
	if err != nil {
		t.Fatalf("Abfrage von LadeEinstellungen noch einmal: %v", err)
	}
	defer rows.Close()
	zeilen := 0
	for rows.Next() {
		var schluessel string
		var wert *string
		if err := rows.Scan(&schluessel, &wert); err != nil {
			t.Fatalf("Zeile lesen: %v", err)
		}
		zeilen++
		if schluessel == AusweisLayoutSchluessel {
			t.Errorf("LadeEinstellungen liest die Zeile %q mit (%d Bytes)", schluessel, len(*wert))
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Zeilen: %v", err)
	}
	if zeilen == 0 {
		t.Fatal("die Abfrage liefert keine Zeile: Die Probe greift ins Leere")
	}

	// Das Design selbst bleibt über seinen eigenen Weg lesbar.
	if got, err := LadeAusweisLayout(ctx, pool); err != nil || got != design {
		t.Errorf("LadeAusweisLayout: %d Bytes, Fehler %v; erwartet %d Bytes", len(got), err, len(design))
	}
}
