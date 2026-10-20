package leser

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

// Die Ausweisnummer eines Importzugangs hat dieselbe Form wie die der Handanlage.
//
// Die Nummern kommen aus derselben Sequenz wie die der Handanlage
// (AusweisnummernAbfragen.ZieheNaechste, ausweis_nummer_start), einmal je Lauf gezogen und
// fortlaufend weitergezählt. Dieser Test prüft nur die Form: dass Import und Handanlage
// dieselbe erzeugen. Dass zwei Läufe sich nicht ins Gehege kommen, prüft
// api/leser/lusd_ausweisnummern_pg_test.go an der echten Datenbank; an einer reinen
// Formatfunktion wäre es nicht zu beweisen.
func TestGenerateImportBarcode_WieDieHandanlage(t *testing.T) {
	seen := make(map[string]bool, 5000)
	for i := 1; i <= 5000; i++ {
		b := repository.AusweisNummer(i)
		if seen[b] {
			t.Fatalf("Barcode-Kollision bei Nummer %d: %s", i, b)
		}
		if !strings.HasPrefix(b, repository.AusweisPraefix) {
			t.Fatalf("unerwartetes Format: %s — erwartet die Vorsilbe %q", b, repository.AusweisPraefix)
		}
		if b != repository.AusweisNummer(i) {
			t.Fatalf("Import erzeugt %q, die Handanlage %q — eine Form, eine Quelle", b, repository.AusweisNummer(i))
		}
		seen[b] = true
	}
}

func lusdSchuelerZeilen(n int) *pgxmock.Rows {
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

// erwarteEinstellungen: Der Lauf liest die Karenzzeit vor der Transaktion (leere
// Tabelle = Vorgabe); ohne diese Erwartung sähe die Attrappe eine fremde Abfrage.
func erwarteEinstellungen(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
}

func TestLUSDAenderungen_MassenabgangHaeltVorJedemSchreiben(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	// 10 aktive Schüler in der DB, CSV enthält nur 2 davon → 8 Abgänger (80%).
	erwarteEinstellungen(mock)
	mock.ExpectBegin()
	// uebernehmen=true: Zuerst kommt der Advisory-Lock des Imports; er reiht parallele Läufe.
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`).
		WillReturnRows(lusdSchuelerZeilen(10))
	// Keine Exec-Erwartungen: Die Bremse muss vor der ersten schreibenden Anweisung
	// greifen. Unerwartete Exec-Aufrufe ließen die Attrappe fehlschlagen.
	mock.ExpectRollback()

	records := []lusd.Zeile{
		{LUSDID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"},
		{LUSDID: "L-b", Vorname: "Vorb", Nachname: "Nachb", Klasse: "8A"},
	}

	_, err = s.berechneLUSDAenderungen(t.Context(), records, true, false)
	var massErr *lusd.MassenabgangFehler
	if !errors.As(err, &massErr) {
		t.Fatalf("s.berechneLUSDAenderungen(…) = %v, erwartet lusd.MassenabgangFehler", err)
	}
	if massErr.Abgaenger != 8 || massErr.Aktive != 10 {
		t.Errorf("Zahlen falsch: %+v", massErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestLUSDAenderungen_VorschauSchreibtNie(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	erwarteEinstellungen(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`).
		WillReturnRows(lusdSchuelerZeilen(10))
	mock.ExpectRollback()

	records := []lusd.Zeile{
		{LUSDID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"}, // Klassenwechsel
		{LUSDID: "L-neu", Vorname: "Neu", Nachname: "Kind", Klasse: "5A"}, // Neuzugang
		{Vorname: "Ohne", Nachname: "ID", Klasse: "5A"},                   // ohne LUSD-ID → übersprungen
	}

	res, err := s.berechneLUSDAenderungen(t.Context(), records, false, false)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(res.Neuzugaenge) != 1 || len(res.Klassenwechsel) != 1 || len(res.Abgaenger) != 9 {
		t.Errorf("Diff falsch: neu=%d wechsel=%d abgaenger=%d", len(res.Neuzugaenge), len(res.Klassenwechsel), len(res.Abgaenger))
	}
	if res.AktiveImBestand != 10 || res.OhneID != 1 {
		t.Errorf("Metadaten falsch: aktiv=%d skipped=%d", res.AktiveImBestand, res.OhneID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Preview darf nie schreiben: %v", err)
	}
}

// Die Vorschau gibt weiter, was das Lesen der Datei über ihre Zeilen weiß: wie viele es sind,
// wie viele Dubletten zusammengelegt wurden und welche davon verschiedene Klassen trugen. Nur so
// fällt vor dem Import auf, wenn zwei Schüler zu einem zusammenfielen.
func TestLUSDVorschau_NenntZusammengelegteDubletten(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	erwarteEinstellungen(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`).
		WillReturnRows(lusdSchuelerZeilen(2))
	mock.ExpectRollback()

	zusammengelegt := []lusd.VorschauSchueler{{ID: "L-a", Vorname: "Vora", Nachname: "Nacha", AlteKlasse: "7A", NeueKlasse: "8A"}}
	datei := lusd.Datei{
		Modus: lusd.ModusID,
		Zeilen: []lusd.Zeile{
			{LUSDID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"},
			{LUSDID: "L-b", Vorname: "Vorb", Nachname: "Nachb", Klasse: "8B"},
			{LUSDID: "L-c", Vorname: "Vorc", Nachname: "Nachc", Klasse: "5A"},
		},
		DublettenInDatei: 2,
		Zusammengelegt:   zusammengelegt,
	}
	res, err := s.berechneLUSD(t.Context(), datei, false, false)
	if err != nil {
		t.Fatalf("s.berechneLUSD(…) = %v", err)
	}
	if res.ZeilenInDatei != 3 || res.DublettenInDatei != 2 {
		t.Errorf("Vorschau: %d Zeilen, %d Dubletten — erwartet 3 und 2", res.ZeilenInDatei, res.DublettenInDatei)
	}
	if diff := cmp.Diff(zusammengelegt, res.DublettenAbweichend); diff != "" {
		t.Errorf("Hinweis auf abweichende Dubletten (-erwartet +ist):\n%s", diff)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

// Ein Lesefehler im Bestand ist ein Fehler und keine kürzere Liste. Liefe die Vorschau mit dem
// halben oder leeren Bestand weiter, stünden die Zeilen der Datei als Neuzugänge da und die
// fehlenden Schüler nicht als Abgänger.
func TestLUSDVorschau_LesefehlerDesBestandsIstFehler(t *testing.T) {
	lesefehler := errors.New("verbindung weg")
	for _, fall := range []struct {
		name    string
		antwort func(*pgxmock.ExpectedQuery)
	}{
		{"die Abfrage scheitert", func(q *pgxmock.ExpectedQuery) { q.WillReturnError(lesefehler) }},
		{"das Lesen bricht nach zwei Zeilen ab", func(q *pgxmock.ExpectedQuery) {
			q.WillReturnRows(lusdSchuelerZeilen(2).RowError(2, lesefehler))
		}},
	} {
		t.Run(fall.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()
			s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

			erwarteEinstellungen(mock)
			mock.ExpectBegin()
			fall.antwort(mock.ExpectQuery(`SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger`))
			mock.ExpectRollback()

			res, err := s.berechneLUSDAenderungen(t.Context(), []lusd.Zeile{
				{LUSDID: "L-a", Vorname: "Vora", Nachname: "Nacha", Klasse: "8A"},
			}, false, false)
			if !errors.Is(err, lesefehler) {
				t.Errorf("s.berechneLUSDAenderungen(…) = %+v, %v, erwartet den Lesefehler", res, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("offene Erwartungen: %v", err)
			}
		})
	}
}
