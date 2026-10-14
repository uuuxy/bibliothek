package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// istGleicherTag vergleicht zwei Zeitpunkte auf Kalendertag-Ebene (Jahr/Monat/Tag) — beide
// zuvor in die Schul-Zeitzone umgerechnet.
//
// Die Umrechnung ist der Kern: Date() liefert den Kalendertag in der Location des
// jeweiligen Zeitpunkts. berechneFrist rechnet bewusst in schulzeit.Zone(), die
// Erwartungswerte der Tests entstehen aber aus time.Now() in der Zeitzone des
// Rechners. Auf einem UTC-Server (CI) sind das ab 22:00 UTC zwei verschiedene
// Kalendertage — derselbe Zeitpunkt, aber 22:09 UTC = 00:09 des Folgetags in Berlin.
// Ohne Normalisierung waren diese Tests jede Nacht zwei Stunden lang rot.
func istGleicherTag(a, b time.Time) bool {
	loc := schulzeit.Zone()
	ay, am, ad := a.In(loc).Date()
	by, bm, bd := b.In(loc).Date()
	return ay == by && am == bm && ad == bd
}

// --- leseJahrgang: reine Jahrgangs-Extraktion (kritisch für mehrjährige LMF-Ausleihen) ---

func TestLeseJahrgang(t *testing.T) {
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
		if got := leseJahrgang(c.klasse); got != c.want {
			t.Errorf("leseJahrgang(%q) = %d, want %d", c.klasse, got, c.want)
		}
	}
}

// --- berechneFrist: Fristberechnung. Ein stiller Fehler hier erzeugt falsche
// Rückgabefristen und damit falsches Mahnwesen. ---

// Feste Uhr an einem Tag, dessen Fristen auf Schultage fallen: Dienstag 01.09.2026 plus 21
// Tage ist Dienstag 22.09., plus 7 Tage Dienstag 08.09. Mit time.Now() hinge das Ergebnis
// vom Kalender ab — Fristen, die auf Wochenende, Feiertag oder Ferien fallen, rücken auf den
// nächsten Schultag (tagesfrist_test.go).
func TestBerechneFrist_GewoehnlichesBuch(t *testing.T) {
	jetzt := time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone())
	got := berechneFrist(jetzt, fristAngaben{IstLernmittel: false, Medientyp: "Buch", LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
	want := time.Date(2026, time.September, 22, 23, 59, 59, 0, schulzeit.Zone())
	if !got.Equal(want) {
		t.Errorf("reguläres Buch: got %v, want %v", got, want)
	}
}

func TestBerechneFrist_Medien(t *testing.T) {
	jetzt := time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone())
	for _, mt := range []string{"CD", "DVD", "Audio-CD", "dvd"} {
		got := berechneFrist(jetzt, fristAngaben{IstLernmittel: false, Medientyp: mt, LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
		want := time.Date(2026, time.September, 8, 23, 59, 59, 0, schulzeit.Zone())
		if !got.Equal(want) {
			t.Errorf("Medium %q: got %v, want %v", mt, got, want)
		}
	}
}

func TestBerechneFrist_LMF_StichtagVorgabe(t *testing.T) {
	for _, titel := range []string{"Mathe 9", "Deutsch 5"} {
		got := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})

		if got.Month() != time.July || got.Day() != 31 {
			t.Errorf("LMF %q: Stichtag soll 31.07 sein, got %02d-%02d", titel, got.Month(), got.Day())
		}
		if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
			t.Errorf("LMF %q: Frist soll auf 23:59:59 enden, got %v", titel, got)
		}

		// Jahr: laufendes Schuljahr, ab August rollt es ins nächste Kalenderjahr.
		// In Schul-Zeitzone rechnen, konsistent mit berechneFrist (sonst flaky
		// am Monatswechsel nahe Mitternacht).
		now := time.Now().In(schulzeit.Zone())
		wantYear := now.Year()
		if now.Month() >= time.August {
			wantYear++
		}
		if got.Year() != wantYear {
			t.Errorf("LMF %q: Stichtagsjahr got %d, want %d", titel, got.Year(), wantYear)
		}

		// Der Stichtag muss in der Schul-Zeitzone liegen, nicht in der Server-Zeitzone.
		if got.Location().String() != schulzeit.Zone().String() {
			t.Errorf("LMF %q: Stichtag soll in %s liegen, war in %s",
				titel, schulzeit.Zone(), got.Location())
		}
	}
}

// Der Regelfall bei manueller Neuanlage über die Admin-Oberfläche: Ein Titel in Klartext trägt
// „LMF" nur in der Signatur. Es entscheidet allein das Feld, Titel und Signatur dürfen beliebig
// heißen; das Schulbuch bekommt den Stichtag, nicht die 21-Tage-Frist.
func TestBerechneFrist_LMF_NurInSignatur(t *testing.T) {
	got := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
	if got.Month() != time.July || got.Day() != 31 {
		t.Errorf("LMF nur in Signatur: Stichtag soll 31.07 sein, got %02d-%02d", got.Month(), got.Day())
	}
}

func TestBerechneFrist_LMF_EigenerStichtag(t *testing.T) {
	got := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: "06-15", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
	if got.Month() != time.June || got.Day() != 15 {
		t.Errorf("benutzerdefinierter Stichtag 06-15: got %02d-%02d", got.Month(), got.Day())
	}
}

func TestBerechneFrist_LMF_UngueltigerStichtagGiltAls31Juli(t *testing.T) {
	for _, bad := range []string{"99-99", "kaputt", "13-40", ""} {
		got := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: bad, FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
		if got.Month() != time.July || got.Day() != 31 {
			t.Errorf("ungültiger Stichtag %q soll auf 31.07 zurückfallen, got %02d-%02d", bad, got.Month(), got.Day())
		}
	}
}

func TestBerechneFrist_LMF_ZusatzJahre(t *testing.T) {
	base := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
	plus2 := berechneFrist(time.Now(), fristAngaben{IstLernmittel: true, Medientyp: "Buch", LMFStichtag: "07-31", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 2, Sommerferien: ""})

	if plus2.Year() != base.Year()+2 {
		t.Errorf("additionalYears=2 soll Stichtagsjahr um 2 erhöhen: base %d, got %d", base.Year(), plus2.Year())
	}
	// Monat/Tag bleiben unverändert.
	if plus2.Month() != base.Month() || plus2.Day() != base.Day() {
		t.Errorf("additionalYears darf Monat/Tag nicht verändern: base %02d-%02d, got %02d-%02d",
			base.Month(), base.Day(), plus2.Month(), plus2.Day())
	}
}

// --- ladeEinstellungen: robustes Einlesen der Konfiguration inkl. Default-Fallbacks ---

func newDienstMitAttrappe(t *testing.T) (*ausleihDienst, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock konnte nicht initialisiert werden: %v", err)
	}
	return &ausleihDienst{pool: mock}, mock
}

func TestLadeEinstellungen_LiestWerteUndUebergehtUngueltige(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("frist_buch_tage", strPtr("30")).
		AddRow("frist_medien_tage", strPtr("kaputt")). // ungültig → Default 7 bleibt
		AddRow("max_ausleihen_schueler", strPtr("8")).
		AddRow("lmf_stichtag", strPtr("06-30")).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("max_overdue_items", strPtr("3"))

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	got, err := svc.ladeEinstellungen(context.Background())
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
	if got.LMFStichtag != "06-30" {
		t.Errorf("LMFStichtag = %q, want 06-30", got.LMFStichtag)
	}
	if !got.FerienLeseclubAktiv {
		t.Error("FerienLeseclubAktiv soll true sein")
	}
	if got.UeberfaelligSperreAbMedien != 3 {
		t.Errorf("UeberfaelligSperreAbMedien = %d, want 3", got.UeberfaelligSperreAbMedien)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestLadeEinstellungen_LeereTabelleBehaeltVorgaben(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))

	got, err := svc.ladeEinstellungen(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.FristBuchTage != 21 || got.MaxAusleihenSchueler != 5 || got.LMFStichtag != "07-31" {
		t.Errorf("leere Tabelle soll sichere Defaults liefern, got %+v", got)
	}
}

func TestLadeEinstellungen_DatenbankfehlerGehtWeiter(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnError(errors.New("connection refused"))

	if _, err := svc.ladeEinstellungen(context.Background()); err == nil {
		t.Error("DB-Fehler soll propagiert werden, bekam nil")
	}
}

// --- bestimmeFrist: kombiniert Einstellungen + Sonderregeln ---

func TestBestimmeFrist_LeseclubGehtVor(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	ziel := "2030-09-15"
	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("ferien_leseclub_zieldatum", strPtr(ziel))

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() != 2030 || got.Month() != time.September || got.Day() != 15 {
		t.Errorf("Leseclub-Zieldatum soll greifen: got %v, want 2030-09-15", got)
	}
}

// Bleibt der Leseclub nach den Ferien eingeschaltet, bekäme jede neue Ausleihe das vergangene
// Ferienende als Frist: sofort überfällig, nach 14 Tagen gesperrt, am selben Tag im Mahnwesen.
// Die Theke zeigt die Frist beim Ausleihen nicht an, niemand bemerkte es. Ein Zieldatum vor
// heute gilt deshalb nicht.
func TestBestimmeFrist_LeseclubVergangenGiltNicht(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("ferien_leseclub_zieldatum", strPtr("2020-09-15"))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !got.After(time.Now()) {
		t.Errorf("vergangenes Leseclub-Zieldatum darf keine Frist sein: got %v", got)
	}
}

// Ein eingestelltes Zieldatum gilt nur bei eingeschaltetem Leseclub: Nach der Aktion bleibt das
// Datum in den Einstellungen stehen, der Schalter ist aus.
func TestBestimmeFrist_LeseclubAusgeschaltetGiltNicht(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", strPtr("false")).
		AddRow("ferien_leseclub_zieldatum", strPtr("2030-09-15"))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() == 2030 && got.Month() == time.September && got.Day() == 15 {
		t.Errorf("Leseclub ist ausgeschaltet, die Frist ist trotzdem sein Zieldatum: %v", got)
	}
	if !got.After(time.Now()) || got.After(time.Now().AddDate(0, 6, 0)) {
		t.Errorf("erwartet die Leihfrist für Bücher ab heute: got %v", got)
	}
}

func TestBestimmeFrist_LMFOhneLeseclub(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("ferien_leseclub_zieldatum", strPtr("2030-09-15"))

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)
	// Kein Klassen-Termin im LMF-Plan → der Stichtag bleibt.
	mock.ExpectQuery("SELECT min\\(t.datum\\)").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"min", "vergangen"}).AddRow((*time.Time)(nil), false))

	// LMF-Schulbücher folgen dem Stichtag, nicht dem Leseclub-Zieldatum.
	copy := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "9a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Month() != time.July || got.Day() != 31 {
		t.Errorf("LMF soll Stichtag (31.07) folgen, nicht Leseclub: got %v", got)
	}
}

// LMF-Plan (Register, Entscheidung 3a): Nennt der Plan für die Klasse einen Rückgabe-
// Termin, ist der die Frist — vor dem Stichtag. Eine mehrjährige Ausleihe folgt ihm nicht.
func TestBestimmeFrist_LMFFolgtDemKlassenTermin(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	termin := time.Date(2027, time.June, 28, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
	mock.ExpectQuery("SELECT min\\(t.datum\\)").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"min", "vergangen"}).AddRow(&termin, false))

	copy := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "9H1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() != 2027 || got.Month() != time.June || got.Day() != 28 || got.Hour() != 23 {
		t.Errorf("Frist soll das Tagesende des Klassen-Termins sein: got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}

	// Mehrjahresband (Spanne bis über den aktuellen Jahrgang): kein Lookup, Stichtag im Zieljahr.
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
	mehrjahr := &repository.Exemplar{Titel: "Atlas", IstLernmittel: true, Medientyp: "Buch", Mehrjahresband: true, JahrgangBis: 10}
	got, err = svc.bestimmeFrist(context.Background(), mehrjahr, "9H1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Month() != time.July || got.Day() != 31 {
		t.Errorf("mehrjährig soll beim Stichtag bleiben: got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mehrjährig: unerwarteter Lookup? %v", err)
	}
}

func TestBestimmeFrist_DatenbankfehlerNimmtFesteWerte(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnError(errors.New("db down"))

	// Feste Uhr: 01.09.2026 plus 21 Tage ist ein Schultag (TestBerechneFrist_GewoehnlichesBuch).
	svc.jetzt = func() time.Time { return time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone()) }
	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), copy, "5a")
	// Bewusst kein Fehler: bei DB-Ausfall greifen Notfall-Defaults (21 Tage),
	// damit die Ausleihe nicht komplett blockiert.
	if err != nil {
		t.Fatalf("DB-Fehler soll mit Notfall-Default abgefangen werden, bekam: %v", err)
	}
	want := time.Date(2026, time.September, 22, 23, 59, 59, 0, schulzeit.Zone())
	if !got.Equal(want) {
		t.Errorf("Notfall-Default soll 21 Tage sein: got %v, want %v", got, want)
	}
}

// Eine Zeile ohne Wert (etwa ein nie gesetztes ferien_leseclub_zieldatum) und eine mit leerem
// Wert lassen die Vorgabe stehen und sind kein Fehler: Sonst scheiterte jede Ausleihe am Lesen
// der Einstellungen. An der Datenbank hält das TestLadeAusleihEinstellungen_AnDerDatenbank.
func TestLadeEinstellungen_LeererWertFaelltAufVorgabeZurueck(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_zieldatum", (*string)(nil)).
		AddRow("frist_buch_tage", strPtr(""))

	mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
		WillReturnRows(rows)

	got, err := svc.ladeEinstellungen(context.Background())
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

// Am oder nach dem Rückgabetermin der Klasse ist die Frist der Stichtag des folgenden
// Schuljahres. Wäre es der Termin selbst (heute 23:59) oder der Stichtag des laufenden
// Schuljahres, ein Tag in den Ferien, stünde nach den Ferien die ganze Klasse überfällig da und
// wäre nach 14 Tagen gesperrt. Die Uhr ist fest, damit der Tag vor, am und nach dem Termin je
// einen Fall hat (Bugklasse „Frist am Tag des Ereignisses"); die Lage kommt hier aus pgxmock,
// am echten Postgres prüft sie lmf_frist_termintag_pg_test.go.
func TestBestimmeFrist_AmOderNachDemTerminGiltDerStichtagDesFolgendenSchuljahres(t *testing.T) {
	termin := time.Date(2027, time.June, 29, 0, 0, 0, 0, schulzeit.Zone())
	faelle := []struct {
		name         string
		heute        time.Time
		bevorstehend *time.Time
		vergangen    bool
		stichtag     string
		want         time.Time
	}{
		{"Tag davor: der Termin ist die Frist", termin.AddDate(0, 0, -1), &termin, false, "07-31",
			time.Date(2027, time.June, 29, 23, 59, 59, 0, schulzeit.Zone())},
		{"am Termintag: Stichtag des folgenden Schuljahres", termin, nil, true, "07-31",
			time.Date(2028, time.July, 31, 23, 59, 59, 0, schulzeit.Zone())},
		{"Tag danach: Stichtag des folgenden Schuljahres", termin.AddDate(0, 0, 1), nil, true, "07-31",
			time.Date(2028, time.July, 31, 23, 59, 59, 0, schulzeit.Zone())},
		{"Tag danach, Stichtag im Herbst: der des folgenden Schuljahres", termin.AddDate(0, 0, 1), nil, true, "09-30",
			time.Date(2027, time.September, 30, 23, 59, 59, 0, schulzeit.Zone())},
		{"Nachzügler-Termin steht noch an, der erste ist vorbei: trotzdem das folgende Schuljahr",
			termin, func() *time.Time { z := termin.AddDate(0, 0, 6); return &z }(), true, "07-31",
			time.Date(2028, time.July, 31, 23, 59, 59, 0, schulzeit.Zone())},
		{"neues Schuljahr, kein Termin mehr: Stichtag des laufenden Schuljahres",
			time.Date(2027, time.September, 6, 10, 0, 0, 0, schulzeit.Zone()), nil, false, "07-31",
			time.Date(2028, time.July, 31, 23, 59, 59, 0, schulzeit.Zone())},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc, mock := newDienstMitAttrappe(t)
			defer mock.Close()
			svc.jetzt = func() time.Time { return fall.heute }
			mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("lmf_stichtag", strPtr(fall.stichtag)))
			mock.ExpectQuery("SELECT min\\(t.datum\\)").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"min", "vergangen"}).AddRow(fall.bevorstehend, fall.vergangen))

			copy := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
			got, err := svc.bestimmeFrist(context.Background(), copy, "9H2")
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !got.Equal(fall.want) {
				t.Errorf("Frist %v, erwartet %v", got.In(schulzeit.Zone()), fall.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unerfüllte Erwartungen: %v", err)
			}
		})
	}
}
