package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Entscheidung vom 24.09.2026: Eine Frist, die in Tagen zählt, rückt auf den nächsten
// Schultag, wenn sie auf ein Wochenende, einen Feiertag oder in die Ferien fällt. Der Anlass
// ist dieser Tag selbst: Donnerstag 24.09.2026 plus 21 Tage ist der 15.10., mitten in den
// Herbstferien (05.10.–17.10.2026).
func TestCalculateDueDate_FristRuecktAufDenNaechstenSchultag(t *testing.T) {
	berlin := schoolLocation()
	am := func(j int, m time.Month, d int) time.Time { return time.Date(j, m, d, 10, 0, 0, 0, berlin) }
	ende := func(j int, m time.Month, d int) time.Time { return time.Date(j, m, d, 23, 59, 59, 0, berlin) }
	eigene := `[{"jahr":2031,"von":"2031-07-14","bis":"2031-08-22"}]`
	for _, c := range []struct {
		name         string
		jetzt        time.Time
		medientyp    string
		buchTage     int
		sommerferien string
		soll         time.Time
	}{
		{"Buch, Frist in den Herbstferien", am(2026, time.September, 24), "Buch", 21, "", ende(2026, time.October, 19)},
		{"Buch, Frist am ersten Ferientag", am(2026, time.September, 14), "Buch", 21, "", ende(2026, time.October, 19)},
		{"Medium, Frist am ersten Ferientag", am(2026, time.September, 28), "DVD", 21, "", ende(2026, time.October, 19)},
		{"Buch, Frist am Samstag", am(2026, time.September, 16), "Buch", 10, "", ende(2026, time.September, 28)},
		{"Buch, Frist an einem Schultag", am(2026, time.September, 1), "Buch", 21, "", ende(2026, time.September, 22)},
		{"eigene Sommerferien aus der Einstellung", am(2031, time.June, 23), "Buch", 21, eigene, ende(2031, time.August, 25)},
		{"ohne Eintrag kennt 2031 keine Sommerferien", am(2031, time.June, 23), "Buch", 21, "", ende(2031, time.July, 14)},
	} {
		got := calculateDueDate(c.jetzt, DueDateOptions{IstLernmittel: false, Medientyp: c.medientyp, LmfStichtag: "07-31",
			FristBuchTage: c.buchTage, FristMedienTage: 7, AdditionalYears: 0, Sommerferien: c.sommerferien})
		if !got.Equal(c.soll) {
			t.Errorf("%s: %s, erwartet %s", c.name, got.In(berlin), c.soll)
		}
	}
}

// Lernmittel behalten ihren Stichtag, auch wenn er in den Sommerferien liegt — sonst stünde
// jedes Schulbuch am ersten Schultag danach, dem Tag der Bücherausgabe.
func TestCalculateDueDate_LernmittelBleibtAmStichtag(t *testing.T) {
	jetzt := time.Date(2026, time.September, 24, 10, 0, 0, 0, schoolLocation())
	got := calculateDueDate(jetzt, DueDateOptions{IstLernmittel: true, Medientyp: "Buch", LmfStichtag: "07-31",
		FristBuchTage: 21, FristMedienTage: 7, AdditionalYears: 0, Sommerferien: ""})
	// Samstag 31.07.2027, in den Sommerferien 28.06.–06.08.2027.
	if want := time.Date(2027, time.July, 31, 23, 59, 59, 0, schoolLocation()); !got.Equal(want) {
		t.Errorf("Lernmittel: %s, erwartet %s", got.In(schoolLocation()), want)
	}
}

// Die eigenen Sommerferien kommen über die Einstellung „sommerferien" in die Frist — dieselbe
// Liste, die der LMF-Planer liest, keine zweite.
func TestResolveCheckoutDueDate_SommerferienAusDerEinstellung(t *testing.T) {
	svc, mock := newServiceWithMock(t)
	defer mock.Close()
	svc.jetzt = func() time.Time { return time.Date(2031, time.June, 23, 10, 0, 0, 0, schoolLocation()) }
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow(lmfplan.SommerferienSchluessel, `[{"jahr":2031,"von":"2031-07-14","bis":"2031-08-22"}]`))

	got, err := svc.resolveCheckoutDueDate(context.Background(), &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch"}, "5a")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if want := time.Date(2031, time.August, 25, 23, 59, 59, 0, schoolLocation()); !got.Equal(want) {
		t.Errorf("Frist %s, erwartet %s (erster Schultag nach den eingetragenen Sommerferien)", got.In(schoolLocation()), want)
	}
}

// Gezählt wird ab dem Berliner Kalendertag: 23.09.2026 22:30 UTC ist in Berlin schon der
// 24.09. — plus 21 Tage der 15.10. in den Herbstferien, also der 19.10. In UTC gezählt käme
// der 14.10. heraus, ein Schultag.
func TestTagesfrist_ZaehltAbDemBerlinerKalendertag(t *testing.T) {
	ab := time.Date(2026, time.September, 23, 22, 30, 0, 0, time.UTC)
	got := Tagesfrist(ab, 21, lmfplan.Hessen())
	if want := time.Date(2026, time.October, 19, 23, 59, 59, 0, schoolLocation()); !got.Equal(want) {
		t.Errorf("Frist %s, erwartet %s", got.In(schoolLocation()), want)
	}
}
