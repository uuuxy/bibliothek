package system

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/bereitschaft"
	"bibliothek/pkg/logger"

	"github.com/google/go-cmp/cmp"
)

// Der Wortlaut, den Seite und Mail zeigen: die jüngste Meldung zuerst, je Meldung Anzahl, Zeit
// und Ursache, die Zeile zu einer Anfrage mit ihrer Route, die Sammelgruppe unter eigenem Namen,
// und die Zeit vom Tag der Abfrage aus. Nur für die Mail: erste und letzte Zeit mit Datum und die
// letzte Zeile im Log.
func TestBaueFehlerImLog_Wortlaut(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("Zeitzone: %v", err)
	}
	jetzt := time.Date(2026, 10, 24, 15, 40, 0, 0, berlin)
	gruppen := []logger.Fehlergruppe{
		// In UTC und nicht nach Zeit geordnet, wie sie ankommen können.
		{Meldung: "Backup: S3 upload failed", Anzahl: 1, Erstmals: time.Date(2026, 10, 24, 0, 30, 0, 0, time.UTC),
			Zuletzt: time.Date(2026, 10, 24, 0, 31, 0, 0, time.UTC), Ursache: "403 AccessDenied",
			Zeile: `{"time":"2026-10-24T00:31:00Z","level":"ERROR","msg":"Backup: S3 upload failed","fehler":"403 AccessDenied"}`},
		{Meldung: "Anfrage mit Serverfehler beendet", Route: "GET /api/buecher/titel/{id}", Anzahl: 2,
			Erstmals: time.Date(2026, 10, 24, 9, 10, 0, 0, time.UTC), Zuletzt: time.Date(2026, 10, 24, 9, 47, 0, 0, time.UTC),
			Ursache: "titel laden: verbindung zur datenbank verloren"},
		{Meldung: "Mahnlauf: Mail nicht zugestellt", Anzahl: 3, Erstmals: time.Date(2026, 10, 23, 13, 0, 0, 0, time.UTC),
			Zuletzt: time.Date(2026, 10, 23, 14, 10, 0, 0, time.UTC), Ursache: "dial tcp: i/o timeout"},
		{Sammel: true, Anzahl: 4, Erstmals: time.Date(2026, 10, 24, 4, 0, 0, 0, time.UTC),
			Zuletzt: time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC), Ursache: "Etikett: Vorlage fehlt"},
	}

	a := baueFehlerImLog(gruppen, jetzt, 0)
	if a.Stufe != bereitschaft.StufeKritisch || a.Anzahl != 10 {
		t.Errorf("Stufe %q mit Anzahl %d, erwartet %q mit 10", a.Stufe, a.Anzahl, bereitschaft.StufeKritisch)
	}
	if want := "10 Fehlermeldungen in den letzten 24 Stunden, zuletzt heute um 11:47."; a.Befund != want {
		t.Errorf("Befund = %q, erwartet %q", a.Befund, want)
	}
	want := []fehlerZeile{
		{Text: "Anfrage mit Serverfehler beendet: GET /api/buecher/titel/{id}", Zusatz: "2 × · zuletzt 11:47 · titel laden: verbindung zur datenbank verloren",
			Zeitraum: "erstmals gegen 24.10.2026 11:10, zuletzt 24.10.2026 11:47:00"},
		{Text: "Weitere Meldungen, zusammen gezählt", Zusatz: "4 × · zuletzt 07:00 · Etikett: Vorlage fehlt",
			Zeitraum: "erstmals gegen 24.10.2026 06:00, zuletzt 24.10.2026 07:00:00"},
		{Text: "Backup: S3 upload failed", Zusatz: "1 × · 02:31 · 403 AccessDenied", Zeitraum: "am 24.10.2026 02:31:00",
			Logzeile: `{"time":"2026-10-24T00:31:00Z","level":"ERROR","msg":"Backup: S3 upload failed","fehler":"403 AccessDenied"}`},
		{Text: "Mahnlauf: Mail nicht zugestellt", Zusatz: "3 × · zuletzt gestern 16:10 · dial tcp: i/o timeout",
			Zeitraum: "erstmals gegen 23.10.2026 15:00, zuletzt 23.10.2026 16:10:00"},
	}
	if diff := cmp.Diff(want, a.Meldungen); diff != "" {
		t.Errorf("Meldungen (-erwartet +ist):\n%s", diff)
	}
	if a.Folge == "" || a.Abhilfe == "" {
		t.Errorf("Folge %q, Abhilfe %q, erwartet beide", a.Folge, a.Abhilfe)
	}
}

// Eine einzelne Zeile steht ohne „zuletzt"; eine ältere Zeit mit Datum.
func TestBaueFehlerImLog_EineZeileVorgestern(t *testing.T) {
	jetzt := time.Date(2026, 10, 26, 0, 5, 0, 0, time.UTC)
	a := baueFehlerImLog([]logger.Fehlergruppe{
		{Meldung: "A", Anzahl: 1, Zuletzt: time.Date(2026, 10, 24, 23, 58, 0, 0, time.UTC)},
	}, jetzt, 0)
	if want := "1 Fehlermeldung in den letzten 24 Stunden, am 24.10. um 23:58."; a.Befund != want {
		t.Errorf("Befund = %q, erwartet %q", a.Befund, want)
	}
	if want := "1 × · 24.10. 23:58"; len(a.Meldungen) != 1 || a.Meldungen[0].Zusatz != want {
		t.Errorf("Meldungen = %+v, erwartet eine mit Zusatz %q", a.Meldungen, want)
	}
}

// Die Seite nennt alle Meldungen; die Mail höchstens hoechstzahlMeldungenInDerMail, die Zeilen der
// übrigen zählt eine letzte, die auf die Seite zeigt.
func TestBaueFehlerImLog_HoechstzahlNurInDerMail(t *testing.T) {
	jetzt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	var gruppen []logger.Fehlergruppe
	for i := range hoechstzahlMeldungenInDerMail + 2 {
		gruppen = append(gruppen, logger.Fehlergruppe{
			Meldung: fmt.Sprintf("Meldung %d", i), Anzahl: i + 1, Zuletzt: jetzt.Add(-time.Duration(i) * time.Minute),
		})
	}

	if seite := baueFehlerImLog(gruppen, jetzt, 0); len(seite.Meldungen) != len(gruppen) {
		t.Errorf("Seite: %d Meldungen, erwartet alle %d", len(seite.Meldungen), len(gruppen))
	}
	mail := baueFehlerImLog(gruppen, jetzt, hoechstzahlMeldungenInDerMail)
	if len(mail.Meldungen) != hoechstzahlMeldungenInDerMail+1 {
		t.Fatalf("Mail: %d Zeilen, erwartet %d Meldungen und eine für den Rest", len(mail.Meldungen), hoechstzahlMeldungenInDerMail+1)
	}
	// Die übrigen zwei Meldungen tragen hoechstzahl+1 und hoechstzahl+2 Zeilen.
	rest := 2*hoechstzahlMeldungenInDerMail + 3
	if got, want := mail.Meldungen[hoechstzahlMeldungenInDerMail], (fehlerZeile{
		Text: fmt.Sprintf("%d weitere Fehlermeldungen unter System → Einstellungen → Fehler im Log", rest),
	}); got != want {
		t.Errorf("letzte Zeile = %+v, erwartet %+v", got, want)
	}
	n := len(gruppen)
	if want := fmt.Sprintf("%d Fehlermeldungen in den letzten 24 Stunden, zuletzt heute um 12:00.", n*(n+1)/2); mail.Befund != want {
		t.Errorf("Befund = %q, erwartet %q", mail.Befund, want)
	}
}

// Gezählt und nichts gefunden ist in Ordnung; zählt der Server nicht, sagt die Seite das.
func TestBaueFehlerImLog_OhneZeileUndNichtErhoben(t *testing.T) {
	leer := baueFehlerImLog([]logger.Fehlergruppe{}, time.Now(), 0)
	if leer.Stufe != bereitschaft.StufeOK || leer.Befund != "Keine Fehlermeldung in den letzten 24 Stunden." || len(leer.Meldungen) != 0 {
		t.Errorf("ohne Zeile = %+v, erwartet ok ohne Meldungen", leer)
	}
	nicht := baueFehlerImLog(nil, time.Now(), 0)
	if nicht.Stufe != bereitschaft.StufeWarnung || !strings.HasPrefix(nicht.Befund, "Nicht erhoben") {
		t.Errorf("ohne Fehlerstand = %+v, erwartet die Warnung „nicht erhoben“", nicht)
	}
}

// Die Tür liest den Fehlerstand der Türen: mit ihm die Zeilen samt Ursache, ohne ihn „nicht
// erhoben"; die Liste ist nie null, die Seite zählt sie. Warnungen zeigt die Seite nicht, und die
// ganze Logzeile bleibt in der Mail.
func TestFehlerImLog_Tuer(t *testing.T) {
	stand := logger.NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.DiscardHandler))
	log.Error("Backup: S3 upload failed", "fehler", "403 AccessDenied")
	log.Warn("Vorgabe statt Einstellung")
	nurWarnung := logger.NewFehlerstand()
	slog.New(nurWarnung.SchalteVor(slog.DiscardHandler)).Warn("Vorgabe statt Einstellung")

	faelle := []struct {
		name        string
		fehlerstand *logger.Fehlerstand
		stufe       string
		meldungen   int
	}{
		{"mit Fehlerstand", stand, bereitschaft.StufeKritisch, 1},
		{"nur eine Warnung", nurWarnung, bereitschaft.StufeOK, 0},
		{"ohne Fehlerstand", nil, bereitschaft.StufeWarnung, 0},
	}
	for _, f := range faelle {
		rec := httptest.NewRecorder()
		(&Tueren{Fehlerstand: f.fehlerstand}).FehlerImLog().
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/admin/system/fehler-im-log", nil))
		if !strings.Contains(rec.Body.String(), `"meldungen":[`) {
			t.Errorf("%s: Antwort %s, erwartet eine Liste der Meldungen", f.name, rec.Body.String())
		}
		var a fehlerImLogAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatalf("%s: Antwort %d unlesbar: %v", f.name, rec.Code, err)
		}
		if a.Stufe != f.stufe || len(a.Meldungen) != f.meldungen {
			t.Errorf("%s: Antwort = %+v, erwartet Stufe %q mit %d Meldungen", f.name, a, f.stufe, f.meldungen)
		}
		if f.meldungen > 0 && !strings.Contains(a.Meldungen[0].Zusatz, "403 AccessDenied") {
			t.Errorf("%s: Meldung = %+v, erwartet die Ursache im Zusatz", f.name, a.Meldungen[0])
		}
		if strings.Contains(rec.Body.String(), "Vorgabe statt Einstellung") || strings.Contains(rec.Body.String(), `\"level\"`) {
			t.Errorf("%s: Antwort %s nennt eine Warnung oder eine ganze Logzeile", f.name, rec.Body.String())
		}
	}
}

// Warnungen fasst die Mail wie Fehler, mit eigenem Wortlaut und ohne Verweis auf die Seite, die
// nur Fehler zeigt. trenneWarnungen teilt die Gruppen und lässt nil als „nicht erhoben" stehen.
func TestBaueWarnungenImLog_UndTrennung(t *testing.T) {
	jetzt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	gruppen := []logger.Fehlergruppe{
		{Meldung: "Fehler A", Anzahl: 1, Zuletzt: jetzt},
		{Meldung: "LUSD-Abgleich abgewiesen", Route: "POST /api/lusd/import", Anzahl: 2, Warnung: true,
			Erstmals: jetzt.Add(-time.Hour), Zuletzt: jetzt.Add(-time.Minute), Ursache: "In der Kopfzeile fehlt die Pflichtspalte",
			Zeile: `{"level":"WARN","msg":"LUSD-Abgleich abgewiesen"}`},
	}
	fehler, warnungen := trenneWarnungen(gruppen)
	if len(fehler) != 1 || fehler[0].Meldung != "Fehler A" || len(warnungen) != 1 || !warnungen[0].Warnung {
		t.Fatalf("Trennung: Fehler %+v, Warnungen %+v", fehler, warnungen)
	}
	if f, w := trenneWarnungen(nil); f != nil || w != nil {
		t.Errorf("nil: %v / %v, erwartet nil (der Server zählt nicht)", f, w)
	}
	if f, w := trenneWarnungen([]logger.Fehlergruppe{}); f == nil || w == nil {
		t.Errorf("leer: %v / %v, erwartet leere Listen (gezählt, nichts gefunden)", f, w)
	}

	a := baueWarnungenImLog(warnungen, jetzt, hoechstzahlMeldungenInDerMail)
	if a.Stufe != bereitschaft.StufeWarnung || a.Anzahl != 2 {
		t.Errorf("Stufe %q, Anzahl %d; erwartet warnung und 2", a.Stufe, a.Anzahl)
	}
	if want := "2 Warnungen in den letzten 24 Stunden, zuletzt heute um 11:59."; a.Befund != want {
		t.Errorf("Befund = %q, erwartet %q", a.Befund, want)
	}
	if len(a.Meldungen) != 1 || a.Meldungen[0].Text != "LUSD-Abgleich abgewiesen: POST /api/lusd/import" ||
		a.Meldungen[0].Logzeile != `{"level":"WARN","msg":"LUSD-Abgleich abgewiesen"}` ||
		a.Meldungen[0].Zeitraum != "erstmals gegen 24.10.2026 11:00, zuletzt 24.10.2026 11:59:00" {
		t.Errorf("Meldungen = %+v", a.Meldungen)
	}
	if !strings.Contains(a.Abhilfe, `"level":"WARN"`) {
		t.Errorf("Abhilfe ohne den Befehl für Warnungen: %q", a.Abhilfe)
	}
	if leer := baueWarnungenImLog(nil, jetzt, 0); leer.Anzahl != 0 || leer.Befund != "" {
		t.Errorf("ohne Warnungen = %+v, erwartet nichts zu melden", leer)
	}
	var viele []logger.Fehlergruppe
	for i := range 3 {
		viele = append(viele, logger.Fehlergruppe{Meldung: fmt.Sprintf("W%d", i), Anzahl: 1, Zuletzt: jetzt, Warnung: true})
	}
	if m := baueWarnungenImLog(viele, jetzt, 2).Meldungen; len(m) != 3 || m[2].Text != "1 weitere Warnung im Log" {
		t.Errorf("Rest der Warnungen = %+v, erwartet „1 weitere Warnung im Log“", m)
	}
}
