package littera

import (
	"strings"
	"testing"
)

// TestOhneZuordnungNenntGruppeUndZahlen: Der Halt vor dem ersten Schreiben muss sagen, WELCHE
// Gruppe in Littera zuzuordnen ist und was an ihr hängt — Personen UND Ausleihen, denn die
// Ausleihen sind der Schaden (das Buch gälte als verfügbar). Ein Name steht dabei nirgends.
func TestOhneZuordnungNenntGruppeUndZahlen(t *testing.T) {
	undef := func(id string) Leser {
		return Leser{ID: id, Vorname: "Vorname" + id, Nachname: "Nachname" + id, Klasse: "UNDEF",
			Gruppe: "Undefinierte Untergruppe", GruppeNr: "9", Art: ArtUnbekannt}
	}
	ab := &Altbestand{
		Leser: []Leser{
			{ID: "S", Vorname: "Vorname-S", Nachname: "Nachname-S", Klasse: "07H1", Art: ArtSchueler},
			{ID: "X", Vorname: "Vorname-X", Nachname: "Nachname-X", GruppeNr: "10"}, // Gruppe fehlt in Leser_UG
			undef("U1"),
			undef("U2"),
		},
		Ausleihen: []Ausleihe{
			{ID: "1", LeserID: "U1"}, {ID: "2", LeserID: "U1"}, {ID: "3", LeserID: "U2"},
			{ID: "4", LeserID: "X"},
			{ID: "5", LeserID: "S"}, {ID: "6", LeserID: "S"},
		},
	}

	befund := OhneZuordnung(ab)
	if len(befund) != 2 {
		t.Fatalf("zwei Gruppen ohne Zuordnung erwartet, gefunden: %+v", befund)
	}
	// Reihenfolge nach dem Schlüssel als Zahl: 9 vor 10 (als Text stünde 10 vorn).
	erwartet := []string{
		"Lesergruppe 9 „Undefinierte Untergruppe“ (UNDEF): 2 Personen, 3 Ausleihen",
		"Lesergruppe 10 (fehlt in Leser_UG): 1 Person, 1 Ausleihe",
	}
	for i, b := range befund {
		if b.String() != erwartet[i] {
			t.Errorf("Befund %d:\n  ist  %q\n  soll %q", i, b.String(), erwartet[i])
		}
		if strings.Contains(b.String(), "Vorname") || strings.Contains(b.String(), "Nachname") {
			t.Errorf("der Befund darf keinen Namen tragen: %q", b.String())
		}
	}
}

// TestOhneZuordnungLeerWennAlleZugeordnet: Jede bekannte Art — auch Sonstige und Abgegangen —
// läuft durch; der Halt gilt nur für Leser ohne Art.
func TestOhneZuordnungLeerWennAlleZugeordnet(t *testing.T) {
	ab := &Altbestand{Leser: []Leser{
		{ID: "1", Art: ArtSchueler}, {ID: "2", Art: ArtLehrkraft}, {ID: "3", Art: ArtLiV},
		{ID: "4", Art: ArtAbgegangen}, {ID: "5", Art: ArtSonstige},
	}}
	if befund := OhneZuordnung(ab); len(befund) != 0 {
		t.Errorf("kein Befund erwartet, gefunden: %+v", befund)
	}
}
