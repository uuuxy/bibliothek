package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// snapshotTitelRows liefert die Zeile, die DeleteTitle vor dem Löschen für das Audit-Log
// sichert.
func snapshotTitelRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"titel", "autor", "isbn"}).
		AddRow("Der Vorleser", "Bernhard Schlink", "9783161484100")
}

// TestDeleteTitle_MeldetAktiveAusleihenAlsSentinel sichert die Bedienbarkeit des
// Löschpfads ab.
//
// Die Sperre selbst hat immer funktioniert — ein Titel mit verliehenen Exemplaren wurde
// nie gelöscht. Unbrauchbar war die Rückmeldung: Der Handler erkannte den Fall an
// err.Error()[:22] == "Löschen fehlgeschlagen:" und traf damit nie. Der Text begann
// klein, das Literal groß, und 22 Bytes können ein 24-Byte-Literal ohnehin nicht treffen
// (die Umlaute zählen doppelt). Jeder blockierte Löschversuch endete deshalb als HTTP 500
// — und die Liste der noch verliehenen Barcodes, also genau die Auskunft, mit der man
// weiterarbeitet, verschwand hinter „Serverfehler".
//
// Deshalb ein Sentinel statt eines Textvergleichs: errors.Is überlebt jede Umformulierung.
func TestDeleteTitle_MeldetAktiveAusleihenAlsSentinel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	const titelID = "11111111-1111-1111-1111-111111111111"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).
			AddRow("B-00042").AddRow("B-00043"))
	// KEIN ExpectExec("DELETE …"): Der Abbruch muss vor jedem zerstörenden Statement kommen.
	mock.ExpectRollback()

	err = repo.DeleteTitle(context.Background(), titelID, "admin-1")

	if !errors.Is(err, ErrTitelHatAktiveAusleihen) {
		t.Fatalf("erwartet ErrTitelHatAktiveAusleihen (der Handler antwortet darauf mit HTTP 400), bekam: %v", err)
	}
	// Die Barcodes sind der Grund, warum diese Meldung überhaupt zum Nutzer soll.
	for _, barcode := range []string{"B-00042", "B-00043"} {
		if !strings.Contains(err.Error(), barcode) {
			t.Errorf("Meldung nennt den noch verliehenen Barcode %s nicht: %v", barcode, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// TestDeleteTitle_LoeschtOhneAktiveAusleihen ist die Gegenprobe: Ohne offene Ausleihen
// darf die Sperre nicht greifen, sonst liesse sich nie ein Titel entfernen.
func TestDeleteTitle_LoeschtOhneAktiveAusleihen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	const titelID = "22222222-2222-2222-2222-222222222222"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"})) // keine offenen Ausleihen
	// Barcode-Snapshots der Exemplare vor den DELETEs — die Tresen-Auskunft findet
	// gelöschte Exemplare nur über diese Spur (Befund 01.09.2026).
	mock.ExpectQuery("FROM buecher_exemplare WHERE titel_id").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id"}).
			AddRow("ex-1", "B-00001"))
	mock.ExpectExec("DELETE FROM schadensfaelle").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM ausleihen").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM buecher_exemplare").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))
	mock.ExpectExec("DELETE FROM buecher_titel").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	// Der Audit-Eintrag ist revisionssicher gefordert — Tabelle, Aktion und Datensatz
	// werden deshalb festgenagelt, der Rest (Bearbeiter, Kontext, Details-JSON) nicht.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("buecher_titel", "DELETE", titelID,
			pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Je Exemplar der Barcode-Snapshot im Geschwister-Format (DeleteCopy, Verlust).
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("buecher_exemplare", "DELETE", "ex-1",
			pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := repo.DeleteTitle(context.Background(), titelID, "admin-1"); err != nil {
		t.Fatalf("Löschen ohne offene Ausleihen muss durchgehen, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// TestDeleteTitle_LeseFehlerBlocktLoeschung: Bricht die Ausleih-Abfrage mittendrin ab,
// darf NICHT von „keine offenen Ausleihen" ausgegangen werden — sonst löschte ein
// Verbindungsfehler den Titel samt verliehener Exemplare.
func TestDeleteTitle_LeseFehlerBlocktLoeschung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	const titelID = "33333333-3333-3333-3333-333333333333"
	leseFehler := errors.New("Verbindung während der Iteration verloren")

	mock.ExpectBegin()
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).
			AddRow("B-00042").RowError(0, leseFehler))
	// KEIN ExpectExec: Bei unklarer Lage wird nichts gelöscht.
	mock.ExpectRollback()

	err = repo.DeleteTitle(context.Background(), titelID, "admin-1")
	if err == nil {
		t.Fatal("ein Lesefehler darf nicht als 'keine offenen Ausleihen' durchgehen")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}
