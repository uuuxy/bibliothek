package bereitschaft

import (
	"slices"
	"testing"
	"time"

	"bibliothek/pkg/logger"
)

// Was als Punkt zählt, entscheidet, ob die Mail wieder hinausgeht. Zählten Zahlen oder Uhrzeiten
// mit, wäre jede Mail neu, und der Wächter schickte nach jedem Update dieselbe noch einmal.
func TestAlarmPunkte(t *testing.T) {
	befunde := []Befund{
		{Bereich: "Auslagerung der Backups", Stufe: StufeKritisch, Befund: "Kein Ziel außer Haus eingerichtet."},
		{Bereich: "Nächtliches Backup", Stufe: StufeKritisch, Befund: "Letztes Backup vor 30 Stunden."},
		{Bereich: "Demo-Daten", Stufe: StufeWarnung, Befund: "Demo-Daten im Bestand."},
		{Bereich: "Anmeldung", Stufe: StufeOK, Befund: "Anmeldung läuft."},
	}
	gruppen := []logger.Fehlergruppe{
		{Meldung: "Voriger Lauf des Servers ist abgestürzt", Anzahl: 1, Ursache: "jobs/fehlerstand.go:61: panic: x"},
		{Meldung: "Anfrage fehlgeschlagen", Route: "GET /api/katalog/{id}", Anzahl: 3},
		{Sammel: true, Anzahl: 7, Ursache: "andere Meldung: y"},
	}
	want := []string{
		"Befund: Auslagerung der Backups",
		"Befund: Nächtliches Backup",
		"Fehler: Anfrage fehlgeschlagen GET /api/katalog/{id}",
		"Fehler: Voriger Lauf des Servers ist abgestürzt",
		"Fehler: weitere Meldungen",
	}
	if got := AlarmPunkte(befunde, gruppen, false); !slices.Equal(got, want) {
		t.Fatalf("AlarmPunkte =\n%q\nerwartet\n%q", got, want)
	}

	// Ein Backup, das älter wurde, und Meldungen, die öfter kamen, machen keinen neuen Punkt.
	befunde[1].Befund = "Letztes Backup vor 31 Stunden."
	gruppen[0].Anzahl, gruppen[0].Zuletzt, gruppen[0].Ursache = 2, time.Now(), "api/probe.go:7: panic: y"
	gruppen[1].Anzahl = 40
	if got := AlarmPunkte(befunde, gruppen, false); !slices.Equal(got, want) {
		t.Errorf("mit neuen Zahlen, Uhrzeiten und Ursachen =\n%q\nerwartet unverändert\n%q", got, want)
	}
}

// Mit dem Schalter alarm_warnungen zählen auch die Warnungen als Punkte, als eigene Art; ohne
// ihn nicht, sonst löste eine Warnung, die die Mail nicht nennt, eine neue Mail aus.
func TestAlarmPunkte_MitWarnungen(t *testing.T) {
	befunde := []Befund{
		{Bereich: "Demo-Daten", Stufe: StufeWarnung},
		{Bereich: "Littera-Übernahme", Stufe: StufeKritisch},
	}
	gruppen := []logger.Fehlergruppe{
		{Meldung: "LUSD-Abgleich abgewiesen", Route: "POST /api/lusd/import", Anzahl: 2, Warnung: true},
		{Sammel: true, Warnung: true, Anzahl: 3},
		{Meldung: "Anfrage mit Serverfehler beendet", Anzahl: 1},
	}
	if got, want := AlarmPunkte(befunde, gruppen, false), []string{
		"Befund: Littera-Übernahme",
		"Fehler: Anfrage mit Serverfehler beendet",
	}; !slices.Equal(got, want) {
		t.Errorf("ohne Schalter =\n%q\nerwartet\n%q", got, want)
	}
	if got, want := AlarmPunkte(befunde, gruppen, true), []string{
		"Befund: Littera-Übernahme",
		"Fehler: Anfrage mit Serverfehler beendet",
		"Warnung im Log: LUSD-Abgleich abgewiesen POST /api/lusd/import",
		"Warnung im Log: weitere Meldungen",
		"Warnung: Demo-Daten",
	}; !slices.Equal(got, want) {
		t.Errorf("mit Schalter =\n%q\nerwartet\n%q", got, want)
	}
}

// Unverändertes geht einmal am Tag hinaus, ein neuer Punkt beim nächsten Lauf. Ein Punkt, der
// wegfiel, ist nichts Neues.
func TestIstMailFaellig(t *testing.T) {
	jetzt := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	auslagerung := "Befund: Auslagerung der Backups"
	absturz := "Fehler: Voriger Lauf des Servers ist abgestürzt"
	vor := func(abstand time.Duration, punkte ...string) GemeldeterAlarm {
		return GemeldeterAlarm{Zeit: jetzt.Add(-abstand), Punkte: punkte}
	}
	faelle := []struct {
		name    string
		zuletzt GemeldeterAlarm
		punkte  []string
		want    bool
	}{
		{"nie gemeldet", GemeldeterAlarm{}, []string{auslagerung}, true},
		{"unverändert seit zwei Stunden", vor(2*time.Hour, auslagerung), []string{auslagerung}, false},
		{"unverändert seit 22 Stunden 59 Minuten", vor(23*time.Hour-time.Minute, auslagerung), []string{auslagerung}, false},
		{"unverändert seit einem Tag, der Takt kam Sekunden früher", vor(24*time.Hour-5*time.Second, auslagerung), []string{auslagerung}, true},
		{"neuer Punkt", vor(2*time.Hour, auslagerung), []string{absturz, auslagerung}, true},
		{"ein Punkt fiel weg", vor(2*time.Hour, absturz, auslagerung), []string{auslagerung}, false},
	}
	for _, f := range faelle {
		if got := IstMailFaellig(f.zuletzt, f.punkte, jetzt); got != f.want {
			t.Errorf("%s: IstMailFaellig = %v, erwartet %v", f.name, got, f.want)
		}
	}
}
