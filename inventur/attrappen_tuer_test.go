package inventur

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v5"
)

// Erwartungen an die Datenbank für die Tests der Türen, solange die Türen und ihre Abfragen in
// verschiedenen Paketen stehen.

// errTestTuer ist ein beliebiger Fehler der Datenbank in den Tests der Türen.
var errTestTuer = errors.New("test error")

// erwarteAuflagenSperreTuer: LoescheTitel nimmt als Erstes die Sperre der Auflagen und liest die
// Werke der Titel (repository.LadeWerkeDerTitel); hier gehört keiner zu einem, also räumt
// repository.RaeumeWerkeAuf danach nichts auf.
func erwarteAuflagenSperreTuer(mock pgxmock.PgxPoolIface, ids any) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT DISTINCT werk_id::text FROM buecher_titel`).WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"werk_id"}))
}

// erwarteBarcodeVergabeTuer erwartet die zwei Abfragen von repository.ZieheFreieExemplarBarcodes
// (nextval aus barcode_seq, dann der Abgleich mit dem Bestand ohne Treffer) und liefert die
// Nummern, die der Schreibweg danach einfügt. Korrektur des Bestands und Sammelimport ziehen aus
// barcode_seq.
func erwarteBarcodeVergabeTuer(mock pgxmock.PgxPoolIface, anzahl int) []string {
	codes := make([]string, anzahl)
	zeilen := pgxmock.NewRows([]string{"nextval"})
	for i := range codes {
		codes[i] = fmt.Sprintf("B-%05d", 10001+i)
		zeilen.AddRow(int64(10001 + i))
	}
	mock.ExpectQuery(`nextval\('barcode_seq'\)`).WithArgs(anzahl).WillReturnRows(zeilen)
	mock.ExpectQuery(`SELECT barcode_id FROM buecher_exemplare WHERE barcode_id = ANY`).
		WithArgs(codes).WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}))
	return codes
}

// erwarteFachBekanntTuer spielt den Nachschlag der Fach-Registrierung nach: Das Fach ist
// bereits in systematik_kategorien registriert, der Schreibpfad bekommt die kanonische
// Bezeichnung zurück. Jeder pgxmock-Test eines subject-Schreibers braucht diese
// Erwartung vor seinem eigentlichen Statement (siehe StelleFaecherSicher).
func erwarteFachBekanntTuer(mock pgxmock.PgxPoolIface, fach string) {
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs(fach).
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow(fach))
}

// erwarteKeineDubletteTuer: Die Dublettenkontrolle fragt vor dem Anlegen und vor dem Ändern
// der ISBN einmal nach dem Titel, der die Nummer trägt. „Nichts gefunden" ist der Normalfall;
// die Mock-Tests prüfen den Schreibpfad, die Kontrolle hat ihren Test am echten Postgres.
func erwarteKeineDubletteTuer(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT bt\.id::text, bt\.titel, EXISTS .+ FROM buecher_titel`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)
}

// erwarteTitelstandTuer: Vor jedem Ändern liest der Schreibpfad, was am Titel steht
// (pruefeAenderung). Trägt er die ISBN der Änderung, folgt keine Dublettenkontrolle.
func erwarteTitelstandTuer(mock pgxmock.PgxPoolIface, isbn, autor string) {
	mock.ExpectQuery(sqlTitelstandMusterTuer).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"isbn", "autor", "ist_lernmittel", "mehrjahresband", "jahrgang_von", "jahrgang_bis"}).
			AddRow(isbn, autor, false, false, 5, 10))
}

// bearbeiterFuerAussonderungTuer legt das Konto an, das ein kleinerer Bestand im Protokoll nennt,
// und räumt es nach dem Test mit seinen Einträgen wieder ab.
func bearbeiterFuerAussonderungTuer(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Bestand', 'Korrektur', 'bestandskorrektur@schule.invalid', 'mitarbeiter', true)
		ON CONFLICT (lower(email)) DO UPDATE SET aktiv = true
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM audit_log WHERE bearbeiter_id = $1`,
			`DELETE FROM benutzer WHERE id = $1`,
		} {
			if _, err := pool.Exec(context.Background(), sql, id); err != nil {
				t.Errorf("Aufräumen Bearbeiter: %v", err)
			}
		}
	})
	return id
}

// sqlTitelstandMusterTuer ist die Abfrage, mit der pruefeAenderung den Titel sperrt und liest.
const sqlTitelstandMusterTuer = `SELECT COALESCE\(isbn, ''\), COALESCE\(autor, ''\), ist_lernmittel, mehrjahresband, COALESCE\(jahrgang_von, 0\), COALESCE\(jahrgang_bis, 0\) FROM buecher_titel WHERE id = \$1 FOR UPDATE`
