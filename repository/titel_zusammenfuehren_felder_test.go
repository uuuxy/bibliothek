package repository

import (
	"errors"
	"slices"
	"testing"
)

func text(s string) *string { return &s }

// Leer heißt unbekannt und wird aufgefüllt, auch ein Feld aus Leerraum; was am Ziel steht,
// bleibt. Der Schulzweig ist die Ausnahme: Leer heißt dort „gilt für alle".
func TestFuegeTitelZusammen_FuelltLeeresAuf(t *testing.T) {
	ziel := titelStand{
		id: "z", titel: "Mathematik 5", medientyp: "Buch", erweitert: "{}",
		untertitel: text("  "), autor: text("Anna Autorin"),
	}
	quelle := titelStand{
		id: "q", titel: "Mathematik 5", medientyp: "Buch", erweitert: `{"littera_eigentumsvermerk": "Land"}`,
		isbn: text("9783060130764"), untertitel: text("Arbeitsheft"), autor: text("A. Autorin"),
		zweig: text("G"), jahrgangVon: text("5"), jahrgangBis: text("6"), zaehldatum: text("2026-10-01"),
		coverURL: text("/uploads/cover/q.jpg"), coverStatus: text("FOUND"),
	}

	nachher, uebernommen, abweichend, err := fuegeTitelZusammen(ziel, quelle)
	if err != nil {
		t.Fatalf("fuegeTitelZusammen: %v", err)
	}
	if wert(nachher.isbn) != "9783060130764" || wert(nachher.untertitel) != "Arbeitsheft" ||
		wert(nachher.autor) != "Anna Autorin" || nachher.zweig != nil ||
		wert(nachher.jahrgangVon) != "5" || wert(nachher.coverStatus) != "FOUND" {
		t.Errorf("nachher: %+v", nachher)
	}
	if nachher.erweitert != quelle.erweitert {
		t.Errorf("weitere Angaben %q, erwartet die der Quelle zum Ergänzen", nachher.erweitert)
	}
	want := []TitelFeld{
		{"ISBN", "9783060130764"}, {"Untertitel", "Arbeitsheft"}, {"Zähldatum", "01.10.2026"},
		{"Jahrgang", "5–6"}, {"Cover", ""},
	}
	if !slices.Equal(uebernommen, want) {
		t.Errorf("übernommen %+v, erwartet %+v", uebernommen, want)
	}
	wantAb := []TitelAbweichung{{"Schulzweig", "alle", "G"}, {"Autor", "Anna Autorin", "A. Autorin"}}
	if !slices.Equal(abweichend, wantAb) {
		t.Errorf("abweichend %+v, erwartet %+v", abweichend, wantAb)
	}
}

// Titel, Art und Lernmittel füllt die Quelle nie auf; sind sie verschieden, stehen sie vorn,
// weil sie bestimmen, was das Ziel danach ist.
func TestFuegeTitelZusammen_NenntAbweichungenVorn(t *testing.T) {
	ziel := titelStand{titel: "Natur und Technik 7/10", medientyp: "Buch", istLernmittel: true, verlag: text("Cornelsen"), listenpreis: text("28.99")}
	quelle := titelStand{titel: "Natur und Technik - Biologie", medientyp: "Arbeitsheft", verlag: text("Klett"), listenpreis: text("12.50")}

	_, _, abweichend, err := fuegeTitelZusammen(ziel, quelle)
	if err != nil {
		t.Fatalf("fuegeTitelZusammen: %v", err)
	}
	want := []TitelAbweichung{
		{"Titel", "Natur und Technik 7/10", "Natur und Technik - Biologie"},
		{"Art des Mediums", "Buch", "Arbeitsheft"},
		{"Lernmittel", "ja", "nein"},
		{"Verlag", "Cornelsen", "Klett"},
		{"Listenpreis", "28,99 €", "12,50 €"},
	}
	if !slices.Equal(abweichend, want) {
		t.Errorf("abweichend %+v, erwartet %+v", abweichend, want)
	}
}

// Zwei ISBN sind zwei Ausgaben.
func TestFuegeTitelZusammen_ZweiISBN(t *testing.T) {
	_, _, _, err := fuegeTitelZusammen(titelStand{isbn: text("9783060158423")}, titelStand{isbn: text("9783060130764")})
	if !errors.Is(err, ErrTitelZusammenfuehrenZweiISBN) {
		t.Fatalf("Fehler %v, erwartet ErrTitelZusammenfuehrenZweiISBN", err)
	}
}
