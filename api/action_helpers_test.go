package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleStudentCheckoutFlow(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}
	studentRepo := repository.NewStudentRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)

	copy := &repository.BookCopy{
		ID:            "copy-1",
		BarcodeID:     "B-1234",
		TitelID:       "titel-1",
		IstAusleihbar: true,
	}
	studentID := "student-1"
	staffID := "staff-1"

	// Mock StudentRepo.GetByID
	mock.ExpectQuery("SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR\\(geburtsdatum, 'YYYY-MM-DD'\\), strasse, hausnummer, plz, ort, eltern_email, erstellt_am, aktualisiert_am FROM schueler WHERE id = \\$1 LIMIT 1").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "ist_gesperrt", "lusd_id", "ist_abgaenger", "geburtsdatum", "strasse", "hausnummer", "plz", "ort", "eltern_email", "erstellt_am", "aktualisiert_am"}).
			AddRow(studentID, "123456", "Max", "Mustermann", "10A", nil, false, nil, false, nil, nil, nil, nil, nil, nil, time.Now(), time.Now()))

	// 2. querySettings inside resolveCheckoutDueDate
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", "5").
			AddRow("standard_ausleihfrist_tage", "14"))

	// 3. Mock tx begin
	mock.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})

	// 4. Mock lock on schueler
	mock.ExpectExec("SELECT id FROM schueler WHERE id = \\$1 FOR UPDATE").
		WithArgs(studentID).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	// 5. Mock count active loans
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM ausleihen WHERE schueler_id = \\$1 AND rueckgabe_am IS NULL").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	// 6. Mock GetActiveLoanByCopyIDTx (returns 0 rows -> no active loan)
	mock.ExpectQuery("SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat FROM ausleihen WHERE exemplar_id = \\$1 AND rueckgabe_am IS NULL LIMIT 1 FOR UPDATE").
		WithArgs(copy.ID).
		WillReturnRows(pgxmock.NewRows([]string{}))

	// 7. querySettings inside early limit check
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", "5").
			AddRow("standard_ausleihfrist_tage", "14"))

	// Expected query to check active reservation inside HandleStudentCheckoutFlow
	mock.ExpectQuery("SELECT v.schueler_id, s.vorname, s.nachname FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id WHERE v.bereitgestellt_exemplar_id = \\$1 AND v.status = 'abholbereit' AND v.bereitgestellt_bis > CURRENT_TIMESTAMP").
		WithArgs(copy.ID).
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}))

	// Mock CreateLoanTx
	mock.ExpectQuery("INSERT INTO ausleihen \\(exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\) ON CONFLICT DO NOTHING RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat").
		WithArgs(copy.ID, studentID, pgxmock.AnyArg(), staffID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan-1", &copy.ID, &studentID, nil, time.Now(), time.Now(), nil, staffID, nil, false, false))

	// Mock Commit
	mock.ExpectCommit()

	// Audit Log (runs in its own transaction inside logLoanEvent)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	var resp ActionResponse
	err = server.handleUnifiedCheckoutFlow(context.Background(), copy, &studentID, nil, staffID, studentRepo, loanRepo, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != "ausleihe" {
		t.Errorf("expected response type 'ausleihe', got '%s'", resp.Type)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandleBookReturn(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}
	studentRepo := repository.NewStudentRepository(mock)
	bookRepo := repository.NewBookRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)

	copyID := "copy-1"
	barcode := "B-9999"
	staffID := "staff-1"

	// Mock GetCopyByBarcode
	mock.ExpectQuery("SELECT e\\.id, e\\.titel_id, e\\.barcode_id, coalesce\\(e\\.zustand_notiz, ''\\), e\\.erworben_am, e\\.ist_ausleihbar, e\\.ist_ausgesondert, e\\.erstellt_am, e\\.aktualisiert_am, t\\.titel, coalesce\\(t\\.autor, ''\\), coalesce\\(t\\.verlag, ''\\), coalesce\\(t\\.isbn, ''\\), coalesce\\(t\\.cover_url, ''\\), t\\.medientyp, t\\.erweiterte_eigenschaften FROM buecher_exemplare e JOIN buecher_titel t ON e\\.titel_id = t\\.id WHERE e\\.barcode_id = \\$1 LIMIT 1").
		WithArgs(barcode).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "barcode_id", "zustand_notiz", "erworben_am", "ist_ausleihbar", "ist_ausgesondert", "erstellt_am", "aktualisiert_am", "titel", "autor", "verlag", "isbn", "cover_url", "medientyp", "erweiterte_eigenschaften"}).
			AddRow(copyID, "titel-1", barcode, "", time.Now(), true, false, time.Now(), time.Now(), "Testbuch", "", "", "", "", "", map[string]any{}))

	// Mock Tx
	mock.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})

	// Mock GetActiveLoanByCopyIDTx -> return an active loan
	activeLoanID := "loan-1"
	studentID := "student-1"
	mock.ExpectQuery("SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat FROM ausleihen WHERE exemplar_id = \\$1 AND rueckgabe_am IS NULL LIMIT 1 FOR UPDATE").
		WithArgs(copyID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow(activeLoanID, &copyID, &studentID, nil, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), nil, staffID, nil, false, false))

	// Student lookup fallback
	mock.ExpectQuery("SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, lusd_id, ist_abgaenger, TO_CHAR\\(geburtsdatum, 'YYYY-MM-DD'\\), strasse, hausnummer, plz, ort, eltern_email, erstellt_am, aktualisiert_am FROM schueler WHERE id = \\$1 LIMIT 1").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "ist_gesperrt", "lusd_id", "ist_abgaenger", "geburtsdatum", "strasse", "hausnummer", "plz", "ort", "eltern_email", "erstellt_am", "aktualisiert_am"}).
			AddRow(studentID, "123456", "Max", "Mustermann", "10A", nil, false, nil, false, nil, nil, nil, nil, nil, nil, time.Now(), time.Now()))

	// ReturnLoanTx
	mock.ExpectExec("UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = \\$1, ist_fremdrueckgabe = \\$2 WHERE id = \\$3 AND rueckgabe_am IS NULL").
		WithArgs(staffID, false, activeLoanID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	// Audit Log (runs in its own transaction inside logLoanEvent)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	var resp ActionResponse
	claims := &auth.Claims{UserID: staffID, Rolle: auth.RoleMitarbeiter}

	err = server.handleBookAction(context.Background(), barcode, claims, nil, nil, studentRepo, bookRepo, loanRepo, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != "rueckgabe" {
		t.Errorf("expected type 'rueckgabe', got '%s'", resp.Type)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
