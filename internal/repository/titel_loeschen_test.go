package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Bricht die Abfrage der verliehenen Exemplare mittendrin ab, gilt nicht „keine verliehenen":
// Sonst löschte ein Fehler der Verbindung den Titel samt der verliehenen Exemplare, die niemand
// bestätigt hat.
func TestLoescheTitel_LeseFehlerBlocktLoeschung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	const titelID = "33333333-3333-3333-3333-333333333333"
	leseFehler := errors.New("verbindung während der iteration verloren")

	mock.ExpectBegin()
	erwarteSperreDerAuflagen(mock)
	mock.ExpectQuery(`FROM buecher_titel WHERE id = ANY`).WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn"}).AddRow(titelID, "Titel", "", ""))
	mock.ExpectQuery(`FROM buecher_exemplare e WHERE e\.titel_id = ANY`).WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"titel_id", "id", "barcode_id", "lmf_barcode", "war_im_bestand"}))
	mock.ExpectQuery(`SELECT e\.titel_id::text, e\.id::text FROM buecher_exemplare e`).WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"titel_id", "id"}).
			AddRow(titelID, "ex-1").RowError(0, leseFehler))
	// Kein Löschen: Bei unklarer Lage wird nichts entfernt.
	mock.ExpectRollback()

	_, err = NewTitelAbfragen(mock).LoescheTitel(context.Background(), TitelLoeschauftrag{
		TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: "admin-1"})
	if err == nil {
		t.Fatal("ein Lesefehler darf nicht als 'keine verliehenen Exemplare' durchgehen")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// ProtokolliereAusleihe und ProtokolliereRueckgabe schreiben in die übergebene Transaktion und
// öffnen keine eigene: pgxmock kennt nur das eine Begin des Rufers, ein zweites Begin oder ein
// Commit in der Abfrage träfe keine Erwartung und machte den Test rot.
func TestProtokolliereAusleihe_SchreibtInDieUebergebeneTransaktion(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	repo := NewProtokollAbfragen(mock)
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", "ex-1", pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "RETURN", "ex-1", pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.ProtokolliereAusleihe(ctx, tx, "ex-1", "s-1", "", "b-1"); err != nil {
		t.Fatalf("ProtokolliereAusleihe: %v", err)
	}
	if err := repo.ProtokolliereRueckgabe(ctx, tx, "ex-1", "s-1", "", "b-1"); err != nil {
		t.Fatalf("ProtokolliereRueckgabe: %v", err)
	}
	// Den Commit setzt der Rufer, nicht die Abfrage.
	mock.ExpectCommit()
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Erwartungen: %v", err)
	}
}

// erwarteSperreDerAuflagen: Das Löschen von Titeln nimmt als Erstes die Sperre der Auflagen
// (nimmAuflagenSperre), vor jeder Zeilensperre.
func erwarteSperreDerAuflagen(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(auflagenSperre).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
}

// erwarteAuflagenSperre: Vor dem Löschen der Titel nimmt ladeWerkeDerTitel die Sperre der
// Auflagen noch einmal und liest die Werke. Hier gehört kein Titel zu einem, also räumt
// raeumeWerkeAuf danach nichts auf.
func erwarteAuflagenSperre(mock pgxmock.PgxPoolIface, ids []string) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(auflagenSperre).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT DISTINCT werk_id::text FROM buecher_titel`).WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"werk_id"}))
}

// Das Löschen räumt nur laufende Ausleihen ab, deren Exemplar der Mensch bestätigt hat. Eine,
// die nach der Rückfrage zustande kam, bricht es ab, statt das Buch ungesehen auszutragen.
func TestPruefeVerlieheneBestaetigt(t *testing.T) {
	offene := []offeneAusleihe{{ExemplarID: "ex-1"}, {ExemplarID: "ex-2"}}
	if err := pruefeVerlieheneBestaetigt(offene, []string{"ex-2", "ex-1"}); err != nil {
		t.Errorf("beide bestätigt: %v", err)
	}
	if err := pruefeVerlieheneBestaetigt(nil, nil); err != nil {
		t.Errorf("nichts verliehen: %v", err)
	}
	if err := pruefeVerlieheneBestaetigt(offene, []string{"ex-1"}); !errors.Is(err, ErrTitelInzwischenVerliehen) {
		t.Errorf("ex-2 nicht bestätigt: %v, erwartet ErrTitelInzwischenVerliehen", err)
	}
}
