package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"
)

// Der eindeutige Index lässt höchstens eine offene Ausleihe je Exemplar zu. Prallt das INSERT
// daran ab, liefert AusleihAbfragen.LegeAn ErrAusleiheKonflikt. Ohne den Fehlerwert bekäme der
// Rufer keine Zeile und keinen Fehler, hielte das für Erfolg und meldete dem Arbeitsplatz eine
// Ausleihe, die nie geschrieben wurde.
//
// Nur am echten Postgres prüfbar: Es geht um das Verhalten von ON CONFLICT DO NOTHING …
// RETURNING am partiellen Unique-Index.
func TestAusleiheLegeAn_MeldetKonfliktStattStillemNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "Konflikt", 1)
	ersterSchueler := seedSchueler(t, pool, "KFL-1", "Anna", "7a")
	zweiterSchueler := seedSchueler(t, pool, "KFL-2", "Bernd", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	frist := time.Now().AddDate(0, 0, 21)

	repo := NewAusleihAbfragen(pool)

	tx1, err := repo.BeginneTx(ctx)
	if err != nil {
		t.Fatalf("BeginneTx: %v", err)
	}
	loan, err := repo.LegeAn(ctx, tx1, ex[0], ersterSchueler, bearbeiter, frist, false)
	if err != nil {
		t.Fatalf("erste Ausleihe: %v", err)
	}
	if loan == nil {
		t.Fatal("erste Ausleihe lieferte keine Zeile")
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Ein zweiter Vorgang auf dasselbe Exemplar für einen anderen Schüler prallt ab.
	tx2, err := repo.BeginneTx(ctx)
	if err != nil {
		t.Fatalf("BeginneTx 2: %v", err)
	}
	defer db.Verwerfe(ctx, tx2)

	zweite, err := repo.LegeAn(ctx, tx2, ex[0], zweiterSchueler, bearbeiter, frist, false)

	if zweite != nil {
		t.Errorf("zweite Ausleihe lieferte unerwartet eine Zeile: %+v", zweite)
	}
	if err == nil {
		t.Fatal("zweite Ausleihe meldete KEINEN Fehler — genau der stille Datenverlust")
	}
	if !errors.Is(err, ErrAusleiheKonflikt) {
		t.Errorf("erwartet ErrAusleiheKonflikt, war: %v", err)
	}
}

// Gegenprobe: Nach einer Rückgabe ist der Index wieder frei, und dasselbe Exemplar lässt sich
// erneut ausleihen.
func TestAusleiheLegeAn_NachRueckgabeWiederAusleihbar(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "KonfliktFrei", 1)
	schueler := seedSchueler(t, pool, "KFL-3", "Clara", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	frist := time.Now().AddDate(0, 0, 21)

	repo := NewAusleihAbfragen(pool)

	tx1, err := repo.BeginneTx(ctx)
	if err != nil {
		t.Fatalf("BeginneTx 1: %v", err)
	}
	erste, err := repo.LegeAn(ctx, tx1, ex[0], schueler, bearbeiter, frist, false)
	if err != nil || erste == nil {
		db.Verwerfe(ctx, tx1)
		t.Fatalf("erste Ausleihe: %v", err)
	}
	if err := repo.BucheRueckgabe(ctx, tx1, erste.ID, bearbeiter, false); err != nil {
		db.Verwerfe(ctx, tx1)
		t.Fatalf("Rückgabe: %v", err)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("Commit 1: %v", err)
	}

	tx2, err := repo.BeginneTx(ctx)
	if err != nil {
		t.Fatalf("BeginneTx 2: %v", err)
	}
	defer db.Verwerfe(ctx, tx2)

	zweite, err := repo.LegeAn(ctx, tx2, ex[0], schueler, bearbeiter, frist, false)
	if err != nil {
		t.Fatalf("Ausleihe nach Rückgabe schlug fehl: %v", err)
	}
	if zweite == nil {
		t.Fatal("Ausleihe nach Rückgabe lieferte keine Zeile")
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("Commit 2: %v", err)
	}
}
