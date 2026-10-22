package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

// snapshotTitelRows liefert die Zeile, die LoescheTitel vor dem Löschen für das Protokoll
// liest.
func snapshotTitelRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"titel", "autor", "isbn"}).
		AddRow("Der Vorleser", "Bernhard Schlink", "9783161484100")
}

// Ein Titel mit verliehenen Exemplaren wurde nie gelöscht; unbrauchbar war die Rückmeldung. Die
// Tür erkannte den Fall am Anfang des Fehlertexts und traf nie, jeder abgelehnte Versuch endete
// als Serverfehler, und die Liste der verliehenen Barcodes ging verloren. Deshalb ein
// Fehlerwert: errors.Is übersteht jede Umformulierung.
func TestLoescheTitel_MeldetAktiveAusleihenAlsSentinel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	const titelID = "11111111-1111-1111-1111-111111111111"

	mock.ExpectBegin()
	erwarteAuflagenSperre(mock, []string{titelID})
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).
			AddRow("B-00042").AddRow("B-00043"))
	// Kein ExpectExec("DELETE …"): Der Abbruch kommt vor jeder Anweisung, die etwas löscht.
	mock.ExpectRollback()

	err = NewKatalogAbfragen(mock).LoescheTitel(context.Background(), titelID, "admin-1")

	var mitAusleihen *TitelHatAusleihenFehler
	if !errors.As(err, &mitAusleihen) {
		t.Fatalf("LoescheTitel = %v, erwartet TitelHatAusleihenFehler (die Tür antwortet darauf mit 400)", err)
	}
	// Die Barcodes sind der Grund, warum diese Meldung überhaupt zum Nutzer soll.
	if diff := cmp.Diff([]string{"B-00042", "B-00043"}, mitAusleihen.Barcodes); diff != "" {
		t.Errorf("LoescheTitel: Barcodes (-erwartet +ist):\n%s", diff)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// Die Gegenprobe: Ohne offene Ausleihen greift die Sperre nicht, sonst ließe sich nie ein Titel
// entfernen.
func TestLoescheTitel_LoeschtOhneAktiveAusleihen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	const titelID = "22222222-2222-2222-2222-222222222222"

	mock.ExpectBegin()
	erwarteAuflagenSperre(mock, []string{titelID})
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"})) // keine offenen Ausleihen
	// Die Barcodes der Exemplare vor dem Löschen: Die Theken-Auskunft findet gelöschte
	// Exemplare nur über diese Spur.
	mock.ExpectQuery("FROM buecher_exemplare e WHERE e.titel_id").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "lmf_barcode", "war_im_bestand"}).
			AddRow("ex-1", "B-00001", "", true))
	// Offene Forderungen vor dem Löschen ins Protokoll: Ein unbezahlter Schadensfall ist Geld,
	// das ein Leser schuldet, und verschwände sonst mit dem Titel.
	mock.ExpectQuery("FROM schadensfaelle sf").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "barcode_id", "schuldner",
			"schueler_id", "betrag", "beschreibung", "erstellt_am"}).
			AddRow("sf-1", "ex-1", "B-00001", "Hans Castorp", ptrString("s-1"), "3.00", "Fleck", "2026-05-01"))
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ex-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Was mit dem Titel über CASCADE fällt: Vormerkungen, Klassensatz-Reservierungen und
	// Klassensatz-Zuordnungen. Hier eine wartende Vormerkung, damit auch ihre Zeile im
	// Protokoll erwartet wird.
	mock.ExpectQuery("FROM vormerkungen v").
		WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "wer", "status", "seit", "schueler_id"}).
			AddRow("vm-1", titelID, "Der Zauberberg", "Hans Castorp", "wartend", "2026-05-01T10:00:00+02", ptrString("s-1")))
	mock.ExpectQuery("FROM klassensatz_reservierungen r").
		WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse", "status", "seit"}))
	mock.ExpectQuery("FROM class_books c").
		WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse"}))
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("vormerkungen", titelID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Das lokale Cover des Titels, vor den DELETEs gelesen und nach dem Commit entfernt.
	mock.ExpectQuery("SELECT t.cover_url FROM buecher_titel t").
		WithArgs([]string{titelID}).
		WillReturnRows(pgxmock.NewRows([]string{"cover_url"}))
	mock.ExpectExec("DELETE FROM schadensfaelle").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM ausleihen").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM buecher_exemplare").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))
	mock.ExpectExec("DELETE FROM buecher_titel").WithArgs(titelID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	// Vom Eintrag sind Tabelle, Aktion und Datensatz festgelegt, der Rest (Bearbeiter, Kontext,
	// Details) nicht.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("buecher_titel", "DELETE", titelID,
			pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Je Exemplar der Eintrag mit Barcode in der Form der Geschwister (SondereExemplarAus,
	// Verlust).
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("buecher_exemplare", "DELETE", "ex-1",
			pgxmock.AnyArg(), "USER", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := NewKatalogAbfragen(mock).LoescheTitel(context.Background(), titelID, "admin-1"); err != nil {
		t.Fatalf("Löschen ohne offene Ausleihen muss durchgehen, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// Bricht die Abfrage der Ausleihen mittendrin ab, gilt nicht „keine offenen Ausleihen": Sonst
// löschte ein Fehler der Verbindung den Titel samt verliehener Exemplare.
func TestLoescheTitel_LeseFehlerBlocktLoeschung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	const titelID = "33333333-3333-3333-3333-333333333333"
	leseFehler := errors.New("verbindung während der iteration verloren")

	mock.ExpectBegin()
	erwarteAuflagenSperre(mock, []string{titelID})
	mock.ExpectQuery("FROM buecher_titel WHERE id").
		WithArgs(titelID).
		WillReturnRows(snapshotTitelRows())
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).
			AddRow("B-00042").RowError(0, leseFehler))
	// Kein ExpectExec: Bei unklarer Lage wird nichts gelöscht.
	mock.ExpectRollback()

	err = NewKatalogAbfragen(mock).LoescheTitel(context.Background(), titelID, "admin-1")
	if err == nil {
		t.Fatal("ein Lesefehler darf nicht als 'keine offenen Ausleihen' durchgehen")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// ptrString: pgxmock reicht Zeiger unverändert durch; die Spalte schueler_id ist nullbar.
func ptrString(s string) *string { return &s }

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

// erwarteAuflagenSperre: Das Löschen nimmt zuerst die Sperre der Auflagen und liest die Werke
// des Titels (LadeWerkeDerTitel). Hier gehört er zu keinem, also räumt RaeumeWerkeAuf danach nichts
// auf.
func erwarteAuflagenSperre(mock pgxmock.PgxPoolIface, ids []string) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(auflagenSperre).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT DISTINCT werk_id::text FROM buecher_titel`).WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"werk_id"}))
}
