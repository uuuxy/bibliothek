package api

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/repository"
)

// Die selbst bearbeiteten Vorgänge stehen je Tag und Handlung in einer Zeile, jede Uhrzeit
// bleibt erhalten — in der Schulzeitzone, damit ein Vorgang kurz nach Mitternacht nicht am
// Vortag steht.
func TestDsgvoVorgaengeJeTag(t *testing.T) {
	zeit := func(s string) *time.Time {
		z, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return &z
	}
	ip := "10.0.0.7"
	vorgaenge := []repository.DsgvoEigenerVorgang{
		{Zeitpunkt: zeit("2026-09-23T22:30:00Z"), Handlung: "Ausleihe gebucht"}, // 00:30 am 24.09. in Berlin
		{Zeitpunkt: zeit("2026-09-23T09:05:00Z"), Handlung: "Protokolleintrag: CHECKOUT (ausleihen)"},
		{Zeitpunkt: zeit("2026-09-23T09:00:00Z"), Handlung: "Ausleihe gebucht"},
		{Zeitpunkt: zeit("2026-09-23T08:00:00Z"), Handlung: "Verwaltungseingriff: OVERRIDE_BLOCK", IPAdresse: &ip},
		{Zeitpunkt: zeit("2026-09-23T07:59:00Z"), Handlung: "Ausleihe gebucht"},
		{Handlung: "Rückgabe gebucht"},
	}
	want := []dsgvoVorgangsZeile{
		{tag: "24.09.2026", handlung: "Ausleihe gebucht", zeiten: []string{"00:30"}},
		{tag: "23.09.2026", handlung: "Protokolleintrag: CHECKOUT (ausleihen)", zeiten: []string{"11:05"}},
		{tag: "23.09.2026", handlung: "Ausleihe gebucht", zeiten: []string{"11:00", "09:59"}},
		{tag: "23.09.2026", handlung: "Verwaltungseingriff: OVERRIDE_BLOCK", zeiten: []string{"10:00 (IP 10.0.0.7)"}},
		{tag: "ohne gespeicherten Zeitpunkt", handlung: "Rückgabe gebucht", zeiten: []string{"—"}},
	}
	if got := dsgvoVorgaengeJeTag(vorgaenge); !reflect.DeepEqual(got, want) {
		t.Errorf("Zeilen:\n got %+v\nwant %+v", got, want)
	}
}
