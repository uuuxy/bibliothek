package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// sameDay vergleicht zwei Zeitpunkte auf Kalendertag-Ebene (Jahr/Monat/Tag),
// unabhängig von Uhrzeit und Zeitzone-Offset.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// --- parseGrade: reine Jahrgangs-Extraktion (kritisch für mehrjährige LMF-Ausleihen) ---

func TestParseGrade(t *testing.T) {
	cases := []struct {
		klasse string
		want   int
	}{
		{"5a", 5},
		{"10R", 10},
		{"7", 7},
		{"  9b ", 9},
		{"E1", 11},
		{"EF", 11},
		{"Q1", 12},
		{"Q2", 12},
		{"Q3", 13},
		{"Q4", 13},
		{"q1", 12}, // case-insensitive
		{"", 0},
		{"ohne-zahl", 0},
		{"5f1", 5}, // Klassensatz-Notation: erste Zahl zählt
	}
	for _, c := range cases {
		if got := parseGrade(c.klasse); got != c.want {
			t.Errorf("parseGrade(%q) = %d, want %d", c.klasse, got, c.want)
		}
	}
}

// --- calculateDueDate: Fristberechnung. Ein stiller Fehler hier erzeugt falsche
// Rückgabefristen und damit falsches Mahnwesen. ---

func TestCalculateDueDate_RegularBook(t *testing.T) {
	got := calculateDueDate("Der Hobbit", "Buch", "07-31", 21, 7, 0)
	want := time.Now().AddDate(0, 0, 21)
	if !sameDay(got, want) {
		t.Errorf("reguläres Buch: got %v, want Tag %v", got, want)
	}
}

func TestCalculateDueDate_Media(t *testing.T) {
	for _, mt := range []string{"CD", "DVD", "Audio-CD", "dvd"} {
		got := calculateDueDate("Irgendwas", mt, "07-31", 21, 7, 0)
		want := time.Now().AddDate(0, 0, 7)
		if !sameDay(got, want) {
			t.Errorf("Medium %q: got %v, want Tag %v", mt, got, want)
		}
	}
}

func TestCalculateDueDate_LMF_DefaultStichtag(t *testing.T) {
	for _, titel := range []string{"LMF-Mathe 9", "lmf-deutsch 5"} {
		got := calculateDueDate(titel, "Buch", "07-31", 21, 7, 0)

		if got.Month() != time.July || got.Day() != 31 {
			t.Errorf("LMF %q: Stichtag soll 31.07 sein, got %02d-%02d", titel, got.Month(), got.Day())
		}
		if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
			t.Errorf("LMF %q: Frist soll auf 23:59:59 enden, got %v", titel, got)
		}

		// Jahr: laufendes Schuljahr, ab August rollt es ins nächste Kalenderjahr.
		// In Schul-Zeitzone rechnen, konsistent mit calculateDueDate (sonst flaky
		// am Monatswechsel nahe Mitternacht).
		now := time.Now().In(schoolLocation())
		wantYear := now.Year()
		if now.Month() >= time.August {
			wantYear++
		}
		if got.Year() != wantYear {
			t.Errorf("LMF %q: Stichtagsjahr got %d, want %d", titel, got.Year(), wantYear)
		}

		// Der Stichtag muss in der Schul-Zeitzone liegen, nicht in der Server-Zeitzone.
		if got.Location().String() != schoolLocation().String() {
			t.Errorf("LMF %q: Stichtag soll in %s liegen, war in %s",
				titel, schoolLocation(), got.Location())
		}
	}
}

func TestCalculateDueDate_LMF_CustomStichtag(t *testing.T) {
	got := calculateDueDate("LMF-Bio 7", "Buch", "06-15", 21, 7, 0)
	if got.Month() != time.June || got.Day() != 15 {
		t.Errorf("benutzerdefinierter Stichtag 06-15: got %02d-%02d", got.Month(), got.Day())
	}
}

func TestCalculateDueDate_LMF_InvalidStichtagFallsBackToJuly31(t *testing.T) {
	for _, bad := range []string{"99-99", "kaputt", "13-40", ""} {
		got := calculateDueDate("LMF-Chemie 8", "Buch", bad, 21, 7, 0)
		if got.Month() != time.July || got.Day() != 31 {
			t.Errorf("ungültiger Stichtag %q soll auf 31.07 zurückfallen, got %02d-%02d", bad, got.Month(), got.Day())
		}
	}
}

func TestCalculateDueDate_LMF_AdditionalYears(t *testing.T) {
	base := calculateDueDate("LMF-Mathe 9", "Buch", "07-31", 21, 7, 0)
	plus2 := calculateDueDate("LMF-Mathe 9", "Buch", "07-31", 21, 7, 2)

	if plus2.Year() != base.Year()+2 {
		t.Errorf("additionalYears=2 soll Stichtagsjahr um 2 erhöhen: base %d, got %d", base.Year(), plus2.Year())
	}
	// Monat/Tag bleiben unverändert.
	if plus2.Month() != base.Month() || plus2.Day() != base.Day() {
		t.Errorf("additionalYears darf Monat/Tag nicht verändern: base %02d-%02d, got %02d-%02d",
			base.Month(), base.Day(), plus2.Month(), plus2.Day())
	}
}

// --- querySettings: robustes Einlesen der Konfiguration inkl. Default-Fallbacks ---

func newServiceWithMock(t *testing.T) (*defaultLoanService, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock konnte nicht initialisiert werden: %v", err)
	}
	return &defaultLoanService{pool: mock}, mock
}

func TestQuerySettings_ParsesValuesAndIgnoresInvalid(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("frist_buch_tage", "30").
		AddRow("frist_medien_tage", "kaputt"). // ungültig → Default 7 bleibt
		AddRow("max_ausleihen_schueler", "8").
		AddRow("lmf_stichtag", "06-30").
		AddRow("ferien_leseclub_aktiv", "true").
		AddRow("max_overdue_items", "3")

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(rows)

	got, err := svc.querySettings(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if got.FristBuchTage != 30 {
		t.Errorf("FristBuchTage = %d, want 30", got.FristBuchTage)
	}
	if got.FristMedienTage != 7 {
		t.Errorf("ungültiger Wert soll Default behalten: FristMedienTage = %d, want 7", got.FristMedienTage)
	}
	if got.MaxAusleihenSchueler != 8 {
		t.Errorf("MaxAusleihenSchueler = %d, want 8", got.MaxAusleihenSchueler)
	}
	if got.LmfStichtag != "06-30" {
		t.Errorf("LmfStichtag = %q, want 06-30", got.LmfStichtag)
	}
	if !got.FerienLeseclubAktiv {
		t.Error("FerienLeseclubAktiv soll true sein")
	}
	if got.MaxOverdueItems != 3 {
		t.Errorf("MaxOverdueItems = %d, want 3", got.MaxOverdueItems)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestQuerySettings_EmptyTableKeepsDefaults(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))

	got, err := svc.querySettings(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.FristBuchTage != 21 || got.MaxAusleihenSchueler != 5 || got.LmfStichtag != "07-31" {
		t.Errorf("leere Tabelle soll sichere Defaults liefern, got %+v", got)
	}
}

func TestQuerySettings_DBErrorPropagates(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnError(errors.New("connection refused"))

	if _, err := svc.querySettings(context.Background()); err == nil {
		t.Error("DB-Fehler soll propagiert werden, bekam nil")
	}
}

// --- resolveCheckoutDueDate: kombiniert Einstellungen + Sonderregeln ---

func TestResolveCheckoutDueDate_LeseclubOverride(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	ziel := "2030-09-15"
	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", "true").
		AddRow("ferien_leseclub_zieldatum", ziel)

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(rows)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.resolveCheckoutDueDate(context.Background(), copy, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() != 2030 || got.Month() != time.September || got.Day() != 15 {
		t.Errorf("Leseclub-Zieldatum soll greifen: got %v, want 2030-09-15", got)
	}
}

func TestResolveCheckoutDueDate_LMFIgnoresLeseclub(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", "true").
		AddRow("ferien_leseclub_zieldatum", "2030-09-15")

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(rows)

	// LMF-Schulbücher folgen dem Stichtag, nicht dem Leseclub-Zieldatum.
	copy := &repository.BookCopy{Titel: "LMF-Mathe 9", Medientyp: "Buch"}
	got, err := svc.resolveCheckoutDueDate(context.Background(), copy, "9a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Month() != time.July || got.Day() != 31 {
		t.Errorf("LMF soll Stichtag (31.07) folgen, nicht Leseclub: got %v", got)
	}
}

func TestResolveCheckoutDueDate_DBErrorUsesEmergencyDefaults(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnError(errors.New("db down"))

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.resolveCheckoutDueDate(context.Background(), copy, "5a")
	// Bewusst kein Fehler: bei DB-Ausfall greifen Notfall-Defaults (21 Tage),
	// damit die Ausleihe nicht komplett blockiert.
	if err != nil {
		t.Fatalf("DB-Fehler soll mit Notfall-Default abgefangen werden, bekam: %v", err)
	}
	want := time.Now().AddDate(0, 0, 21)
	if !sameDay(got, want) {
		t.Errorf("Notfall-Default soll 21 Tage sein: got %v, want Tag %v", got, want)
	}
}

// Regressionstest: Eine NULL-wert-Zeile (z. B. nie gesetztes
// ferien_leseclub_zieldatum) machte vor dem coalesce-Fix JEDEN Checkout zum 500 —
// der Scan in string brach die pgx-Iteration ab und rows.Err() schlug durch.
// Mit coalesce kommt sie als leerer String an und fällt auf Defaults zurück.
func TestQuerySettings_LeererWertFaelltAufDefaultsZurueck(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_zieldatum", ""). // war in der DB: NULL
		AddRow("frist_buch_tage", "")

	mock.ExpectQuery(`SELECT schluessel, coalesce\(wert, ''\) FROM system_einstellungen`).
		WillReturnRows(rows)

	got, err := svc.querySettings(context.Background())
	if err != nil {
		t.Fatalf("leere Werte dürfen keinen Fehler auslösen: %v", err)
	}
	if got.FerienLeseclubZieldatum != nil {
		t.Errorf("leeres Zieldatum muss nil bleiben, bekam %q", *got.FerienLeseclubZieldatum)
	}
	if got.FristBuchTage != 21 {
		t.Errorf("leerer Zahlwert muss Default behalten: FristBuchTage = %d, want 21", got.FristBuchTage)
	}
}
