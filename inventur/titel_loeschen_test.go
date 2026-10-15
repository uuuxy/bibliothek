package inventur

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Hier wird die Reihenfolge geprüft; pgxmock verlangt sie genau so, wie sie erwartet wird: Begin
// zuerst, dann jedes Lesen in der Transaktion, dann die beiden Befehle des Löschens, die ihre Spur
// über RETURNING liefern. Ein Lesen vor dem Begin lässt den Test rot werden, weil dann eine
// Abfrage vor dem erwarteten Begin ankommt.
func TestLoescheTitel(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1", "id-2"}

	// Die Leser in der Transaktion, in ihrer Reihenfolge — den Erfolgs- und den
	// Nicht-gefunden-Fall unterscheidet erst der Titel-DELETE danach.
	erwarteLeser := func(mock pgxmock.PgxPoolIface, cover string) {
		mock.ExpectBegin()
		erwarteAuflagenSperre(mock, ids)
		// Schnappschüsse der Barcodes aller Exemplare vor den DELETEs: Die Auskunft an der Theke findet
		// gelöschte Exemplare nur über diese Spur.
		mock.ExpectQuery(`FROM buecher_exemplare e`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "titel", "war_im_bestand"}).
				AddRow("ex-1", "BC-1", "Titel Eins", true))
		// Die drei Kinder des Titels über CASCADE: Vormerkungen, Reservierungen und Zuordnungen von
		// Klassensätzen.
		mock.ExpectQuery(`FROM vormerkungen v`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "wer", "status", "seit", "schueler_id"}))
		mock.ExpectQuery(`FROM klassensatz_reservierungen r`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse", "status", "seit"}))
		mock.ExpectQuery(`FROM class_books c`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse"}))
		zeilen := pgxmock.NewRows([]string{"cover_url"})
		if cover != "" {
			zeilen.AddRow(cover)
		}
		mock.ExpectQuery(`SELECT t\.cover_url FROM buecher_titel t\s+WHERE t\.id = ANY\(\$1::uuid\[\]\) AND t\.cover_url LIKE '/uploads/%'`).
			WithArgs(ids).
			WillReturnRows(zeilen)
	}
	schadenSpalten := []string{"id", "exemplar_id", "barcode_id", "titel", "schuldner", "schueler_id", "betrag", "beschreibung", "seit"}
	ausleiheSpalten := []string{"id", "exemplar_id", "barcode_id", "titel", "entleiher", "schueler_id", "seit"}

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		err = abfragen.LoescheTitel(ctx, []string{})
		if err != nil {
			t.Fatalf("abfragen.LoescheTitel: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	// Scheitert das Löschen der Ausleihen, bleibt nichts stehen: kein Titel-DELETE,
	// keine Spur, Rollback. Ohne die Spur verschwände ein verliehenes Buch lautlos.
	t.Run("Ausleihen nicht löschbar", func(t *testing.T) {
		imTestVerzeichnis(t) // LoescheTitel legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		erwarteLeser(mock, "")
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db error"))
		mock.ExpectRollback()

		err = abfragen.LoescheTitel(ctx, ids)
		if err == nil || !strings.Contains(err.Error(), "ausleihen konnten nicht gelöscht werden") {
			t.Errorf("abfragen.LoescheTitel: Fehler %v, erwartet einen mit %q", err, "ausleihen konnten nicht gelöscht werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		imTestVerzeichnis(t) // LoescheTitel legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		erwarteLeser(mock, "/uploads/cover1.jpg")
		// Offene Forderungen und laufende Ausleihen kommen als Spur aus dem Befehl des Löschens selbst
		// (RETURNING): Eine unbezahlte Forderung ist Geld, das ein Schüler schuldet, und darf nicht
		// spurlos mit dem Titel verschwinden.
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(ausleiheSpalten))
		mock.ExpectExec(`DELETE FROM buecher_titel WHERE id = ANY\(\$1::uuid\[\]\)`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectExec(`INSERT INTO audit_log`).
			WithArgs("ex-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		err = abfragen.LoescheTitel(ctx, ids)
		if err != nil {
			t.Fatalf("abfragen.LoescheTitel: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("book not found", func(t *testing.T) {
		imTestVerzeichnis(t) // LoescheTitel legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		erwarteLeser(mock, "/uploads/cover1.jpg")
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(ausleiheSpalten))
		mock.ExpectExec(`DELETE FROM buecher_titel`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectRollback()

		err = abfragen.LoescheTitel(ctx, ids)
		if !errors.Is(err, ErrTitelNichtGefunden) {
			t.Errorf("abfragen.LoescheTitel: Fehler %v, erwartet %v", err, ErrTitelNichtGefunden)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

// erwarteAuflagenSperre: LoescheTitel nimmt als Erstes die Sperre der Auflagen und liest die
// Werke der Titel (repository.LadeWerkeDerTitel); hier gehört keiner zu einem, also räumt
// repository.RaeumeWerkeAuf danach nichts auf.
func erwarteAuflagenSperre(mock pgxmock.PgxPoolIface, ids any) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT DISTINCT werk_id::text FROM buecher_titel`).WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"werk_id"}))
}
