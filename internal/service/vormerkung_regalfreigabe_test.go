package service

import (
	"context"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

const deleteVormerkungQuery = "DELETE FROM vormerkungen WHERE titel_id = \\$1 AND schueler_id = \\$2"

// TestRegalfreigabe_AnderesExemplar deckt den "Geisterbuch"-Fall ab: Für den Schüler
// lag Exemplar A im Reservierungsfach, er nimmt aber Exemplar B (Freihand). Die
// erfüllte Vormerkung wird gelöscht UND der Barcode von A als Regal-Hinweis gemeldet,
// damit A nicht unauffindbar im Fach liegen bleibt.
func TestRegalfreigabe_AnderesExemplar(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool: %v", err)
	}
	defer mock.Close()
	tx := beginTx(t, mock)

	copyB := &repository.BookCopy{ID: "exB", TitelID: "titel1"}

	// Die gelöschte Vormerkung hatte Exemplar A bereitgestellt (nicht B).
	mock.ExpectQuery(deleteVormerkungQuery).WithArgs("titel1", "s1").
		WillReturnRows(pgxmock.NewRows([]string{"bereitgestellt_exemplar_id"}).AddRow(strPtr("exA")))
	mock.ExpectQuery("SELECT barcode_id FROM buecher_exemplare").WithArgs("exA").
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("BC-A-123"))

	resp := &LoanResult{}
	entferneErfuellteVormerkung(context.Background(), tx, copyB, "s1", resp)

	if resp.RegalfreigabeBarcode != "BC-A-123" {
		t.Errorf("Regal-Hinweis erwartet 'BC-A-123', war %q", resp.RegalfreigabeBarcode)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// TestRegalfreigabe_SelbesExemplar: Nimmt der Schüler genau das reservierte Exemplar,
// gibt es keinen Regal-Hinweis (nichts liegt falsch).
func TestRegalfreigabe_SelbesExemplar(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool: %v", err)
	}
	defer mock.Close()
	tx := beginTx(t, mock)

	copyA := &repository.BookCopy{ID: "exA", TitelID: "titel1"}

	mock.ExpectQuery(deleteVormerkungQuery).WithArgs("titel1", "s1").
		WillReturnRows(pgxmock.NewRows([]string{"bereitgestellt_exemplar_id"}).AddRow(strPtr("exA")))
	// KEIN barcode-Lookup erwartet.

	resp := &LoanResult{}
	entferneErfuellteVormerkung(context.Background(), tx, copyA, "s1", resp)

	if resp.RegalfreigabeBarcode != "" {
		t.Errorf("kein Regal-Hinweis erwartet, war %q", resp.RegalfreigabeBarcode)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// TestRegalfreigabe_KeineVormerkung: Der Normalfall — keine Vormerkung vorhanden.
// Kein Fehler, kein Hinweis.
func TestRegalfreigabe_KeineVormerkung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool: %v", err)
	}
	defer mock.Close()
	tx := beginTx(t, mock)

	copyX := &repository.BookCopy{ID: "exX", TitelID: "titel9"}

	mock.ExpectQuery(deleteVormerkungQuery).WithArgs("titel9", "s1").
		WillReturnError(pgx.ErrNoRows)

	resp := &LoanResult{}
	entferneErfuellteVormerkung(context.Background(), tx, copyX, "s1", resp)

	if resp.RegalfreigabeBarcode != "" {
		t.Errorf("kein Hinweis erwartet, war %q", resp.RegalfreigabeBarcode)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}
