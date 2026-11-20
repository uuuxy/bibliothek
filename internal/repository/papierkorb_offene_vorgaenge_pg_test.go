package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Die Tür prüft offene Ausleihen vor dem Löschen; dazwischen kann an der Theke eine Ausleihe
// entstehen. LegeLeserInPapierkorb prüft deshalb selbst, unter der Sperre der Leserzeile, und legt
// einen Leser mit einem Buch draußen nicht in den Papierkorb.
func TestLegeLeserInPapierkorb_NichtMitOffenerAusleihe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	leserID := testdaten.LegeSchuelerAn(t, pool, "S-PK-1", "Papierkorb", "6a")
	testdaten.LegeAusleiheMitFristAn(t, pool, leserID, "Buch draußen", time.Now().AddDate(0, 0, 14))

	err := repository.NewLeserAbfragen(pool).LegeLeserInPapierkorb(ctx, leserID, "", "Manuelle Löschung", nil)
	var blockiert *repository.LoeschenBlockiertFehler
	if !errors.As(err, &blockiert) || blockiert.OffeneAusleihen != 1 {
		t.Fatalf("LegeLeserInPapierkorb = %v, erwartet LoeschenBlockiertFehler mit 1 offenen Ausleihe", err)
	}
	var imPapierkorb bool
	if err := pool.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1`, leserID).Scan(&imPapierkorb); err != nil {
		t.Fatal(err)
	}
	if imPapierkorb {
		t.Error("der Leser liegt mit offener Ausleihe im Papierkorb")
	}
}
