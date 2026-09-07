package inventur

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exemplare, die das System selbst anlegt — Bestandskorrektur in der Buchmaske und
// Excel-Sammelimport — bekommen ihre Nummer aus barcode_seq, derselben Quelle wie das
// Bestellwesen, die Handvergabe und der Littera-Import (Migration 068: „EINE Quelle für
// alle Wege"). Bis zum 07.09.2026 zogen genau diese zwei Pfade aus einer eigenen, nirgends
// deklarierten Sequenz und prägten „SYS-…"-Nummern — ein zweiter Nummernkreis, den 068
// schlicht übersehen hatte (Migration 105 räumt ihn ab).
//
// Warum ein PG-Test: Die Vergabe ist SQL (nextval gegen Bestandsabgleich), und die
// eigentliche Zusicherung — keine Kollision mit einer von Hand vorweggenommenen Nummer —
// zeigt sich erst am UNIQUE-Constraint. pgxmock spielt das nur nach.
func TestBestandskorrekturUndImportZiehenAusBarcodeSeq(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const titelKorrektur = "00000000-0000-0000-0000-00000000c001"
	const titelImport = "00000000-0000-0000-0000-00000000c002"
	const isbnImport = "978-nk-import"

	for _, sql := range []string{
		`DELETE FROM buecher_exemplare WHERE titel_id IN ('` + titelKorrektur + `','` + titelImport + `')`,
		`DELETE FROM buecher_titel WHERE id IN ('` + titelKorrektur + `','` + titelImport + `')`,
		`INSERT INTO buecher_titel (id, titel, isbn) VALUES
			('` + titelKorrektur + `', 'Nummernkreis Korrektur', '978-nk-korr'),
			('` + titelImport + `', 'Nummernkreis Import', '` + isbnImport + `')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%.50s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id IN ($1, $2)`, titelKorrektur, titelImport); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	// Von Hand die NÄCHSTE Sequenznummer vorwegnehmen — der Fall, den 068 beschreibt:
	// Die Sequenz kennt nur ihre eigenen Züge; eine in der Exemplarkarte eingetippte
	// Nummer steht in der Tabelle, nicht in der Sequenz.
	var letzte int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&letzte); err != nil {
		t.Fatalf("barcode_seq lesen: %v", err)
	}
	vorweg := fmt.Sprintf("B-%05d", letzte+1)
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true)`,
		titelKorrektur, vorweg); err != nil {
		t.Fatalf("Handnummer %s: %v", vorweg, err)
	}

	repo := NewBookRepository(pool)

	// Bestandskorrektur: 1 vorhanden, 4 gewünscht → 3 neue.
	if err := repo.syncBookStock(ctx, pool, titelKorrektur, 4); err != nil {
		t.Fatalf("syncBookStock: %v", err)
	}
	pruefeNummernkreis(t, pool, titelKorrektur, 4)

	// Excel-Sammelimport: 2 Stück zur ISBN.
	if err := repo.legeImportExemplareAn(ctx, pool, []string{isbnImport}, []int32{2}); err != nil {
		t.Fatalf("legeImportExemplareAn: %v", err)
	}
	pruefeNummernkreis(t, pool, titelImport, 2)
}

// pruefeNummernkreis verlangt: genau erwartet Exemplare am Titel, jedes mit einer
// B-Nummer im Format der Vergabe — und keine einzige aus dem alten SYS-Kreis.
func pruefeNummernkreis(t *testing.T, pool *pgxpool.Pool, titelID string, erwartet int) {
	t.Helper()
	var gesamt, b, sys int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*),
		       count(*) FILTER (WHERE barcode_id ~ '^B-[0-9]{5,}$'),
		       count(*) FILTER (WHERE barcode_id LIKE 'SYS-%')
		FROM buecher_exemplare WHERE titel_id = $1`, titelID).Scan(&gesamt, &b, &sys); err != nil {
		t.Fatalf("Exemplare zählen: %v", err)
	}
	if gesamt != erwartet {
		t.Errorf("%s: %d Exemplare, erwartet %d", titelID, gesamt, erwartet)
	}
	if sys != 0 {
		t.Errorf("%s: %d Exemplar(e) aus dem alten SYS-Kreis — die Vergabe läuft nicht über barcode_seq", titelID, sys)
	}
	if b != gesamt {
		t.Errorf("%s: nur %d von %d Exemplaren tragen eine B-Nummer", titelID, b, gesamt)
	}
}
