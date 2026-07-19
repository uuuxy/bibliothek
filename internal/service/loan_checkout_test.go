package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// expectSettings bedient die system_einstellungen-Abfrage aus querySettings.
func expectSettings(mock pgxmock.PgxPoolIface, maxAusleihen string) {
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", maxAusleihen))
}

func schuelerCtx(id string) *checkoutContext {
	return &checkoutContext{borrowerType: "student", borrowerID: id}
}

func buchMitTitel(titel string) *repository.BookCopy {
	return &repository.BookCopy{Titel: titel, Medientyp: "Buch", IstAusleihbar: true}
}

// --- Ausleihlimit (max_ausleihen_schueler) ---

func TestPruefeAusleihlimit_ErreichtesLimitBlockiert(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 5, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("erreichtes Limit (5 von 5) soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_UnterLimitErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 4, false)

	if err != nil {
		t.Errorf("4 von 5 Ausleihen soll erlaubt sein, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_LMFBuchIstAusgenommen(t *testing.T) {
	// Lernmittelfreiheit-Bücher zählen nicht gegen das Limit — sonst könnte ein
	// Schüler mit vollem Klassensatz keine regulären Bücher mehr leihen.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("LMF-Mathematik 7"), 10, false)

	if err != nil {
		t.Errorf("LMF-Buch soll trotz überschrittenem Limit durchgehen, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_EigeneRueckgabeIstAusgenommen(t *testing.T) {
	// Beim Zurückgeben des eigenen Buchs darf das Limit nicht blockieren.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 10, true)

	if err != nil {
		t.Errorf("eigene Rückgabe soll trotz Limit durchgehen, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_LehrerOhneLimit(t *testing.T) {
	// Für Lehrkräfte (Handapparat) gilt kein Limit — die Regel greift vor jeder
	// Settings-Abfrage, daher ist hier bewusst KEINE Query zu erwarten.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	lehrerCtx := &checkoutContext{borrowerType: "teacher", borrowerID: "l1"}
	err := svc.pruefeSchuelerAusleihlimit(context.Background(), lehrerCtx, buchMitTitel("Der Hobbit"), 99, false)

	if err != nil {
		t.Errorf("Lehrkraft soll kein Ausleihlimit haben, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("für Lehrkräfte darf keine Settings-Abfrage laufen: %v", err)
	}
}

// --- Race-Condition-Übersetzung (Unique-Index aus Migration 033) ---

func TestMapLoanCreateErr(t *testing.T) {
	// Der partielle Unique-Index verhindert zwei aktive Ausleihen desselben
	// Exemplars. Greifen zwei Terminals gleichzeitig zu, muss daraus ein sauberer
	// Konflikt werden (409) statt eines 500ers.
	uniqueViolation := &pgconn.PgError{Code: "23505"}
	fkViolation := &pgconn.PgError{Code: "23503"}
	generic := errors.New("connection reset")

	tests := []struct {
		name       string
		in         error
		wantConfl  bool
		wantSameAs error
	}{
		{"Unique-Verletzung wird Konflikt", uniqueViolation, true, nil},
		{"andere PG-Fehler bleiben unveraendert", fkViolation, false, fkViolation},
		{"generische Fehler bleiben unveraendert", generic, false, generic},
		{"nil bleibt nil", nil, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapLoanCreateErr(tt.in)

			if tt.wantConfl {
				if !errors.Is(got, ErrConflict) {
					t.Fatalf("erwartete ErrConflict, bekam: %v", got)
				}
				return
			}
			if got != tt.wantSameAs {
				t.Errorf("Fehler soll unveraendert durchgereicht werden: got %v, want %v", got, tt.wantSameAs)
			}
		})
	}
}

// --- Lehrkraft als Entleiher (Handapparat) ---

// Die Rolle kommt aus benutzer.rolle — NICHT mehr aus benutzer_rollen (siehe
// Regressionstest unten).
const lehrerQuery = "SELECT b.id, b.barcode_id, b.vorname, b.nachname, b.rolle::text"

func TestResolveTeacher_AktiveLehrkraftBekommtJahresfrist(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	mock.ExpectQuery(lehrerQuery).WithArgs("l1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "rolle"}).
			AddRow("l1", "B-L1", "Anna", "Lehrerin", "lehrer"))

	chkCtx, err := svc.resolveTeacherBorrower(context.Background(), "l1")

	if err != nil {
		t.Fatalf("aktive Lehrkraft soll aufgelöst werden, bekam: %v", err)
	}
	if chkCtx.borrowerType != "teacher" || chkCtx.borrowerID != "l1" {
		t.Errorf("erwartete teacher/l1, bekam %q/%q", chkCtx.borrowerType, chkCtx.borrowerID)
	}
	// Handapparat: Dauerleihgabe über ein Jahr (nicht die Schüler-Frist), auf das Tagesende
	// der Schul-Zeitzone normalisiert — einheitlich mit allen anderen Fristen.
	erwarteterTag := time.Now().In(schoolLocation()).AddDate(1, 0, 0)
	if !sameDay(chkCtx.dueTime, erwarteterTag) {
		t.Errorf("Leihfrist soll ~1 Jahr betragen, bekam %v (erwarteter Tag ~%v)", chkCtx.dueTime, erwarteterTag)
	}
	if h, m, sec := chkCtx.dueTime.In(schoolLocation()).Clock(); h != 23 || m != 59 || sec != 59 {
		t.Errorf("Frist soll auf das Tagesende (23:59:59) fallen, bekam %02d:%02d:%02d", h, m, sec)
	}
}

func TestResolveTeacher_UnbekannteOderInaktiveLehrkraft(t *testing.T) {
	// Deckt zugleich Nicht-Lehrer und inaktive Konten ab: die Query filtert
	// br.rolle='LEHRER' AND b.aktiv=true, liefert also schlicht keine Zeile.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	mock.ExpectQuery(lehrerQuery).WithArgs("l9").WillReturnError(pgx.ErrNoRows)

	_, err := svc.resolveTeacherBorrower(context.Background(), "l9")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekannte/inaktive Lehrkraft soll ErrNotFound liefern, bekam: %v", err)
	}
}

// TestResolveTeacher_FragtNichtBenutzerRollen ist der Regressionstest fuer den
// Handapparat-Bug: benutzer_rollen wird nur beim Bootstrap einmalig befuellt, das
// Admin-UI schreibt beim Anlegen ausschliesslich benutzer.rolle. Der frühere
// INNER JOIN auf benutzer_rollen liess deshalb JEDE neu angelegte Lehrkraft
// auflaufen ("Aktives Lehrerprofil nicht gefunden"). Die Abfrage darf diese
// Tabelle nicht mehr berühren.
func TestResolveTeacher_FragtNichtBenutzerRollen(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	// Erwartet wird exakt EINE Abfrage — ohne benutzer_rollen. Ein Join-Rückfall
	// würde hier an der nicht erfüllten Erwartung scheitern.
	mock.ExpectQuery("FROM benutzer b\\s+WHERE b.id = \\$1 AND LOWER\\(b.rolle::text\\) = 'lehrer'").
		WithArgs("neu1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "vorname", "nachname", "rolle"}).
			AddRow("neu1", "B-N1", "Neue", "Lehrkraft", "lehrer"))

	chkCtx, err := svc.resolveTeacherBorrower(context.Background(), "neu1")

	if err != nil {
		t.Fatalf("neu angelegte Lehrkraft (ohne benutzer_rollen-Zeile) muss ausleihen koennen, bekam: %v", err)
	}
	if chkCtx.borrowerID != "neu1" {
		t.Errorf("erwartete borrowerID neu1, bekam %q", chkCtx.borrowerID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Abfrage entspricht nicht der Erwartung (benutzer_rollen-Join zurueck?): %v", err)
	}
}

// --- Aktive Ausleihen zählen (Basis fürs Limit) ---

func TestZaehleAktiveAusleihen_NichtSchuelerZaehltNull(t *testing.T) {
	// Für Lehrkräfte gilt kein Limit — es darf gar nicht gezählt werden.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	count, err := svc.zaehleAktiveSchuelerAusleihen(context.Background(), tx,
		&checkoutContext{borrowerType: "teacher", borrowerID: "l1"})

	if err != nil || count != 0 {
		t.Errorf("Nicht-Schüler soll 0 ohne Fehler liefern, bekam %d / %v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("für Lehrkräfte darf keine Zähl-Abfrage laufen: %v", err)
	}
}

func TestZaehleAktiveAusleihen_SchuelerWirdGesperrtUndGezaehlt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	// Row-Level-Lock gegen parallele Scans, dann Zählung ohne LMF-Titel.
	mock.ExpectExec("SELECT id FROM schueler WHERE id = \\$1 FOR UPDATE").
		WithArgs("s1").WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery("SELECT COUNT").WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))

	count, err := svc.zaehleAktiveSchuelerAusleihen(context.Background(), tx, schuelerCtx("s1"))

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if count != 3 {
		t.Errorf("erwartete 3 aktive Ausleihen, bekam %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Lock + Zählung müssen laufen: %v", err)
	}
}

// --- Vormerkungs-Konflikt (abholbereit reserviertes Exemplar) ---

// beginTx startet eine gemockte Transaktion für die Konflikt-Prüfung.
func beginTx(t *testing.T, mock pgxmock.PgxPoolIface) pgx.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	return tx
}

const vormerkungQuery = "SELECT v.schueler_id, s.vorname, s.nachname"

func TestVormerkungKonflikt_FremdeReservierungBlockiert(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}).
			AddRow("s2", "Erika", "Musterfrau"))

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if !errors.Is(err, ErrConflict) {
		t.Errorf("für s2 reserviertes Exemplar darf nicht an s1 gehen, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_EigeneReservierungErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}).
			AddRow("s1", "Max", "Mustermann"))

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if err != nil {
		t.Errorf("eigene Reservierung soll abholbar sein, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_OhneReservierungErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").WillReturnError(pgx.ErrNoRows)

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if err != nil {
		t.Errorf("ohne Reservierung soll die Ausleihe durchgehen, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_BeiRueckgabeKeinePruefung(t *testing.T) {
	// Eine Rückgabe darf nie an einer Reservierung scheitern — es wird bewusst
	// gar keine Abfrage erwartet.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), true)

	if err != nil {
		t.Errorf("Rückgabe soll nicht geprüft werden, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("bei Rückgabe darf keine Vormerkungs-Abfrage laufen: %v", err)
	}
}

// --- Eigene Rückgabe vs. Fremdrückgabe ---

func TestIstEigeneRueckgabe(t *testing.T) {
	tests := []struct {
		name     string
		chkCtx   *checkoutContext
		loan     *repository.Loan
		expected bool
	}{
		{"keine aktive Ausleihe", schuelerCtx("s1"), nil, false},
		{
			"Schüler scannt eigenes Buch",
			schuelerCtx("s1"),
			&repository.Loan{SchuelerID: strPtr("s1")},
			true,
		},
		{
			"Schüler scannt fremdes Buch",
			schuelerCtx("s1"),
			&repository.Loan{SchuelerID: strPtr("s2")},
			false,
		},
		{
			"Lehrkraft scannt eigenes Buch",
			&checkoutContext{borrowerType: "teacher", borrowerID: "l1"},
			&repository.Loan{AusleiherBenutzerID: strPtr("l1")},
			true,
		},
		{
			"Lehrkraft scannt fremdes Buch",
			&checkoutContext{borrowerType: "teacher", borrowerID: "l1"},
			&repository.Loan{AusleiherBenutzerID: strPtr("l2")},
			false,
		},
		{
			"Schüler scannt Buch einer Lehrkraft",
			schuelerCtx("s1"),
			&repository.Loan{AusleiherBenutzerID: strPtr("l1")},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := istEigeneRueckgabe(tt.chkCtx, tt.loan); got != tt.expected {
				t.Errorf("istEigeneRueckgabe() = %v; want %v", got, tt.expected)
			}
		})
	}
}
