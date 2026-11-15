package adressabgleich

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBerechneZuordnung(t *testing.T) {
	const ph = "@littera.invalid"
	eintraege := []Eintrag{
		{Nachname: "Beispiel", Vorname: "Anna", Kuerzel: "ABE", Adressen: []string{"Anna.Beispiel@Schule.example"}},
		{Nachname: "Muster", Vorname: "Max", Adressen: []string{"max1@schule.example"}},
		{Nachname: "Muster", Vorname: "Max", Adressen: []string{"max2@schule.example"}},
		{Nachname: "Doppel", Vorname: "Paul", Adressen: []string{"paul@schule.example"}},
		{Nachname: "Vergeben", Vorname: "Lena", Adressen: []string{"lena@schule.example"}},
		{Nachname: "Privat", Vorname: "Nur", Adressen: []string{"nur@mail.example"}},
		{Nachname: "Zwei", Vorname: "Adressen", Adressen: []string{"a1@schule.example", "a2@schule.example"}},
		{Nachname: "Eins", Vorname: "Gleich", Adressen: []string{"gleich@schule.example"}},
		{Nachname: "Zwei", Vorname: "Gleich", Adressen: []string{"gleich@schule.example"}},
		{Nachname: "Groß", Vorname: "Jörg", Adressen: []string{"jg@schule.example", "jg.privat@mail.example"}},
		{Nachname: "Echt", Vorname: "Eva", Adressen: []string{"eva.neu@schule.example"}},
		{Nachname: "Ruhend", Vorname: "Rita", Adressen: []string{"rita@schule.example"}},
		{Nachname: "", Vorname: "", Adressen: []string{"sekretariat@schule.example"}},
		{Nachname: "Unterdomain", Vorname: "Uwe", Adressen: []string{"uwe@x.schule.example.evil"}},
	}
	konten := []Konto{
		{ID: "anna", Vorname: "Anna", Nachname: "Beispiel", Email: "littera-1@littera.invalid", Aktiv: true},
		{ID: "max", Vorname: "Max", Nachname: "Muster", Email: "littera-2@littera.invalid", Aktiv: true},
		{ID: "erika", Vorname: "Erika", Nachname: "Ohne", Email: "littera-3@littera.invalid", Aktiv: true},
		{ID: "paul-ph", Vorname: "Paul", Nachname: "Doppel", Email: "littera-4@littera.invalid", Aktiv: true},
		{ID: "paul-echt", Vorname: "Paul", Nachname: "Doppel", Email: "p.doppel@schule.example", Aktiv: false},
		{ID: "lena", Vorname: "Lena", Nachname: "Vergeben", Email: "littera-5@littera.invalid", Aktiv: true},
		{ID: "lena-selbst", Vorname: "L.", Nachname: "Selbst", Email: "LENA@schule.example", Aktiv: false},
		{ID: "nur", Vorname: "Nur", Nachname: "Privat", Email: "littera-6@littera.invalid", Aktiv: true},
		{ID: "zwei", Vorname: "Adressen", Nachname: "Zwei", Email: "littera-7@littera.invalid", Aktiv: true},
		{ID: "gleich1", Vorname: "Gleich", Nachname: "Eins", Email: "littera-8@littera.invalid", Aktiv: true},
		{ID: "gleich2", Vorname: "Gleich", Nachname: "Zwei", Email: "littera-9@littera.invalid", Aktiv: true},
		// Unicode in anderer Form (o + Trema) und andere Schreibung
		{ID: "joerg", Vorname: "jörg ", Nachname: " GROSS", Email: "littera-10@littera.invalid", Aktiv: true},
		{ID: "joerg2", Vorname: "Jörg", Nachname: "Groß", Email: "littera-11@littera.invalid", Aktiv: true},
		{ID: "eva", Vorname: "Eva", Nachname: "Echt", Email: "eva@schule.example", Aktiv: true},
		{ID: "rita", Vorname: "Rita", Nachname: "Ruhend", Email: "littera-12@littera.invalid", Aktiv: false},
		{ID: "uwe", Vorname: "Uwe", Nachname: "Unterdomain", Email: "littera-13@littera.invalid", Aktiv: true},
	}

	z := BerechneZuordnung(eintraege, konten, "schule.example", ph)

	gotUebernahmen := map[string]string{}
	for _, u := range z.Uebernahmen {
		gotUebernahmen[u.KontoID] = u.Adresse
	}
	wantUebernahmen := map[string]string{
		"anna":   "anna.beispiel@schule.example",
		"joerg2": "jg@schule.example",
	}
	if diff := cmp.Diff(wantUebernahmen, gotUebernahmen); diff != "" {
		t.Errorf("BerechneZuordnung, Übernahmen (-erwartet +ist):\n%s", diff)
	}

	gotGruende := map[string]string{}
	for _, u := range z.Uebergangen {
		gotGruende[u.KontoID] = u.Grund
	}
	wantGruende := map[string]string{
		"max":     grundNameMehrfach,
		"erika":   grundNichtImAdressbuch,
		"paul-ph": grundMehrereKonten,
		"lena":    grundAdresseVergeben,
		"nur":     grundNichtImAdressbuch,
		"zwei":    grundMehrereSchuladressen,
		"gleich1": grundAdresseDoppelt,
		"gleich2": grundAdresseDoppelt,
		"joerg":   grundMehrereKonten,
		"uwe":     grundNichtImAdressbuch,
	}
	if diff := cmp.Diff(wantGruende, gotGruende); diff != "" {
		t.Errorf("BerechneZuordnung, übergangene Konten (-erwartet +ist):\n%s", diff)
	}
	// Aktive Konten mit Platzhalter: alle außer paul-echt, lena-selbst, eva (echte Adresse) und
	// rita (inaktiv).
	if z.KontenMitPlatzhalter != 12 {
		t.Errorf("KontenMitPlatzhalter = %d, erwartet 12", z.KontenMitPlatzhalter)
	}
}

func TestBerechneZuordnung_NeverTouchesRealAddress(t *testing.T) {
	eintraege := []Eintrag{{Nachname: "Echt", Vorname: "Eva", Adressen: []string{"eva.neu@schule.example"}}}
	konten := []Konto{{ID: "eva", Vorname: "Eva", Nachname: "Echt", Email: "eva@schule.example", Aktiv: true}}

	z := BerechneZuordnung(eintraege, konten, "schule.example", "@littera.invalid")
	if len(z.Uebernahmen) != 0 || len(z.Uebergangen) != 0 || z.KontenMitPlatzhalter != 0 {
		t.Errorf("BerechneZuordnung mit echter Adresse = %+v, erwartet keine Änderung und keinen Eintrag", z)
	}
}

func TestBerechneZuordnung_WithoutPlaceholderDomainChangesNothing(t *testing.T) {
	eintraege := []Eintrag{{Nachname: "Beispiel", Vorname: "Anna", Adressen: []string{"anna@schule.example"}}}
	konten := []Konto{{ID: "anna", Vorname: "Anna", Nachname: "Beispiel", Email: "littera-1@littera.invalid", Aktiv: true}}

	if z := BerechneZuordnung(eintraege, konten, "schule.example", ""); len(z.Uebernahmen) != 0 {
		t.Errorf("BerechneZuordnung ohne Platzhalter-Domain = %+v, erwartet keine Übernahme", z.Uebernahmen)
	}
}

func TestBerechneZuordnung_AbbreviationBeforeName(t *testing.T) {
	eintraege := []Eintrag{
		// Name im Adressbuch nach der Heirat geändert; das Kürzel blieb.
		{Nachname: "Neu", Vorname: "Nora", Kuerzel: "NOR", Adressen: []string{"nora@schule.example"}},
		{Nachname: "Ka", Vorname: "Kai", Kuerzel: "KA", Adressen: []string{"kai1@schule.example"}},
		{Nachname: "Ka", Vorname: "Karl", Kuerzel: "KA", Adressen: []string{"karl@schule.example"}},
		{Nachname: "Name", Vorname: "Nur", Kuerzel: "NN", Adressen: []string{"nur@schule.example"}},
	}
	konten := []Konto{
		{ID: "nora", Vorname: "Nora", Nachname: "Alt", Email: "littera-1@littera.invalid", Aktiv: true, LDAPKennung: "nor"},
		{ID: "kai", Vorname: "Kai", Nachname: "Ka", Email: "littera-2@littera.invalid", Aktiv: true, LDAPKennung: "Ka"},
		// Die Kennung steht nicht als Kürzel im Adressbuch: Es zählt der Name.
		{ID: "nur", Vorname: "Nur", Nachname: "Name", Email: "littera-3@littera.invalid", Aktiv: true, LDAPKennung: "nname"},
	}

	z := BerechneZuordnung(eintraege, konten, "schule.example", "@littera.invalid")

	got := map[string]string{}
	for _, u := range z.Uebernahmen {
		got[u.KontoID] = u.Adresse
	}
	for _, u := range z.Uebergangen {
		got[u.KontoID] = u.Grund
	}
	want := map[string]string{
		"nora": "nora@schule.example",
		"kai":  grundKuerzelMehrfach,
		"nur":  "nur@schule.example",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("BerechneZuordnung mit Kürzeln (-erwartet +ist):\n%s", diff)
	}
}

func TestBerechneZuordnung_NameStages(t *testing.T) {
	eintraege := []Eintrag{
		{Nachname: "Schmidt", Vorname: "Anna Karla", Adressen: []string{"a.schmidt@schule.example"}},
		{Nachname: "Mustermann-Beispiel", Vorname: "Erika", Adressen: []string{"emustermann-beispiel@schule.example"}},
		{Nachname: "Müller", Vorname: "Jürgen", Adressen: []string{"jmueller@schule.example"}},
		{Nachname: "von Berg", Vorname: "Clara", Adressen: []string{"cberg@schule.example"}},
		{Nachname: "Lang", Vorname: "Tom", Adressen: []string{"tlang@schule.example"}},
		{Nachname: "Lang-Kurz", Vorname: "Tom", Adressen: []string{"tlang-kurz@schule.example"}},
		{Nachname: "Weiß", Vorname: "Ina", Adressen: []string{"iweiss@schule.example"}},
	}
	konten := []Konto{
		// Weiterer Vorname nur im Adressbuch
		{ID: "anna", Vorname: "Anna", Nachname: "Schmidt", Email: "littera-1@littera.invalid", Aktiv: true},
		// Doppelname mit Leerzeichen statt Bindestrich
		{ID: "erika", Vorname: "Erika", Nachname: "Mustermann Beispiel", Email: "littera-2@littera.invalid", Aktiv: true},
		// Umlaut ausgeschrieben, Titel davor
		{ID: "juergen", Vorname: "Juergen", Nachname: "Dr. Mueller", Email: "littera-3@littera.invalid", Aktiv: true},
		// Namenszusatz fehlt: Teil des Nachnamens
		{ID: "clara", Vorname: "Clara", Nachname: "Berg", Email: "littera-4@littera.invalid", Aktiv: true},
		// „Tom Lang" passt genau auf „Lang"; auf „Lang-Kurz" nur als Teil. Die genaue Stufe gewinnt.
		{ID: "tom", Vorname: "Tom", Nachname: "Lang", Email: "littera-5@littera.invalid", Aktiv: true},
		// „Tom Kurz" passt nur als Teil auf „Lang-Kurz"; dieser Eintrag passt aber genauso als Teil zu
		// „Tom Lang". Zwei Konten, kein sicherer Treffer.
		{ID: "tomkurz", Vorname: "Tom", Nachname: "Kurz", Email: "littera-6@littera.invalid", Aktiv: true},
		// Anderer Vorname mit gleichem Anfangsbuchstaben bekommt nichts.
		{ID: "ines", Vorname: "Ines", Nachname: "Weiss", Email: "littera-7@littera.invalid", Aktiv: true},
	}

	z := BerechneZuordnung(eintraege, konten, "schule.example", "@littera.invalid")

	got := map[string]string{}
	for _, u := range z.Uebernahmen {
		got[u.KontoID] = u.Adresse + " (" + u.Weg + ")"
	}
	for _, u := range z.Uebergangen {
		got[u.KontoID] = u.Grund
	}
	want := map[string]string{
		"anna":    "a.schmidt@schule.example (Name in anderer Schreibung)",
		"erika":   "emustermann-beispiel@schule.example (Name in anderer Schreibung)",
		"juergen": "jmueller@schule.example (Name in anderer Schreibung)",
		"clara":   "cberg@schule.example (Teil des Nachnamens)",
		"tom":     "tlang@schule.example (Name)",
		"tomkurz": grundMehrereKonten,
		"ines":    grundNichtImAdressbuch,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("BerechneZuordnung mit Stufen des Namens (-erwartet +ist):\n%s", diff)
	}
}
