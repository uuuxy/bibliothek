package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pkg/lmfplan"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Sommerferien 2031 nach der Einstellung: Montag 07.07. bis 15.08.; letzter Schultag davor ist
// Freitag, der 04.07.2031.
const sommerferien2031 = `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`

func fristEnde2031(m time.Month, d int) time.Time {
	return schulzeit.TagesEnde(time.Date(2031, m, d, 12, 0, 0, 0, schulzeit.Zone()))
}

// Ein Abgänger behält nichts über den letzten Schultag vor den Sommerferien hinaus. Wer später
// ausleiht, bekommt das Ende des Tages; wer keiner Abschlussklasse angehört, behält seine Frist.
func TestAbgaengerFrist(t *testing.T) {
	kalender := lmfplan.FerientabelleAus(sommerferien2031)
	vorDenFerien := time.Date(2031, time.June, 30, 10, 0, 0, 0, schulzeit.Zone())
	inDenFerien := time.Date(2031, time.July, 20, 10, 0, 0, 0, schulzeit.Zone())
	faelle := []struct {
		name, klasse string
		heute, frist time.Time
		erwartet     time.Time
	}{
		{"Realschule 10 nach den Ferien fällig", "10R1", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.July, 4)},
		{"Hauptschule 9", "9H1", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.July, 4)},
		{"Oberstufe 13 der Schule", "13T1", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.July, 4)},
		{"Frist vor dem letzten Schultag bleibt", "10R1", vorDenFerien, fristEnde2031(time.July, 2), fristEnde2031(time.July, 2)},
		{"in den Ferien: Ende des Tages", "10R1", inDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.July, 20)},
		{"Jahrgang 12 geht nicht ab", "12T1", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.August, 18)},
		{"Einführungsphase", "ET1", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.August, 18)},
		{"Klasse 5", "5a", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.August, 18)},
		{"ohne Klasse", "", vorDenFerien, fristEnde2031(time.August, 18), fristEnde2031(time.August, 18)},
	}
	for _, f := range faelle {
		if got := AbgaengerFrist(f.frist, f.klasse, kalender, f.heute); !got.Equal(f.erwartet) {
			t.Errorf("%s: AbgaengerFrist(%v, %q) = %v, erwartet %v", f.name, f.frist, f.klasse, got, f.erwartet)
		}
	}
}

// Beim Buch gilt der Ferien-Leseclub für einen Abgänger nicht, und seine Frist endet am letzten
// Schultag vor den Ferien. Ein Kind der Klasse 5 bekommt das Datum des Leseclubs. Liegt der
// Leseclub vor den Sommerferien (Ostern), bekommt der Abgänger die Leihfrist der Bücher.
func TestBestimmeFristAm_AbgaengerOhneLeseclubUndVorDenFerien(t *testing.T) {
	svc, mock := newDienstMitAttrappe(t)
	defer mock.Close()
	sommer := time.Date(2031, time.June, 30, 10, 0, 0, 0, schulzeit.Zone())
	ostern := time.Date(2031, time.April, 1, 10, 0, 0, 0, schulzeit.Zone())
	exemplar := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}

	for _, f := range []struct {
		klasse, leseclub string
		heute, erwartet  time.Time
	}{
		{"10R1", "2031-08-20", sommer, fristEnde2031(time.July, 4)},
		{"5a", "2031-08-20", sommer, fristEnde2031(time.August, 20)},
		{"10R1", "2031-04-25", ostern, fristEnde2031(time.April, 22)},
		{"5a", "2031-04-25", ostern, fristEnde2031(time.April, 25)},
	} {
		mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
			WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
				AddRow("ferien_leseclub_aktiv", strPtr("true")).
				AddRow("ferien_leseclub_zieldatum", strPtr(f.leseclub)).
				AddRow("sommerferien", strPtr(sommerferien2031)))
		got, err := svc.bestimmeFristAm(context.Background(), exemplar, f.klasse, f.heute)
		if err != nil {
			t.Fatalf("%s: %v", f.klasse, err)
		}
		if !got.Equal(f.erwartet) {
			t.Errorf("Klasse %s am %s: Frist %v, erwartet %v", f.klasse, f.heute.Format(time.DateOnly), got, f.erwartet)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
