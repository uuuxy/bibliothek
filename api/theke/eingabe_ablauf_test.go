package theke

import (
	"context"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestAusleiheAnDerTheke(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanSvc := service.NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo)

	exemplar := &repository.Exemplar{
		ID:            "copy-1",
		BarcodeID:     "B-1234",
		TitelID:       "titel-1",
		IstAusleihbar: true,
	}
	studentID := "student-1"
	staffID := "staff-1"

	// Der Leser nach seiner Kennung.
	mock.ExpectQuery("SELECT id, coalesce\\(barcode_id, ''\\), coalesce\\(vorname, ''\\), coalesce\\(nachname, ''\\), coalesce\\(klasse, ''\\), coalesce\\(abgaenger_jahr, 0\\), coalesce\\(ist_gesperrt, false\\), lusd_id, coalesce\\(ist_abgaenger, false\\), TO_CHAR\\(geburtsdatum, 'YYYY-MM-DD'\\), erstellt_am, aktualisiert_am, coalesce\\(is_manually_blocked, false\\), block_reason, coalesce\\(strasse, ''\\), coalesce\\(hausnummer, ''\\), coalesce\\(plz, ''\\), coalesce\\(ort, ''\\), coalesce\\(eltern_email, ''\\), anonymized_at IS NOT NULL, art FROM leser WHERE id = \\$1 AND deleted_at IS NULL LIMIT 1").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "ist_gesperrt", "lusd_id", "ist_abgaenger", "geburtsdatum", "erstellt_am", "aktualisiert_am", "is_manually_blocked", "block_reason", "strasse", "hausnummer", "plz", "ort", "eltern_email", "anonymisiert", "art"}).
			AddRow(studentID, "123456", "Max", "Mustermann", "10A", 0, false, nil, false, nil, time.Now(), time.Now(), false, nil, "", "", "", "", "", false, "schueler"))

	// Die Einstellungen für die Frist (ladeEinstellungen in bestimmeFrist).
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", testdaten.ZeigerAuf("5")).
			AddRow("standard_ausleihfrist_tage", testdaten.ZeigerAuf("14")))

	// Die Transaktion beginnt.
	mock.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})

	// Die Zeile des Lesers wird gesperrt.
	mock.ExpectExec("SELECT id FROM leser WHERE id = \\$1 FOR UPDATE").
		WithArgs(studentID).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	// Die offenen Ausleihen des Lesers werden gezählt.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM ausleihen a JOIN buecher_exemplare be ON a\\.exemplar_id = be\\.id JOIN buecher_titel bt ON be\\.titel_id = bt\\.id WHERE a\\.schueler_id = \\$1 AND a\\.rueckgabe_am IS NULL").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	// AusleihAbfragen.SperreOffene findet keine offene Ausleihe des Exemplars.
	mock.ExpectQuery("SELECT id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat FROM ausleihen WHERE exemplar_id = \\$1 AND rueckgabe_am IS NULL LIMIT 1 FOR UPDATE").
		WithArgs(exemplar.ID).
		WillReturnRows(pgxmock.NewRows([]string{}))

	// Sperrgründe erst jetzt, nach dem Lesen der Ausleihe, weil die eigene Rückgabe nie
	// gesperrt ist: offene Schäden (0), Einstellungen, Überfällige (0).
	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COALESCE\\(SUM\\(betrag\\), 0\\) FROM schadensfaelle WHERE schueler_id = \\$1 AND ist_bezahlt = false").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"count", "summe"}).AddRow(0, "0"))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", testdaten.ZeigerAuf("5")).
			AddRow("standard_ausleihfrist_tage", testdaten.ZeigerAuf("14")).
			AddRow("max_overdue_days", testdaten.ZeigerAuf("14")).
			AddRow("max_overdue_items", testdaten.ZeigerAuf("1")))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM ausleihen WHERE schueler_id = \\$1 AND rueckgabe_am IS NULL AND rueckgabe_frist < CURRENT_TIMESTAMP - \\(INTERVAL '1 day' \\* \\$2\\) AND ist_handapparat = false AND geraet_id IS NULL").
		WithArgs(studentID, 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	// Die Einstellungen für die Prüfung der Höchstzahl.
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", testdaten.ZeigerAuf("5")).
			AddRow("standard_ausleihfrist_tage", testdaten.ZeigerAuf("14")))
	// Liegt das Exemplar für jemanden bereit?
	mock.ExpectQuery("SELECT v.schueler_id, s.vorname, s.nachname FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id WHERE v.bereitgestellt_exemplar_id = \\$1 AND v.status = 'abholbereit' AND v.bereitgestellt_bis > CURRENT_TIMESTAMP").
		WithArgs(exemplar.ID).
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}))

	// Die Ausleihe (AusleihAbfragen.LegeAn).
	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(exemplar.ID, studentID, pgxmock.AnyArg(), staffID, false, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan-1", &exemplar.ID, &studentID, time.Now(), time.Now(), nil, &staffID, nil, false, false))
	mock.ExpectExec("UPDATE buecher_exemplare SET letzte_bewegung_am").
		WithArgs(exemplar.ID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Die eigene Vormerkung fällt weg; RETURNING liefert bereitgestellt_exemplar_id für den
	// Hinweis aufs Regal. Hier gibt es keine, also ErrNoRows.
	mock.ExpectQuery("DELETE FROM vormerkungen WHERE titel_id = \\$1 AND schueler_id = \\$2").
		WithArgs(exemplar.TitelID, studentID).
		WillReturnError(pgx.ErrNoRows)

	// Die Zeile im Protokoll in derselben Transaktion, vor dem Commit.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	lr, err := loanSvc.BucheAmLeser(context.Background(), exemplar, &studentID, staffID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.Leser == nil || lr.Leser.ID != studentID {
		t.Errorf("BucheAmLeser(…).Leser = %v, erwartet %s", lr.Leser, studentID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRueckgabeAnDerTheke(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanSvc := service.NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewGeraeteDienst(mock, studentRepo, loanRepo, auditRepo)
	omniboxSvc := service.NewOmniboxDienst(mock, studentRepo, bookRepo, loanRepo, loanSvc, deviceSvc)

	copyID := "copy-1"
	barcode := "B-9999"
	staffID := "staff-1"

	// Das Exemplar zum Barcode (FindeExemplar).
	mock.ExpectQuery("SELECT e\\.id, e\\.titel_id, coalesce\\(e\\.barcode_id, ''\\), coalesce\\(e\\.zustand_notiz, ''\\), e\\.erworben_am, coalesce\\(e\\.ist_ausleihbar, false\\), coalesce\\(e\\.ist_ausgesondert, false\\), e\\.erstellt_am, e\\.aktualisiert_am, coalesce\\(t\\.titel, ''\\), coalesce\\(t\\.autor, ''\\), coalesce\\(t\\.verlag, ''\\), coalesce\\(t\\.isbn, ''\\), coalesce\\(t\\.cover_url, ''\\), coalesce\\(t\\.medientyp, ''\\), coalesce\\(t\\.signatur, ''\\), t\\.mehrjahresband, coalesce\\(t\\.jahrgang_bis, 0\\), t\\.ist_lernmittel, coalesce\\(t\\.erweiterte_eigenschaften, '\\{\\}'::jsonb\\) FROM buecher_exemplare e JOIN buecher_titel t ON e\\.titel_id = t\\.id WHERE e\\.barcode_id = \\$1 OR e\\.lmf_barcode = \\$1 LIMIT 1").
		WithArgs(barcode).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "barcode_id", "zustand_notiz", "erworben_am", "ist_ausleihbar", "ist_ausgesondert", "erstellt_am", "aktualisiert_am", "titel", "autor", "verlag", "isbn", "cover_url", "medientyp", "signatur", "mehrjahresband", "jahrgang_bis", "ist_lernmittel", "erweiterte_eigenschaften"}).
			AddRow(copyID, "titel-1", barcode, "", time.Now(), true, false, time.Now(), time.Now(), "Testbuch", "", "", "", "", "", "", false, 0, false, map[string]any{}))

	// Die Transaktion beginnt.
	mock.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})

	// AusleihAbfragen.SperreOffene findet die offene Ausleihe.
	activeLoanID := "loan-1"
	studentID := "student-1"
	mock.ExpectQuery("SELECT id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat FROM ausleihen WHERE exemplar_id = \\$1 AND rueckgabe_am IS NULL LIMIT 1 FOR UPDATE").
		WithArgs(copyID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow(activeLoanID, &copyID, &studentID, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), nil, &staffID, nil, false, false))

	// Der Leser der Ausleihe.
	mock.ExpectQuery("SELECT id, coalesce\\(barcode_id, ''\\), coalesce\\(vorname, ''\\), coalesce\\(nachname, ''\\), coalesce\\(klasse, ''\\), coalesce\\(abgaenger_jahr, 0\\), coalesce\\(ist_gesperrt, false\\), lusd_id, coalesce\\(ist_abgaenger, false\\), TO_CHAR\\(geburtsdatum, 'YYYY-MM-DD'\\), erstellt_am, aktualisiert_am, coalesce\\(is_manually_blocked, false\\), block_reason, coalesce\\(strasse, ''\\), coalesce\\(hausnummer, ''\\), coalesce\\(plz, ''\\), coalesce\\(ort, ''\\), coalesce\\(eltern_email, ''\\), anonymized_at IS NOT NULL, art FROM leser WHERE id = \\$1 AND deleted_at IS NULL LIMIT 1").
		WithArgs(studentID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "ist_gesperrt", "lusd_id", "ist_abgaenger", "geburtsdatum", "erstellt_am", "aktualisiert_am", "is_manually_blocked", "block_reason", "strasse", "hausnummer", "plz", "ort", "eltern_email", "anonymisiert", "art"}).
			AddRow(studentID, "123456", "Max", "Mustermann", "10A", 0, false, nil, false, nil, time.Now(), time.Now(), false, nil, "", "", "", "", "", false, "schueler"))

	// Die Rückgabe (AusleihAbfragen.BucheRueckgabe).
	mock.ExpectExec("UPDATE ausleihen").
		WithArgs(staffID, false, activeLoanID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec("UPDATE buecher_exemplare e SET letzte_bewegung_am").
		WithArgs(activeLoanID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Zuteilung an eine Vormerkung: Niemand wartet. Ein Fehler dieser Abfrage wird nicht
	// verschluckt, deshalb muss die Attrappe sie ausdrücklich kennen.
	mock.ExpectQuery("SELECT v.id, s.vorname, s.nachname").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	// Die Zeile im Protokoll in derselben Transaktion, vor dem Commit.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	claims := &auth.Claims{KontoID: staffID, Rolle: auth.RolleMitarbeiter}

	res, err := omniboxSvc.VerarbeiteEingabe(context.Background(), service.OmniboxEingabe{Text: barcode, BearbeiterID: claims.KontoID})
	var resp eingabeAntwort
	if res != nil {
		resp = *zurEingabeAntwort(res)
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Art != "rueckgabe" {
		t.Errorf("resp.Art = '%s', erwartet type 'rueckgabe'", resp.Art)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
