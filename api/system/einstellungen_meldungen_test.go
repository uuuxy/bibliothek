package system

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
)

// Was die Einstellungen ablehnen, sagt die Meldung mit dem Namen der Einstellung, dem
// eingegebenen Wert und der Regel, gegen die er verstößt. Der Test hält jeden Satz fest, den
// das Formular zeigen kann.
func TestEinstellungenSchreiben_MeldungJePruefung(t *testing.T) {
	faelle := []struct {
		name    string
		rumpf   string
		meldung string
	}{
		{"Zahlen außerhalb der Spanne", `{"max_overdue_items": 999999, "frist_buch_tage": 0}`,
			"Nichts gespeichert. Leihfrist Bücher (Tage): 0 liegt außerhalb von 1 bis 365; " +
				"Max. überfällige Titel: 999999 liegt außerhalb von 1 bis 100."},
		{"Eingangsjahrgang mit Schrägstrich", `{"lmf_eingangsjahrgaenge": "8/9"}`,
			`Eingangsjahrgänge: "8/9" ist kein Jahrgang zwischen 1 und 13 — erwartet wird etwa „5, 7“`},
		{"Stichtag mit Punkten", `{"lmf_stichtag": "30.06."}`,
			`Lernmittel-Stichtag: "30.06." ist kein Kalendertag im Format MM-TT — erwartet wird etwa „07-31“ für den 31. Juli`},
		{"Sommerferien unlesbar", `{"sommerferien": "kaputt"}`,
			"Sommerferien: die Liste ist nicht lesbar"},
		{"Sommerferien doppelt", sommerferienRumpf(`{"jahr":2027,"von":"2027-06-28","bis":"2027-08-06"},{"jahr":2027,"von":"2027-06-28","bis":"2027-08-06"}`),
			"Sommerferien 2027: das Jahr steht zweimal in der Liste"},
		{"Sommerferien Datum unlesbar", sommerferienRumpf(`{"jahr":2027,"von":"2027-13-01","bis":"2027-08-06"}`),
			"Sommerferien 2027: Datum nicht lesbar (JJJJ-MM-TT)"},
		{"Sommerferien Zahlendreher", sommerferienRumpf(`{"jahr":2131,"von":"2131-06-28","bis":"2131-08-06"}`),
			"Sommerferien 2131: das Jahr liegt außerhalb von 2000 bis 2100. Ein Zahlendreher?"},
		{"Sommerferien im falschen Jahr", sommerferienRumpf(`{"jahr":2027,"von":"2028-06-28","bis":"2028-08-06"}`),
			"Sommerferien 2027: Beginn und Ende müssen im Jahr 2027 liegen"},
		{"Sommerferien Ende vor Beginn", sommerferienRumpf(`{"jahr":2027,"von":"2027-08-06","bis":"2027-06-28"}`),
			"Sommerferien 2027: das Ende liegt nicht nach dem Beginn"},
		{"Sommerferien zu kurz", sommerferienRumpf(`{"jahr":2027,"von":"2027-07-01","bis":"2027-07-10"}`),
			"Sommerferien 2027: 10 Tage sind keine Sommerferien (erwartet 4 bis 9 Wochen)"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &einstellungsAttrappe{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/einstellungen", strings.NewReader(f.rumpf))
			(&Tueren{Basis: &tuer.Basis{}}).EinstellungenSchreiben(repo).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("EinstellungenSchreiben: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
			if got := tuertest.Fehlermeldung(t, rec); got != f.meldung {
				t.Errorf("EinstellungenSchreiben: Meldung %q, erwartet %q", got, f.meldung)
			}
			if repo.gespeichert {
				t.Error("EinstellungenSchreiben hat trotz Ablehnung gespeichert")
			}
		})
	}
}

// sommerferienRumpf setzt Einträge in den Rumpf: Die Einstellung ist eine JSON-Liste in einer
// Zeichenkette.
func sommerferienRumpf(eintraege string) string {
	return `{"sommerferien": "[` + strings.ReplaceAll(eintraege, `"`, `\"`) + `]"}`
}
