package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Baustein hält beide Hälften in der Transaktion des Aufrufers: Rollt der Aufrufer
// zurück, ist das Buch weiter ausgesondert UND die Forderung weiter offen; committet er,
// ist beides erledigt. Vor dem 15.09.2026 committete der Rumpf selbst — ein Aufrufer, der
// danach scheiterte, ließ das Buch zurück im Regal und den Rest seiner Buchung ungetan.
func TestHoleExemplarZurueck_HaeltBeideHaelftenInDerTransaktionDesAufrufers(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	exemplarID, schadensfallID, bearbeiterID := rueckholenAufbau(t, pool)

	lage := func() (ausgesondert, offen bool) {
		if err := pool.QueryRow(ctx, `
			SELECT e.ist_ausgesondert, f.storniert_am IS NULL
			FROM buecher_exemplare e JOIN schadensfaelle f ON f.exemplar_id = e.id
			WHERE e.id = $1 AND f.id = $2`, exemplarID, schadensfallID).Scan(&ausgesondert, &offen); err != nil {
			t.Fatalf("Lage lesen: %v", err)
		}
		return
	}

	// Rollback: nichts davon bleibt.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	befund, err := HoleExemplarZurueck(ctx, tx, exemplarID, bearbeiterID)
	if err != nil {
		t.Fatalf("zurückholen in tx: %v", err)
	}
	if befund.StornierteForderungen != 1 {
		t.Errorf("in der Transaktion: %d stornierte Forderungen, erwartet 1", befund.StornierteForderungen)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if ausgesondert, offen := lage(); !ausgesondert || !offen {
		t.Fatalf("nach Rollback: ausgesondert=%v, forderung offen=%v — erwartet beides true (nichts darf bleiben)", ausgesondert, offen)
	}

	// Commit: beides erledigt.
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := HoleExemplarZurueck(ctx, tx, exemplarID, bearbeiterID); err != nil {
		t.Fatalf("zurückholen in tx: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if ausgesondert, offen := lage(); ausgesondert || offen {
		t.Fatalf("nach Commit: ausgesondert=%v, forderung offen=%v — erwartet beides false", ausgesondert, offen)
	}
}

// Ein Exemplar, das es nicht (mehr) gibt, ist kein stiller Erfolg.
func TestHoleExemplarZurueck_UnbekanntesExemplarIstKeinErfolg(t *testing.T) {
	pool := pgtest.Pool(t)
	_, _, bearbeiterID := rueckholenAufbau(t, pool)
	_, err := HoleExemplarZurueck(context.Background(), pool, "00000000-0000-0000-0000-000000000000", bearbeiterID)
	if !errors.Is(err, ErrExemplarNichtGefunden) {
		t.Fatalf("erwartet ErrExemplarNichtGefunden, bekommen: %v", err)
	}
}

// rueckholenAufbau: ein als Verlust abgeschriebenes Exemplar mit offener Forderung
// „nicht zurückgegeben" bei einem Kind, dazu die buchende Mitarbeiterin.
func rueckholenAufbau(t *testing.T, pool *pgxpool.Pool) (exemplarID, schadensfallID, bearbeiterID string) {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Rück', 'Holer', $2, 'mitarbeiter', true) RETURNING id`,
		"MA-"+suffix, "rueckholen-"+suffix+"@schule.invalid").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Anna', 'Verlust', '07B', 2031) RETURNING id`, "S-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Rückhol-Testband', 'Prüfer', 'Buch', true) RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, zustand_notiz, einkaufspreis)
		VALUES ($1, $2, false, true, 'VERLUST', 'Nicht zurückgegeben', 20.00) RETURNING id`,
		titelID, "B-RH-"+suffix).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Nicht zurückgegeben', 20.00, 'nicht_zurueckgegeben') RETURNING id`,
		exemplarID, schuelerID).Scan(&schadensfallID); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	return exemplarID, schadensfallID, bearbeiterID
}
