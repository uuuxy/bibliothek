package db_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// regelOhneLMF ist die Regel aus Migration 168: keine LMF-Nummer in barcode_id.
const regelOhneLMF = "chk_exemplar_barcode_keine_lmf_nummer"

// vorDerRegel öffnet eine Transaktion mit dem Stand vor Migration 168, in dem ein Exemplar noch
// eine LMF-Nummer tragen konnte, und rollt sie am Ende zurück; die Regel steht danach wieder.
func vorDerRegel(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx := context.Background()
	tx, err := pgtest.Pool(t).Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion öffnen: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("zurückrollen: %v", err)
		}
	})
	if _, err := tx.Exec(ctx, `ALTER TABLE buecher_exemplare DROP CONSTRAINT `+regelOhneLMF); err != nil {
		t.Fatalf("Regel herausnehmen: %v", err)
	}
	return ctx, tx
}

// fuehreAus spielt die echte Datei einer Migration in der Transaktion ein.
func fuehreAus(t *testing.T, ctx context.Context, tx pgx.Tx, datei string) {
	t.Helper()
	sql, err := os.ReadFile(filepath.Join("..", "migrations", datei))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("%s ausführen: %v", datei, err)
	}
}

// legeTitelAn legt einen Titel in der Transaktion an.
func legeTitelAn(t *testing.T, ctx context.Context, tx pgx.Tx, titel string) string {
	t.Helper()
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, medientyp) VALUES ($1, 'Buch') RETURNING id`, titel).
		Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// legeAn legt ein Exemplar des Titels in der Transaktion an und nennt seine Kennung.
func legeAn(t *testing.T, ctx context.Context, tx pgx.Tx, titelID, nummer string, gedruckt bool) string {
	t.Helper()
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt)
		VALUES ($1, $2, $3) RETURNING id`, titelID, nummer, gedruckt).Scan(&id); err != nil {
		t.Fatalf("Exemplar %s anlegen: %v", nummer, err)
	}
	return id
}

// exemplarStand ist, was die Migrationen an einem Exemplar ändern dürfen.
type exemplarStand struct {
	nummer, alt string
	gedruckt    bool
}

func leseStand(t *testing.T, ctx context.Context, tx pgx.Tx, titelID string) map[string]exemplarStand {
	t.Helper()
	rows, err := tx.Query(ctx, `
		SELECT id::text, barcode_id, coalesce(lmf_barcode, ''), etikett_gedruckt
		FROM buecher_exemplare WHERE titel_id = $1`, titelID)
	if err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}
	defer rows.Close()
	stand := map[string]exemplarStand{}
	for rows.Next() {
		var id string
		var s exemplarStand
		if err := rows.Scan(&id, &s.nummer, &s.alt, &s.gedruckt); err != nil {
			t.Fatalf("Exemplar lesen: %v", err)
		}
		stand[id] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}
	return stand
}

var kurzeNummer = regexp.MustCompile(`^B-([0-9]+)$`)

// ziffern liest die Zahl einer kurzen Nummer.
func ziffern(t *testing.T, nummer string) int64 {
	t.Helper()
	m := kurzeNummer.FindStringSubmatch(nummer)
	if m == nil {
		t.Fatalf("%q ist keine kurze Nummer", nummer)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Migration 167 gibt jedem Exemplar mit LMF-Nummer eine kurze Nummer aus barcode_seq und legt die
// LMF-Nummer nach lmf_barcode. Geprüft wird die echte Datei: Die kurzen Nummern liegen über einer
// von Hand eingetragenen, folgen der Reihenfolge der LMF-Nummern, und alles andere bleibt, auch
// etikett_gedruckt. Ein zweiter Lauf ändert nichts.
func TestMigration167_LMFExemplareBekommenKurzeNummern(t *testing.T) {
	ctx, tx := vorDerRegel(t)

	var stand int64
	if err := tx.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
		t.Fatalf("Stand der Sequenz: %v", err)
	}
	titelID := legeTitelAn(t, ctx, tx, "LMF-Umstellung")
	// Eine von Hand eingetragene Nummer über dem Stand der Sequenz: Keine kurze Nummer darf sie
	// treffen.
	handNummer := fmt.Sprintf("B-%d", stand+500)
	hand := legeAn(t, ctx, tx, titelID, handNummer, false)
	littera := legeAn(t, ctx, tx, titelID, "101634", false)
	// zweites liegt vor erstes: Ohne ORDER BY bekäme zweites die erste Nummer.
	zweites := legeAn(t, ctx, tx, titelID, "LMF-9783060158423-002", false)
	erstes := legeAn(t, ctx, tx, titelID, "LMF-9783060158423-001", true)
	klein := legeAn(t, ctx, tx, titelID, "lmf-9783125487710-001", true)

	fuehreAus(t, ctx, tx, "167_lmf_kurze_nummer.sql")
	nachher := leseStand(t, ctx, tx, titelID)

	for id, want := range map[string]exemplarStand{
		erstes:  {alt: "LMF-9783060158423-001", gedruckt: true},
		zweites: {alt: "LMF-9783060158423-002", gedruckt: false},
		klein:   {alt: "LMF-9783125487710-001", gedruckt: true},
	} {
		if n := ziffern(t, nachher[id].nummer); n <= stand+500 {
			t.Errorf("Exemplar %s bekam %s, erwartet eine Nummer über der von Hand eingetragenen %s", id, nachher[id].nummer, handNummer)
		}
		if got := nachher[id]; got.alt != want.alt || got.gedruckt != want.gedruckt {
			t.Errorf("Exemplar %s: altes Etikett %q, gedruckt %v; erwartet %q, %v", id, got.alt, got.gedruckt, want.alt, want.gedruckt)
		}
	}
	if ziffern(t, nachher[erstes].nummer) >= ziffern(t, nachher[zweites].nummer) ||
		ziffern(t, nachher[zweites].nummer) >= ziffern(t, nachher[klein].nummer) {
		t.Errorf("Nummern %s, %s, %s; erwartet aufsteigend in der Reihenfolge der LMF-Nummern",
			nachher[erstes].nummer, nachher[zweites].nummer, nachher[klein].nummer)
	}
	for id, want := range map[string]string{hand: handNummer, littera: "101634"} {
		if got := nachher[id]; got.nummer != want || got.alt != "" {
			t.Errorf("Exemplar %s: %q mit altem Etikett %q, erwartet unverändert %q ohne", id, got.nummer, got.alt, want)
		}
	}

	fuehreAus(t, ctx, tx, "167_lmf_kurze_nummer.sql")
	for id, s := range leseStand(t, ctx, tx, titelID) {
		if s != nachher[id] {
			t.Errorf("zweiter Lauf änderte Exemplar %s: %+v, vorher %+v", id, s, nachher[id])
		}
	}
}

// Migration 168 stellt um, was seit 167 noch eine LMF-Nummer bekam, und setzt dann die Regel:
// Danach nimmt die Datenbank keine LMF-Nummer mehr in barcode_id an, in keiner Schreibweise. Ein
// zweiter Lauf ändert nichts.
func TestMigration168_RegelNachDerUmstellung(t *testing.T) {
	ctx, tx := vorDerRegel(t)

	titelID := legeTitelAn(t, ctx, tx, "LMF-Regel")
	nachgekommen := legeAn(t, ctx, tx, titelID, "LMF-9783060158423-031", true)

	fuehreAus(t, ctx, tx, "168_barcode_keine_lmf_nummer.sql")
	nachher := leseStand(t, ctx, tx, titelID)
	if s := nachher[nachgekommen]; !kurzeNummer.MatchString(s.nummer) || s.alt != "LMF-9783060158423-031" || !s.gedruckt {
		t.Errorf("Exemplar nach Migration 168: %+v, erwartet eine kurze Nummer mit dem alten Etikett", s)
	}

	for _, nummer := range []string{"LMF-9783060158423-032", "lmf-9783060158423-033"} {
		if _, err := tx.Exec(ctx, `SAVEPOINT probe`); err != nil {
			t.Fatal(err)
		}
		_, err := tx.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, titelID, nummer)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != regelOhneLMF {
			t.Errorf("%s nach Migration 168: %v, erwartet die Ablehnung durch %s", nummer, err, regelOhneLMF)
		}
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT probe`); err != nil {
			t.Fatal(err)
		}
	}

	fuehreAus(t, ctx, tx, "168_barcode_keine_lmf_nummer.sql")
	if s := leseStand(t, ctx, tx, titelID)[nachgekommen]; s != nachher[nachgekommen] {
		t.Errorf("zweiter Lauf änderte das Exemplar: %+v, vorher %+v", s, nachher[nachgekommen])
	}
}
