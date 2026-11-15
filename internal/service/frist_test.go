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
			t.Errorf("leseJahrgang(%q) = %d, erwartet %d", c.klasse, got, c.want)
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
	got := berechneFrist(jetzt, fristAngaben{IstLernmittel: false, Medientyp: "Buch", Klasse: "05G1", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
	want := time.Date(2026, time.September, 22, 23, 59, 59, 0, schulzeit.Zone())
	if !got.Equal(want) {
		t.Errorf("reguläres Buch = %v, erwartet %v", got, want)
	}
}

func TestBerechneFrist_Medien(t *testing.T) {
	jetzt := time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone())
	for _, mt := range []string{"CD", "DVD", "Audio-CD", "dvd"} {
		got := berechneFrist(jetzt, fristAngaben{IstLernmittel: false, Medientyp: mt, Klasse: "05G1", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0, Sommerferien: ""})
		want := time.Date(2026, time.September, 8, 23, 59, 59, 0, schulzeit.Zone())
		if !got.Equal(want) {
			t.Errorf("Medium %q = %v, erwartet %v", mt, got, want)
		}
	}
}

// lmfFrist rechnet die Frist eines Schulbuchs am Dienstag, 01.09.2026, im Schuljahr 2026/27. Die
// Sommerferien 2027 beginnen am Montag, 28.06.; der Donnerstag davor ist der 24.06.2027.
func lmfFrist(klasse string, zusatzJahre int, sommerferien string) time.Time {
	jetzt := time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone())
	return berechneFrist(jetzt, fristAngaben{IstLernmittel: true, Medientyp: "Buch", Klasse: klasse, FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: zusatzJahre, Sommerferien: sommerferien})
}

func TestBerechneFrist_LMF_DonnerstagVorDenSommerferien(t *testing.T) {
	for _, klasse := range []string{"05G1", "9H1", "10R2"} {
		got := lmfFrist(klasse, 0, "")
		if want := time.Date(2027, time.June, 24, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
			t.Errorf("Schulbuch der %s = %v, erwartet das Ende des 24.06.2027", klasse, got)
		}
		// Der Stichtag muss in der Schul-Zeitzone liegen, nicht in der Server-Zeitzone.
		if got.Location().String() != schulzeit.Zone().String() {
			t.Errorf("Schulbuch der %s: Frist soll in %s liegen, war in %s", klasse, schulzeit.Zone(), got.Location())
		}
	}
}

// Die Oberstufe organisiert Rückgabe und Ausgabe selbst und behält das Ende des Schuljahres;
// ebenso ein Leser ohne Klasse.
func TestBerechneFrist_LMF_OberstufeBehaeltDen31Juli(t *testing.T) {
	for _, klasse := range []string{"ET", "12T", "13T", ""} {
		got := lmfFrist(klasse, 0, "")
		if want := time.Date(2027, time.July, 31, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
			t.Errorf("Schulbuch der Klasse %q = %v, erwartet das Ende des 31.07.2027", klasse, got)
		}
	}
}

// Die Jahre nach der Tabelle des Programms trägt die Schule selbst ein; ohne Eintrag gilt der
// 31. Juli.
func TestBerechneFrist_LMF_FerienDerSchule(t *testing.T) {
	jetzt := time.Date(2031, time.September, 2, 10, 0, 0, 0, schulzeit.Zone())
	angaben := fristAngaben{IstLernmittel: true, Medientyp: "Buch", Klasse: "05G1", FristBuchTage: 21, FristMedienTage: 7, ZusatzJahre: 0,
		Sommerferien: `[{"jahr":2032,"von":"2032-06-28","bis":"2032-08-06"}]`}
	if got, want := berechneFrist(jetzt, angaben), time.Date(2032, time.June, 24, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("mit eingetragenen Ferien = %v, erwartet %v", got, want)
	}
	angaben.Sommerferien = ""
	if got, want := berechneFrist(jetzt, angaben), time.Date(2032, time.July, 31, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("ohne bekannte Ferien = %v, erwartet %v", got, want)
	}
}

// Der Regelfall bei manueller Neuanlage über die Admin-Oberfläche: Ein Titel in Klartext trägt
// „LMF" nur in der Signatur. Es entscheidet allein das Feld, Titel und Signatur dürfen beliebig
// heißen; das Schulbuch bekommt den Stichtag, nicht die 21-Tage-Frist.
func TestBerechneFrist_LMF_NurInSignatur(t *testing.T) {
	if got := lmfFrist("7H2", 0, ""); got.Month() != time.June || got.Day() != 24 {
		t.Errorf("berechneFrist(LMF nur in Signatur) = %02d-%02d, erwartet den Stichtag 06-24", got.Month(), got.Day())
	}
}

// Ein Band bis Jahrgang 9 bleibt bei einem Kind der 7 zwei Schuljahre länger: Stichtag ist der
// Donnerstag vor den Sommerferien 2029 (ab 16.07.), nicht derselbe Kalendertag zwei Jahre später.
func TestBerechneFrist_LMF_ZusatzJahre(t *testing.T) {
	if got, want := lmfFrist("07G1", 2, ""), time.Date(2029, time.July, 12, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("berechneFrist(ZusatzJahre 2) = %v, erwartet %v", got, want)
	}
}

// --- ladeEinstellungen: robustes Einlesen der Konfiguration inkl. Default-Fallbacks ---

func newDienstMitAttrappe(t *testing.T) (*AusleihDienst, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock konnte nicht initialisiert werden: %v", err)
	}
	return &AusleihDienst{pool: mock}, mock
}

func TestLadeEinstellungen_LiestWerteUndUebergehtUngueltige(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("frist_buch_tage", strPtr("30")).
		AddRow("frist_medien_tage", strPtr("kaputt")). // ungültig → Default 7 bleibt
		AddRow("max_ausleihen_schueler", strPtr("8")).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("max_overdue_items", strPtr("3"))

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	got, err := svc.ladeEinstellungen(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if got.FristBuchTage != 30 {
		t.Errorf("FristBuchTage = %d, erwartet 30", got.FristBuchTage)
	}
	if got.FristMedienTage != 7 {
		t.Errorf("ungültiger Wert soll Default behalten: FristMedienTage = %d, erwartet 7", got.FristMedienTage)
	}
	if got.MaxAusleihenSchueler != 8 {
		t.Errorf("MaxAusleihenSchueler = %d, erwartet 8", got.MaxAusleihenSchueler)
	}
	if !got.FerienLeseclubAktiv {
		t.Error("FerienLeseclubAktiv soll true sein")
	}
	if got.UeberfaelligSperreAbMedien != 3 {
		t.Errorf("UeberfaelligSperreAbMedien = %d, erwartet 3", got.UeberfaelligSperreAbMedien)
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
	if got.FristBuchTage != 21 || got.MaxAusleihenSchueler != 5 {
		t.Errorf("ladeEinstellungen(…) bei leerer Tabelle = %+v, erwartet die Vorgaben", got)
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

	// Das Zieldatum liegt immer in der Zukunft: Ein vergangenes gilt nicht (der Test darunter).
	ziel := time.Now().AddDate(1, 0, 0)
	rows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("ferien_leseclub_aktiv", strPtr("true")).
		AddRow("ferien_leseclub_zieldatum", strPtr(ziel.Format(time.DateOnly)))

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(rows)

	exemplar := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Format(time.DateOnly) != ziel.Format(time.DateOnly) {
		t.Errorf("Leseclub-Zieldatum soll greifen = %v, erwartet %s", got, ziel.Format(time.DateOnly))
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

	exemplar := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !got.After(time.Now()) {
		t.Errorf("bestimmeFrist(…) = %v, erwartet eine Frist in der Zukunft statt des vergangenen Leseclub-Zieldatums", got)
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

	exemplar := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() == 2030 && got.Month() == time.September && got.Day() == 15 {
		t.Errorf("Leseclub ist ausgeschaltet, die Frist ist trotzdem sein Zieldatum: %v", got)
	}
	if !got.After(time.Now()) || got.After(time.Now().AddDate(0, 6, 0)) {
		t.Errorf("bestimmeFrist(…) = %v, erwartet die Leihfrist für Bücher ab heute", got)
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
		WillReturnRows(pgxmock.NewRows([]string{"min", "tausch_begonnen"}).AddRow((*time.Time)(nil), false))

	// LMF-Schulbücher folgen dem Stichtag, nicht dem Leseclub-Zieldatum: im Schuljahr 2026/27
	// dem Donnerstag vor den Sommerferien 2027.
	svc.jetzt = func() time.Time { return time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone()) }
	exemplar := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "9a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if want := time.Date(2027, time.June, 24, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("bestimmeFrist(LMF) = %v, erwartet den Stichtag %v statt des Leseclubs", got, want)
	}
}

// LMF-Plan (Register, Entscheidung 3a): Nennt der Plan für die Klasse einen Rückgabe-
// Termin, ist der die Frist — vor dem Stichtag. Eine mehrjährige Ausleihe folgt ihm nicht.
func TestBestimmeFrist_LMFFolgtDemKlassenTermin(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()

	svc.jetzt = func() time.Time { return time.Date(2026, time.September, 1, 10, 0, 0, 0, schulzeit.Zone()) }
	termin := time.Date(2027, time.June, 28, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
	mock.ExpectQuery("SELECT min\\(t.datum\\)").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"min", "tausch_begonnen"}).AddRow(&termin, false))

	exemplar := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "9H1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.Year() != 2027 || got.Month() != time.June || got.Day() != 28 || got.Hour() != 23 {
		t.Errorf("bestimmeFrist(…, \"9H1\") = %v, erwartet das Tagesende des Klassen-Termins", got)
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
	// Bis Jahrgang 10: der Donnerstag vor den Sommerferien 2028 (ab 03.07.).
	if want := time.Date(2028, time.June, 29, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("bestimmeFrist(Mehrjahresband) = %v, erwartet den Stichtag %v", got, want)
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
	exemplar := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err := svc.bestimmeFrist(context.Background(), exemplar, "5a")
	// Bewusst kein Fehler: bei DB-Ausfall greifen Notfall-Defaults (21 Tage),
	// damit die Ausleihe nicht komplett blockiert.
	if err != nil {
		t.Fatalf("DB-Fehler soll mit Notfall-Default abgefangen werden, bekam: %v", err)
	}
	want := time.Date(2026, time.September, 22, 23, 59, 59, 0, schulzeit.Zone())
	if !got.Equal(want) {
		t.Errorf("Notfall-Default soll 21 Tage sein = %v, erwartet %v", got, want)
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
		t.Errorf("leerer Zahlwert muss Default behalten: FristBuchTage = %d, erwartet 21", got.FristBuchTage)
	}
}

// Ab dem Beginn des Büchertauschs ist die Frist eines Schulbuchs der Stichtag des folgenden
// Schuljahres, für jede Klasse. Wäre es der Termin der Klasse oder der Stichtag des laufenden
// Schuljahres, stünde nach den Ferien das Kind überfällig da und wäre nach 14 Tagen gesperrt.
// Die Uhr ist fest, damit der Tag vor, am und nach dem Termin je einen Fall hat (Bugklasse
// „Frist am Tag des Ereignisses"); die Lage kommt hier aus pgxmock, am echten Postgres prüft sie
// lmf_frist_termintag_pg_test.go. Der Tausch liegt in der Woche vor den Sommerferien 2027 (ab
// Montag, 28.06.); Stichtage sind der 24.06.2027 und der 29.06.2028, in der Oberstufe der 31.07.
func TestBestimmeFrist_AbBeginnDesBuechertauschsGiltDerStichtagDesFolgendenSchuljahres(t *testing.T) {
	termin := time.Date(2027, time.June, 22, 0, 0, 0, 0, schulzeit.Zone())
	ende := func(jahr int, monat time.Month, tag int) time.Time {
		return time.Date(jahr, monat, tag, 23, 59, 59, 0, schulzeit.Zone())
	}
	faelle := []struct {
		name           string
		heute          time.Time
		klasse         string
		bevorstehend   *time.Time
		tauschBegonnen bool
		want           time.Time
	}{
		{"vor dem Büchertausch: der Termin ist die Frist", termin.AddDate(0, 0, -4), "9H2", &termin, false, ende(2027, time.June, 22)},
		{"Tausch läuft, Termin der Klasse steht noch an: Stichtag des folgenden Schuljahres", termin.AddDate(0, 0, -1), "9H2", &termin, true, ende(2028, time.June, 29)},
		{"Tausch läuft, Klasse steht nicht im Plan: Stichtag des folgenden Schuljahres", termin.AddDate(0, 0, -1), "9H2", nil, true, ende(2028, time.June, 29)},
		{"Tausch läuft, Oberstufe: ihr Stichtag des folgenden Schuljahres", termin.AddDate(0, 0, -1), "ET", nil, true, ende(2028, time.July, 31)},
		{"vor dem Büchertausch, Klasse steht nicht im Plan: Stichtag des laufenden Schuljahres", termin.AddDate(0, 0, -4), "9H2", nil, false, ende(2027, time.June, 24)},
		{"am Termintag: Stichtag des folgenden Schuljahres", termin, "9H2", nil, true, ende(2028, time.June, 29)},
		{"Tag danach: Stichtag des folgenden Schuljahres", termin.AddDate(0, 0, 1), "9H2", nil, true, ende(2028, time.June, 29)},
		{"Nachzügler-Termin steht noch an, der erste ist vorbei: trotzdem das folgende Schuljahr",
			termin, "9H2", func() *time.Time { z := termin.AddDate(0, 0, 2); return &z }(), true, ende(2028, time.June, 29)},
		{"ohne Plan nach dem Donnerstag: das folgende Schuljahr", time.Date(2027, time.June, 25, 10, 0, 0, 0, schulzeit.Zone()), "9H2", nil, false, ende(2028, time.June, 29)},
		{"neues Schuljahr, kein Termin mehr: Stichtag des laufenden Schuljahres",
			time.Date(2027, time.September, 6, 10, 0, 0, 0, schulzeit.Zone()), "9H2", nil, false, ende(2028, time.June, 29)},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc, mock := newDienstMitAttrappe(t)
			defer mock.Close()
			svc.jetzt = func() time.Time { return fall.heute }
			mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
			mock.ExpectQuery("SELECT min\\(t.datum\\)").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"min", "tausch_begonnen"}).AddRow(fall.bevorstehend, fall.tauschBegonnen))

			exemplar := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
			got, err := svc.bestimmeFrist(context.Background(), exemplar, fall.klasse)
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
