package adressabgleich

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLeseAdressbuch(t *testing.T) {
	// So schreibt Roundcube den Export, ergänzt um Faltung, Gruppe, Maskierung und Parameter in
	// Anführungszeichen.
	vcf := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"N:Beispiel;Anna;;;ABE\r\n" +
		"FN:Anna Beispiel\r\n" +
		"EMAIL;TYPE=INTERNET:anna.beispiel@schule.example\r\n" +
		"CATEGORIES:Kollegium\r\n" +
		"END:VCARD\r\n" +
		"BEGIN:VCARD\n" +
		"VERSION:3.0\n" +
		"N:Mül\n ler-Lang;Max\\, Moritz;;;MÜL\n" +
		"item1.EMAIL;TYPE=\"internet:pref\":max@schule.example\n" +
		"EMAIL:max.privat@mail.example\n" +
		"END:VCARD\n" +
		"BEGIN:VCARD\n" +
		"FN:Sekretariat\n" +
		"EMAIL:sekretariat@schule.example\n" +
		"END:VCARD\n" +
		"EMAIL:ausserhalb@schule.example\n" +
		"BEGIN:VCARD\n" +
		"N:Ohne\\;Strichpunkt;Erika\n" +
		"END:VCARD\n"

	got := LeseAdressbuch([]byte(vcf))
	want := []Eintrag{
		{Nachname: "Beispiel", Vorname: "Anna", Kuerzel: "ABE", Adressen: []string{"anna.beispiel@schule.example"}},
		{Nachname: "Müller-Lang", Vorname: "Max, Moritz", Kuerzel: "MÜL", Adressen: []string{"max@schule.example", "max.privat@mail.example"}},
		{Adressen: []string{"sekretariat@schule.example"}},
		{Nachname: "Ohne;Strichpunkt", Vorname: "Erika"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LeseAdressbuch (-erwartet +ist):\n%s", diff)
	}
}

func TestLeseAdressbuch_EmptyAndGarbage(t *testing.T) {
	for _, in := range []string{"", "kein vCard", "BEGIN:VCARD\nN:Offen;Ohne;Ende\n"} {
		if got := LeseAdressbuch([]byte(in)); len(got) != 0 {
			t.Errorf("LeseAdressbuch(%q) = %v, erwartet keine Einträge", in, got)
		}
	}
}
