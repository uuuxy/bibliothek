package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"
)

// Der eindeutige Index lässt höchstens eine offene Ausleihe je Exemplar zu. Prallt
// der INSERT daran ab, gab CreateLoanTx früher (nil, nil) zurück — kein Fehler, keine
// Zeile. Der Aufrufer prüfte nur err, hielt das für Erfolg und meldete dem
// Arbeitsplatz eine Ausleihe, die nie geschrieben wurde.
//
// Nur gegen echtes Postgres prüfbar: Es geht um das Verhalten von
// "ON CONFLICT DO NOTHING ... RETURNING" am partiellen Unique-Index.
func TestCreateLoanTx_MeldetKonfliktStattStillemNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	_, ex := seedSignaturMitExemplaren(t, pool, "Konflikt", 1)
	ersterSchueler := seedSchueler(t, pool, "KFL-1", "Anna", "7a")
	zweiterSchueler := seedSchueler(t, pool, "KFL-2", "Bernd", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	frist := time.Now().AddDate(0, 0, 21)

	repo := NewLoanRepository(pool)

	tx1, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	loan, err := repo.CreateLoanTx(ctx, tx1, ex[0], ersterSchueler, bearbeiter, frist)
	if err != nil {
		t.Fatalf("erste Ausleihe: %v", err)
	}
	if loan == nil {
		t.Fatal("erste Ausleihe lieferte keine Zeile")
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Zweiter Vorgang auf dasselbe Exemplar, anderer Schüler — muss abprallen.
	tx2, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx 2: %v", err)
	}
	defer db.SafeRollback(ctx, tx2)

	zweite, err := repo.CreateLoanTx(ctx, tx2, ex[0], zweiterSchueler, bearbeiter, frist)

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

// Gegenprobe: Nach einer Rückgabe ist der Index wieder frei, dieselbe Signatur darf
// erneut ausgeliehen werden. Sonst hätte der Fix den Normalbetrieb kaputtgemacht.
func TestCreateLoanTx_NachRueckgabeWiederAusleihbar(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	_, ex := seedSignaturMitExemplaren(t, pool, "KonfliktFrei", 1)
	schueler := seedSchueler(t, pool, "KFL-3", "Clara", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	frist := time.Now().AddDate(0, 0, 21)

	repo := NewLoanRepository(pool)

	erste, err := repo.CreateLoan(ctx, ex[0], schueler, bearbeiter, frist)
	if err != nil || erste == nil {
		t.Fatalf("erste Ausleihe: %v", err)
	}
	if err := repo.ReturnLoan(ctx, erste.ID, bearbeiter, false); err != nil {
		t.Fatalf("Rückgabe: %v", err)
	}

	zweite, err := repo.CreateLoan(ctx, ex[0], schueler, bearbeiter, frist)
	if err != nil {
		t.Fatalf("Ausleihe nach Rückgabe schlug fehl: %v", err)
	}
	if zweite == nil {
		t.Fatal("Ausleihe nach Rückgabe lieferte keine Zeile")
	}
}
