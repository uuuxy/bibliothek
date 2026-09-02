package api

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

// Regressionstest: Die frühere Nanosekunden-Barcode-Generierung kollidierte per
// Geburtstagsparadoxon (10.000 Buckets) ab ~50 Neuzugängen regelmäßig —
// barcode_id ist UNIQUE, der gesamte Import brach ab. Der laufende Zähler macht
// Barcodes innerhalb eines Imports deterministisch eindeutig.
func TestGenerateImportBarcode_UniqueWithinImport(t *testing.T) {
	seen := make(map[string]bool, 5000)
	for i := 1; i <= 5000; i++ {
		b := generateImportBarcode(i)
		if seen[b] {
			t.Fatalf("Barcode-Kollision bei Zähler %d: %s", i, b)
		}
		if !strings.HasPrefix(b, "S-") {
			t.Fatalf("unerwartetes Format: %s", b)
		}
		seen[b] = true
	}
}

func lusdStudentRows(n int) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "klasse", "vorname", "nachname", "lusd_id", "geburtsdatum", "ist_abgaenger", "bestaetigt",
		"schul_eintritt_am", "strasse", "plz", "anonymisiert"})
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		lusdID := "L-" + id // Pointer: die Spalte wird als *string gescannt (nullable)
		rows.AddRow("uuid-"+id, "7A", "Vor"+id, "Nach"+id, &lusdID, (*time.Time)(nil), false, true,
			(*time.Time)(nil), "", "", false)
	}
	return rows
}

// erwarteEinstellungen: Der Lauf liest die Karenzzeit VOR der Transaktion (leere
// Tabelle = Vorgabe); ohne diese Erwartung sähe der Mock eine fremde Abfrage.
func erwarteEinstellungen(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
}

func TestComputeLusdChanges_MassGraduationBlockedBeforeAnyWrite(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	s := &Server{DB: &db.Database{Pool: mock}}

	// 10 aktive Schüler in der DB, CSV enthält nur 2 davon → 8 Abgänger (80%).
	erwarteEinstellungen(mock)
	mock.ExpectBegin()
	// apply=true: der Import-Advisory-Lock wird ZUERST genommen (serialisiert Parallel-Läufe).
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`).
		WillReturnRows(lusdStudentRows(10))
	// KEINE Exec-Erwartungen: Die Bremse muss vor dem ersten destruktiven
	// Statement greifen. Unerwartete Execs ließen den Mock fehlschlagen.
	mock.ExpectRollback()

	records := []parsedStudentRow{
		{LusdID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"},
		{LusdID: "L-b", Vorname: "Vorb", Nachname: "Nachb", Klasse: "8A"},
	}

	_, err = s.computeLusdChanges(t.Context(), records, true, false)
	var massErr *errMassGraduation
	if !errors.As(err, &massErr) {
		t.Fatalf("erwartet errMassGraduation, bekam: %v", err)
	}
	if massErr.Graduates != 8 || massErr.Active != 10 {
		t.Errorf("Zahlen falsch: %+v", massErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestComputeLusdChanges_PreviewNeverWrites(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	s := &Server{DB: &db.Database{Pool: mock}}

	erwarteEinstellungen(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`).
		WillReturnRows(lusdStudentRows(10))
	mock.ExpectRollback()

	records := []parsedStudentRow{
		{LusdID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"}, // Klassenwechsel
		{LusdID: "L-neu", Vorname: "Neu", Nachname: "Kind", Klasse: "5A"}, // Neuzugang
		{Vorname: "Ohne", Nachname: "ID", Klasse: "5A"},                   // ohne LUSD-ID → übersprungen
	}

	res, err := s.computeLusdChanges(t.Context(), records, false, false)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(res.NewStudents) != 1 || len(res.ClassChanges) != 1 || len(res.Graduates) != 9 {
		t.Errorf("Diff falsch: neu=%d wechsel=%d abgaenger=%d", len(res.NewStudents), len(res.ClassChanges), len(res.Graduates))
	}
	if res.ActiveDbStudents != 10 || res.SkippedNoID != 1 {
		t.Errorf("Metadaten falsch: aktiv=%d skipped=%d", res.ActiveDbStudents, res.SkippedNoID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Preview darf nie schreiben: %v", err)
	}
}
