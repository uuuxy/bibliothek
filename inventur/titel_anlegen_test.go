package inventur

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestLegeTitelAn(t *testing.T) {
	eintrag := Titel{
		ISBN:                    "123",
		Titel:                   "Title",
		Autor:                   "Author",
		CoverURL:                "URL",
		Fach:                    "Math",
		Zweig:                   "A",
		Bestand:                 10,
		ZuletztGezaehlt:         nil,
		Medientyp:               "Buch",
		ErweiterteEigenschaften: map[string]any{"key": "value"},
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "Subtitle",
		Verlag:                  "Publisher",
		Erscheinungsjahr:        2023,
		Signatur:                "SIG-123",
	}

	anlegeAbfrage := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, signatur, ist_lernmittel, auflage, listenpreis, mehrjahresband\) VALUES \(NULLIF\(\$1, ''\), \$2, \$3, \$4, NULLIF\(\$5, ''\), \$6, NULLIF\(\$7::text, ''\)::date, \$8, \$9, NULLIF\(\$10, 0\), NULLIF\(\$11, 0\), \$12, \$13, \$14, NULLIF\(\$15, ''\), \$16, NULLIF\(\$17, ''\), \$18, \$19\) RETURNING id`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		erwarteKeineDublette(mock)
		mock.ExpectQuery(anlegeAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare e WHERE e.titel_id = \$1 AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		codes := erwarteBarcodeVergabe(mock, 10)
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs("book-123", codes).
			WillReturnResult(pgxmock.NewResult("INSERT", 10))

		mock.ExpectCommit()

		id, err := abfragen.LegeTitelAn(ctx, eintrag)
		if err != nil {
			t.Fatalf("abfragen.LegeTitelAn: %v", err)
		}
		if id != "book-123" {
			t.Errorf("id = %q, erwartet %q", id, "book-123")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		erwarteKeineDublette(mock)
		mock.ExpectQuery(anlegeAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		_, err = abfragen.LegeTitelAn(ctx, eintrag)
		if err == nil || !strings.Contains(err.Error(), "buch konnte nicht erstellt werden") {
			t.Errorf("abfragen.LegeTitelAn: Fehler %v, erwartet einen mit %q", err, "buch konnte nicht erstellt werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("sync error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		erwarteKeineDublette(mock)
		mock.ExpectQuery(anlegeAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.ErweiterteEigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare`).
			WithArgs("book-123").
			WillReturnError(fmt.Errorf("bestand nicht lesbar"))
		mock.ExpectRollback()

		_, err = abfragen.LegeTitelAn(ctx, eintrag)
		if err == nil || !strings.Contains(err.Error(), "exemplare konnten nicht angelegt werden") {
			t.Errorf("abfragen.LegeTitelAn: Fehler %v, erwartet einen mit %q", err, "exemplare konnten nicht angelegt werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

func TestSchreibeTitel(t *testing.T) {
	eintrag := Titel{
		ISBN:                    "123",
		Titel:                   "Title",
		Autor:                   "Author",
		CoverURL:                "URL",
		Fach:                    "Math",
		Zweig:                   "A",
		Bestand:                 2,
		ZuletztGezaehlt:         nil,
		Medientyp:               "Buch",
		ErweiterteEigenschaften: map[string]any{"key": "value"},
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "Subtitle",
		Verlag:                  "Publisher",
		Erscheinungsjahr:        2023,
		Signatur:                "SIG-123",
	}

	schreibAbfrage := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis\) VALUES \(\$1, \$2, \$3, \$4, NULLIF\(\$5, ''\), \$6, NULLIF\(\$7::text, ''\)::date, \$8, NULLIF\(\$9, 0\), NULLIF\(\$10, 0\), \$11, \$12, \$13, \$14, NULLIF\(\$15, ''\), \$16, NULLIF\(\$17, ''\), \$18\) ON CONFLICT \(isbn\) DO UPDATE SET titel = COALESCE\(NULLIF\(buecher_titel.titel, ''\), EXCLUDED.titel\), .+ ist_lernmittel = buecher_titel.ist_lernmittel OR EXCLUDED.ist_lernmittel RETURNING id`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		mock.ExpectQuery(schreibAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, eintrag.ErweiterteEigenschaften, eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		// Upsert calls legeImportExemplareAn
		codes := erwarteBarcodeVergabe(mock, eintrag.Bestand)
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs([]string{eintrag.ISBN}, []int32{int32(eintrag.Bestand)}, codes).
			WillReturnResult(pgxmock.NewResult("INSERT", 2))

		mock.ExpectCommit()

		id, err := abfragen.SchreibeTitel(ctx, eintrag)
		if err != nil {
			t.Fatalf("abfragen.SchreibeTitel: %v", err)
		}
		if id != "book-123" {
			t.Errorf("id = %q, erwartet %q", id, "book-123")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintrag.Fach)
		mock.ExpectBegin()
		mock.ExpectQuery(schreibAbfrage).
			WithArgs(
				eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, eintrag.Fach, eintrag.Zweig, eintrag.ZuletztGezaehlt, eintrag.Medientyp, eintrag.JahrgangVon, eintrag.JahrgangBis, eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, eintrag.ErweiterteEigenschaften, eintrag.Signatur, eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		_, err = abfragen.SchreibeTitel(ctx, eintrag)
		if err == nil || !strings.Contains(err.Error(), "buch konnte nicht importiert werden") {
			t.Errorf("abfragen.SchreibeTitel: Fehler %v, erwartet einen mit %q", err, "buch konnte nicht importiert werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

func TestSchreibeTitelStapel(t *testing.T) {
	eintraege := []Titel{
		{
			ISBN:                    "123",
			Titel:                   "Title",
			Autor:                   "Author",
			CoverURL:                "URL",
			Fach:                    "Math",
			Zweig:                   "A",
			Bestand:                 2,
			ZuletztGezaehlt:         nil,
			Medientyp:               "Buch",
			ErweiterteEigenschaften: map[string]any{"key": "value"},
			JahrgangVon:             5,
			JahrgangBis:             6,
			Untertitel:              "Subtitle",
			Verlag:                  "Publisher",
			Erscheinungsjahr:        2023,
			Signatur:                "SIG-123",
		},
	}

	batchQuery := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis\) SELECT t.isbn, t.titel, t.autor, t.cover_url, NULLIF\(t.subject, ''\), t.track, NULLIF\(t.last_counted_text, ''\)::date, t.medientyp, NULLIF\(t.jahrgang_von, 0\), NULLIF\(t.jahrgang_bis, 0\), t.untertitel, t.verlag, t.erscheinungsjahr, t.erweiterte_eigenschaften, NULLIF\(t.signatur, ''\), t.ist_lernmittel, NULLIF\(t.auflage, ''\), t.listenpreis FROM UNNEST\(\$1::text\[\], \$2::text\[\], \$3::text\[\], \$4::text\[\], \$5::text\[\], \$6::text\[\], \$7::text\[\], \$8::text\[\], \$9::int\[\], \$10::int\[\], \$11::text\[\], \$12::text\[\], \$13::int\[\], \$14::jsonb\[\], \$15::text\[\], \$16::boolean\[\], \$17::text\[\], \$18::numeric\[\]\) AS t\(isbn, titel, autor, cover_url, subject, track, last_counted_text, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis\) ON CONFLICT \(isbn\) DO UPDATE SET titel = COALESCE\(NULLIF\(buecher_titel.titel, ''\), EXCLUDED.titel\), .+ ist_lernmittel = buecher_titel.ist_lernmittel OR EXCLUDED.ist_lernmittel`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintraege[0].Fach)
		mock.ExpectBegin()

		eigenschaftenJSON, err := json.Marshal(eintraege[0].ErweiterteEigenschaften)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		mock.ExpectExec(batchQuery).
			WithArgs(
				[]string{eintraege[0].ISBN},
				[]string{eintraege[0].Titel},
				[]string{eintraege[0].Autor},
				[]string{eintraege[0].CoverURL},
				[]string{eintraege[0].Fach},
				[]string{eintraege[0].Zweig},
				[]*string{eintraege[0].ZuletztGezaehlt},
				[]string{eintraege[0].Medientyp},
				[]int{eintraege[0].JahrgangVon},
				[]int{eintraege[0].JahrgangBis},
				[]string{eintraege[0].Untertitel},
				[]string{eintraege[0].Verlag},
				[]int{eintraege[0].Erscheinungsjahr},
				[][]byte{eigenschaftenJSON},
				[]string{eintraege[0].Signatur},
				[]bool{eintraege[0].IstLernmittel},
				[]string{eintraege[0].Auflage},
				[]*float64{eintraege[0].Listenpreis},
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Batch Upsert calls legeImportExemplareAn
		codes := erwarteBarcodeVergabe(mock, eintraege[0].Bestand)
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs([]string{eintraege[0].ISBN}, []int32{int32(eintraege[0].Bestand)}, codes).
			WillReturnResult(pgxmock.NewResult("INSERT", 2))

		mock.ExpectCommit()

		betroffen, err := abfragen.SchreibeTitelStapel(ctx, eintraege)
		if err != nil {
			t.Fatalf("abfragen.SchreibeTitelStapel: %v", err)
		}
		if betroffen != int64(1) {
			t.Errorf("betroffen = %v, erwartet %v", betroffen, int64(1))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("empty books", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		betroffen, err := abfragen.SchreibeTitelStapel(ctx, []Titel{})
		if err != nil {
			t.Fatalf("abfragen.SchreibeTitelStapel: %v", err)
		}
		if betroffen != int64(0) {
			t.Errorf("betroffen = %v, erwartet %v", betroffen, int64(0))
		}
	})

	t.Run("db error on batch upsert", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, eintraege[0].Fach)
		mock.ExpectBegin()

		eigenschaftenJSON, err := json.Marshal(eintraege[0].ErweiterteEigenschaften)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		mock.ExpectExec(batchQuery).
			WithArgs(
				[]string{eintraege[0].ISBN},
				[]string{eintraege[0].Titel},
				[]string{eintraege[0].Autor},
				[]string{eintraege[0].CoverURL},
				[]string{eintraege[0].Fach},
				[]string{eintraege[0].Zweig},
				[]*string{eintraege[0].ZuletztGezaehlt},
				[]string{eintraege[0].Medientyp},
				[]int{eintraege[0].JahrgangVon},
				[]int{eintraege[0].JahrgangBis},
				[]string{eintraege[0].Untertitel},
				[]string{eintraege[0].Verlag},
				[]int{eintraege[0].Erscheinungsjahr},
				[][]byte{eigenschaftenJSON},
				[]string{eintraege[0].Signatur},
				[]bool{eintraege[0].IstLernmittel},
				[]string{eintraege[0].Auflage},
				[]*float64{eintraege[0].Listenpreis},
			).
			WillReturnError(fmt.Errorf("db batch failed"))
		mock.ExpectRollback()

		_, err = abfragen.SchreibeTitelStapel(ctx, eintraege)
		if err == nil || !strings.Contains(err.Error(), "bücher konnten nicht im batch importiert werden") {
			t.Errorf("abfragen.SchreibeTitelStapel: Fehler %v, erwartet einen mit %q", err, "bücher konnten nicht im batch importiert werden")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}
