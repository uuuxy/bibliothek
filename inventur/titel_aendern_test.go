package inventur

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestAendereTitel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)
	ctx := context.Background()

	eintrag := Titel{
		ISBN:                    "123",
		Titel:                   "Title",
		Autor:                   "Author",
		CoverURL:                "URL",
		Fach:                    "Math",
		Zweig:                   "A",
		Bestand:                 10,
		ZuletztGezaehlt:         nil, // handle date logic if needed
		Medientyp:               "Buch",
		ErweiterteEigenschaften: map[string]any{"key": "value"},
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "Subtitle",
		Verlag:                  "Publisher",
		Erscheinungsjahr:        2023,
		Signatur:                "SIG-123",
	}

	// Die Anweisung, wie AendereTitel sie absetzt; die Felder der Maske stehen als letzter Parameter.
	aenderAbfrage := regexp.QuoteMeta(sqlTitelAendern)
	felder := alleTitelFelder()
	// Danach läuft gleicheBestandAb.

	t.Run("success", func(t *testing.T) {
		erwarteFachBekannt(mock, eintrag.Fach) // läuft auf dem Pool, vor der Tx
		mock.ExpectBegin()
		erwarteTitelstand(mock, eintrag.ISBN, eintrag.Autor)
		mock.ExpectExec(aenderAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, "book-123", eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband, felder,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		// gleicheBestandAb query (in der Tx)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(10))
		mock.ExpectCommit()

		err := abfragen.AendereTitel(ctx, "book-123", eintrag, felder, &Bestandsangabe{Soll: eintrag.Bestand})
		if err != nil {
			t.Fatalf("abfragen.AendereTitel: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		mock.ExpectQuery(sqlTitelstandMuster).
			WithArgs("book-123").
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectRollback()

		err := abfragen.AendereTitel(ctx, "book-123", eintrag, felder, &Bestandsangabe{Soll: eintrag.Bestand})
		if !errors.Is(err, repository.ErrTitelNichtGefunden) {
			t.Errorf("abfragen.AendereTitel: Fehler %v, erwartet %v", err, repository.ErrTitelNichtGefunden)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		erwarteTitelstand(mock, eintrag.ISBN, eintrag.Autor)
		mock.ExpectExec(aenderAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, "book-123", eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband, felder,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		err := abfragen.AendereTitel(ctx, "book-123", eintrag, felder, &Bestandsangabe{Soll: eintrag.Bestand})
		if err == nil || !strings.Contains(err.Error(), "buch konnte nicht aktualisiert werden") {
			t.Errorf("abfragen.AendereTitel: Fehler %v, erwartet einen mit %q", err, "buch konnte nicht aktualisiert werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	// Ein Fehlschlag beim Abgleich des Bestands geht zurück (stünde er nur im Log, gälte ein Titel mit
	// falschem Bestand als gespeichert). Die Änderung des Titels rollt mit zurück.
	t.Run("sync-fehler wird zurückgegeben und rollt zurück", func(t *testing.T) {
		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		erwarteTitelstand(mock, eintrag.ISBN, eintrag.Autor)
		mock.ExpectExec(aenderAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, "book-123", eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband, felder,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare`).
			WithArgs("book-123").
			WillReturnError(fmt.Errorf("bestand nicht lesbar"))
		mock.ExpectRollback()

		err := abfragen.AendereTitel(ctx, "book-123", eintrag, felder, &Bestandsangabe{Soll: eintrag.Bestand})
		if err == nil || !strings.Contains(err.Error(), "exemplare konnten nicht synchronisiert werden") {
			t.Errorf("abfragen.AendereTitel: Fehler %v, erwartet einen mit %q", err, "exemplare konnten nicht synchronisiert werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

func TestGleicheBestandAb(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)
	ctx := context.Background()

	t.Run("increase stock", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

		codes := erwarteBarcodeVergabe(mock, 3) // 5 - 2 = 3
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs("book-123", codes).
			WillReturnResult(pgxmock.NewResult("INSERT", 3))

		err := abfragen.gleicheBestandAb(ctx, mock, "book-123", 5, "")
		if err != nil {
			t.Fatalf("abfragen.gleicheBestandAb: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("decrease stock - only unused", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		// Zwei ungenutzte aussondern.
		mock.ExpectQuery(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", 2).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("ex-1").AddRow("ex-2"))
		erwarteAussonderungsSpur(mock, "ex-1", "ex-2")

		err := abfragen.gleicheBestandAb(ctx, mock, "book-123", 3, "bearbeiter-1")
		if err != nil {
			t.Fatalf("abfragen.gleicheBestandAb: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("decrease stock - fallback to used", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		// Zwei ungenutzte aussondern, aber nur eines gefunden.
		mock.ExpectQuery(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", 2).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("ex-1"))

		// Rückfall für das übrige eine.
		mock.ExpectQuery(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", 1).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("ex-2"))
		erwarteAussonderungsSpur(mock, "ex-1", "ex-2")

		err := abfragen.gleicheBestandAb(ctx, mock, "book-123", 3, "bearbeiter-1")
		if err != nil {
			t.Fatalf("abfragen.gleicheBestandAb: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("decrease stock - ohne Bearbeiter abgelehnt", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		err := abfragen.gleicheBestandAb(ctx, mock, "book-123", 3, "")
		if !errors.Is(err, repository.ErrAussonderungOhneBearbeiter) {
			t.Errorf("abfragen.gleicheBestandAb: Fehler %v, erwartet %v", err, repository.ErrAussonderungOhneBearbeiter)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("no change in stock", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		err := abfragen.gleicheBestandAb(ctx, mock, "book-123", 5, "")
		if err != nil {
			t.Fatalf("abfragen.gleicheBestandAb: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

// erwarteAussonderungsSpur erwartet den Protokolleintrag für die genannten Exemplare.
func erwarteAussonderungsSpur(mock pgxmock.PgxPoolIface, exemplarIDs ...string) {
	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(exemplarIDs, "bearbeiter-1", repository.AussonderungsWegBestandskorrektur, repository.ProtokollAktionAusgesondert, "{}").
		WillReturnResult(pgxmock.NewResult("INSERT", int64(len(exemplarIDs))))
}
