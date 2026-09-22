package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// --- Mocks: nur die im Checkout-Pfad genutzten Methoden tragen Logik,
// der Rest erfüllt das Interface als No-op. ---

type mockStudentRepo struct {
	student *repository.Student
	err     error
}

// GetLeserByID ist die Abfrage des Ausleihpfads seit Migration 125: Sie liest die TABELLE
// und findet damit auch einen Kollegen.
func (m *mockStudentRepo) GetLeserByID(ctx context.Context, id string) (*repository.Student, error) {
	return m.student, m.err
}
func (m *mockStudentRepo) GetLeserByBarcode(ctx context.Context, barcode string) (*repository.Student, error) {
	return nil, nil
}
func (m *mockStudentRepo) SearchStudentsFuzzy(ctx context.Context, q string, limit int) ([]repository.Student, int, error) {
	return nil, 0, nil
}
func (m *mockStudentRepo) HasPhoto(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *mockStudentRepo) HasOpenDamages(ctx context.Context, id string) (bool, error) {
	return false, nil
}
func (m *mockStudentRepo) GetActiveBorrowedBooks(ctx context.Context, id string) ([]repository.BorrowedBook, error) {
	return nil, nil
}
func (m *mockStudentRepo) GetDistinctClasses(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockStudentRepo) EtikettenZeilen(ctx context.Context, ids []string) ([]repository.SchuelerEtikettZeile, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListStudentsWithStats(ctx context.Context, klassen []string, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListLeserMitStats(ctx context.Context, klassen []string, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListEhemaligeWithStats(ctx context.Context, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}

type mockAuditRepo struct {
	adminAktionCalls int
}

func (m *mockAuditRepo) LogAdminAktion(ctx context.Context, adminID, aktion, ip string, details map[string]any) error {
	m.adminAktionCalls++
	return nil
}
func (m *mockAuditRepo) DeleteTitle(ctx context.Context, t, b string) error           { return nil }
func (m *mockAuditRepo) DeleteCopy(ctx context.Context, c, b string) error            { return nil }
func (m *mockAuditRepo) DeleteUser(ctx context.Context, u, b string) error            { return nil }
func (m *mockAuditRepo) DeleteStudent(ctx context.Context, s, b, g string) error      { return nil }
func (m *mockAuditRepo) PurgeStudent(ctx context.Context, s, b string) error          { return nil }
func (m *mockAuditRepo) PurgeAbgaenger(ctx context.Context, s, b string) error        { return nil }
func (m *mockAuditRepo) StornierungGebuehr(ctx context.Context, s, b, g string) error { return nil }
func (m *mockAuditRepo) BezahltGebuehr(ctx context.Context, s, b string) error        { return nil }
func (m *mockAuditRepo) LogAusleihe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) LogRueckgabe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) LogSystemAktion(ctx context.Context, tabelle, aktion, kontext string, details map[string]any) error {
	return nil
}

func strPtr(s string) *string { return &s }

// expectSettingsAndOverdue richtet die Mock-Erwartungen für einen Durchlauf der
// Sperrprüfung ohne Sperre ein: offene Schäden → querySettings → Overdue-Zählung.
func expectSettingsAndOverdue(mock pgxmock.PgxPoolIface, overdueCount int) {
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", "1").
			AddRow("max_overdue_days", "14"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(overdueCount))
}

// expectFristSettings: die eine Einstellungsabfrage, mit der resolveCheckoutDueDate die
// Leihfrist bestimmt.
func expectFristSettings(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("standard_ausleihfrist_tage", "14"))
}

func newValidationService(t *testing.T, student *repository.Student) (*defaultLoanService, *mockAuditRepo, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	audit := &mockAuditRepo{}
	svc := &defaultLoanService{
		pool:        mock,
		studentRepo: &mockStudentRepo{student: student},
		auditRepo:   audit,
	}
	return svc, audit, mock
}

func activeStudent(id string) *string { return &id }

// --- Tests ---

// Die Sperrprüfung hängt seit 11.09.2026 nicht mehr am Auflösen des Ausleihers:
// HandleUnifiedCheckout ruft sie erst, wenn feststeht, dass es keine eigene Rückgabe ist
// (api/theke_eigene_rueckgabe_pg_test.go). Die Fälle hier prüfen sie deshalb direkt.

func TestSperrpruefung_GesperrterSchuelerAbgewiesen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IstGesperrt: true}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestSperrpruefung_ManuelleSperreAbgewiesen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IsManuallyBlocked: true, BlockReason: strPtr("Buch verloren")}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("manuell gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestSperrpruefung_UebergehenLaesstDurchUndProtokolliert(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IstGesperrt: true}
	svc, audit, mock := newValidationService(t, schueler)
	defer mock.Close()

	expectSettingsAndOverdue(mock, 0)

	if err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", true, false); err != nil {
		t.Fatalf("override soll Sperre umgehen, bekam Fehler: %v", err)
	}
	if audit.adminAktionCalls < 1 {
		t.Errorf("override muss als Admin-Aktion auditiert werden, calls=%d", audit.adminAktionCalls)
	}
}

func TestSperrpruefung_UeberfaelligSperrtAutomatisch(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Nicht manuell gesperrt, keine offenen Schäden, aber 2 überfällige Medien bei MaxOverdueItems=1.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("max_overdue_items", "1"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("überfällige Medien über Limit sollen automatisch sperren, bekam: %v", err)
	}
}

func TestSperrpruefung_OffenerSchadenSperrt(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Kein Sperr-Flag, aber ein offener (unbezahlter) Schadensfall -> automatische Sperre,
	// noch bevor Settings/Overdue überhaupt abgefragt werden.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("offener Schadensfall soll automatisch sperren, bekam: %v", err)
	}
}

func TestSperrpruefung_OffenerSchadenUebergangen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, audit, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Offener Schadensfall, aber overrideBlock=true: Ausleihe geht durch, wird auditiert.
	// Danach laufen Settings/Overdue wie im Happy-Path (0 überfällig).
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", "1").AddRow("max_overdue_days", "14"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	if err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", true, false); err != nil {
		t.Fatalf("override soll offene Schäden umgehen, bekam Fehler: %v", err)
	}
	if audit.adminAktionCalls < 1 {
		t.Errorf("override muss als Admin-Aktion auditiert werden, calls=%d", audit.adminAktionCalls)
	}
}

// Alle vier Sperren der Buch-Ausleihe lassen sich übergehen (overrideBlock) und tragen
// deshalb das Merkmal, an dem die Theke den Override-Dialog öffnet (X-Sperre in
// api/action.go). Bis zum 13.09.2026 entschied dort der Wortlaut der Meldung, und die
// Schadens-Sperre bekam keinen Dialog. Geräte kennen kein Override und tragen es nicht.
func TestSperrpruefung_UebergehbareSperrenSindMarkiert(t *testing.T) {
	pruefe := func(t *testing.T, schueler *repository.Student, vorbereiten func(pgxmock.PgxPoolIface)) {
		t.Helper()
		svc, _, mock := newValidationService(t, schueler)
		defer mock.Close()
		vorbereiten(mock)
		err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, false)
		if !errors.Is(err, ErrBlocked) {
			t.Fatalf("erwartet ErrBlocked, bekam %v", err)
		}
		if !IstUebergehbareSperre(err) {
			t.Errorf("übergehbare Sperre ohne Merkmal — die Theke bekommt keinen Override-Dialog: %v", err)
		}
	}

	t.Run("System-Sperre", func(t *testing.T) {
		pruefe(t, &repository.Student{ID: "s1", IstGesperrt: true, BlockReason: strPtr("Abgänger")},
			func(pgxmock.PgxPoolIface) {})
	})
	t.Run("manuelle Sperre", func(t *testing.T) {
		pruefe(t, &repository.Student{ID: "s1", IsManuallyBlocked: true, BlockReason: strPtr("Buch verloren")},
			func(pgxmock.PgxPoolIface) {})
	})
	t.Run("offener Schaden", func(t *testing.T) {
		pruefe(t, &repository.Student{ID: "s1"}, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
				WithArgs("s1").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		})
	})
	t.Run("überfällige Medien", func(t *testing.T) {
		pruefe(t, &repository.Student{ID: "s1"}, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
				WithArgs("s1").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("max_overdue_items", "1"))
			mock.ExpectQuery("SELECT COUNT").
				WithArgs("s1", 14).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		})
	})
	t.Run("Gegenprobe: gesperrtes Gerät trägt kein Merkmal", func(t *testing.T) {
		err := pruefeGeraetAusleihbar(repository.Geraet{IstAusleihbar: false})
		if !errors.Is(err, ErrBlocked) || IstUebergehbareSperre(err) {
			t.Errorf("Geräte-Sperre: erwartet ErrBlocked ohne Merkmal, bekam %v (Merkmal=%v)", err, IstUebergehbareSperre(err))
		}
	})
}

func TestResolveBorrower_HappyPath(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann", Art: "schueler",
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"))

	if err != nil {
		t.Fatalf("regulärer Schüler ohne Sperre soll durchgehen, bekam: %v", err)
	}
	if !ctx.istSchueler() || ctx.borrowerID != "s1" {
		t.Errorf("erwartete den Schüler s1, bekam Art %q / %q", ctx.leser.Art, ctx.borrowerID)
	}
	if ctx.dueTime.IsZero() {
		t.Error("erwartete gesetztes Fälligkeitsdatum")
	}
}

// Auflösen sperrt nicht: Ein gesperrter Schüler wird aufgelöst, die Sperre prüft der
// Checkout danach. Sperrte schon das Auflösen, wäre die eigene Rückgabe wieder gesperrt.
func TestResolveBorrower_SperrtNicht(t *testing.T) {
	svc, audit, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Art: "schueler", IstGesperrt: true, BlockReason: strPtr("überfällig"),
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"))

	if err != nil {
		t.Fatalf("Auflösen darf nicht sperren, bekam: %v", err)
	}
	if ctx.leser == nil || !ctx.leser.IstGesperrt {
		t.Error("erwartete den gesperrten Leser im checkoutContext")
	}
	if audit.adminAktionCalls != 0 {
		t.Errorf("Auflösen darf kein Übergehen protokollieren, calls=%d", audit.adminAktionCalls)
	}
}

func TestResolveBorrower_NoActiveBorrower(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, nil)

	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("ohne aktiven Leser soll ErrInvalidState kommen, bekam: %v", err)
	}
}

// Antwort der Schule vom 22.09.2026 (docs/OFFEN.md 9.3 c): Für ein Lernmittel gibt es keine
// automatische Abweisung — weder wegen einer offenen Forderung noch über die
// Überfällig-Automatik, und auch keine übergehbare. Der Mock erwartet KEINE Abfrage: Zählt
// eine der zwei Automatiken doch, meldet pgxmock die unerwartete Query als Fehler.
// Rot gesehen am Rückbau der Weiche (22.09.2026).
func TestSperrpruefung_LernmittelOhneAutomatik(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, audit, mock := newValidationService(t, schueler)
	defer mock.Close()

	if err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, true); err != nil {
		t.Fatalf("Lernmittel: die Automatik darf nicht abweisen, bekam %v", err)
	}
	if audit.adminAktionCalls != 0 {
		t.Errorf("nichts wurde übergangen, trotzdem %d Einträge protokolliert", audit.adminAktionCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// Die zwei Schalter am Leser gelten auch beim Lernmittel: Sie sind die Entscheidung eines
// Menschen (oder des Abgänger-Verfahrens), keine Automatik.
func TestSperrpruefung_LernmittelSchalterBleiben(t *testing.T) {
	faelle := map[string]*repository.Student{
		"gesperrt": {ID: "s1", Klasse: "5a", IstGesperrt: true, BlockReason: strPtr("Test")},
		"von Hand": {ID: "s1", Klasse: "5a", IsManuallyBlocked: true, BlockReason: strPtr("Test")},
	}
	for name, schueler := range faelle {
		t.Run(name, func(t *testing.T) {
			svc, _, mock := newValidationService(t, schueler)
			defer mock.Close()

			err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false, true)
			if !errors.Is(err, ErrBlocked) {
				t.Errorf("Schalter am Leser muss auch beim Lernmittel abweisen, bekam %v", err)
			}
		})
	}
}
