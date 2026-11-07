package auth

import (
	"strings"
	"testing"
)

// Die Prüfung der Domain ist die einzige Schranke zwischen „hat irgendein Postfach" und „steht
// im Programm". Sie muss die naheliegenden Umgehungen aushalten; deshalb steht hier eine Liste
// von Angriffen und nicht nur ein Fall, der gelingt.
func TestDarfSichSelbstAnmelden(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "philipp-reis-schule.de")

	faelle := []struct {
		email  string
		erlaub bool
		warum  string
	}{
		{"nina.berger@philipp-reis-schule.de", true, "regulärer Fall"},
		{"Nina.Berger@Philipp-Reis-Schule.DE", true, "Groß-/Kleinschreibung ist egal"},
		{"n@philipp-reis-schule.de", true, "ein Zeichen vor dem @ reicht"},
		{"a@fremde-schule.de", false, "fremde Domain"},
		{"a@boesephilipp-reis-schule.de", false, "Suffix-Falle: endet auf die Domain, ist sie aber nicht"},
		{"a@philipp-reis-schule.de.angreifer.net", false, "Domain steckt nur mittendrin"},
		{"a@angreifer.net/philipp-reis-schule.de", false, "Contains-Falle"},
		{"nina.berger@philipp-reis-schule.de@angreifer.net", false, "zweites @ hängt hinten dran"},
		{"ohne-at-zeichen", false, "gar keine Domain"},
		{"@philipp-reis-schule.de", false, "leerer Adressteil"},
	}

	for _, f := range faelle {
		if got := darfSichSelbstAnmelden(f.email); got != f.erlaub {
			t.Errorf("%s: darfSichSelbstAnmelden(%q) = %v, erwartet %v", f.warum, f.email, got, f.erlaub)
		}
	}
}

// Ohne gesetzte Domain ist die Selbstanmeldung aus, auch für die Adresse, die sonst erlaubt
// wäre. Eine sichere Vorgabe: Wer die Einstellung nicht kennt, schaltet sie nicht versehentlich
// ein.
func TestSelbstanmeldungIstOhneEinstellungAus(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "")

	if darfSichSelbstAnmelden("nina.berger@philipp-reis-schule.de") {
		t.Error("ohne SELBSTANMELDUNG_DOMAIN darf sich niemand selbst anlegen")
	}
	if got := SelbstanmeldungStatus(); !strings.Contains(got, "abgeschaltet") {
		t.Errorf("SelbstanmeldungStatus() = %q, erwartet „abgeschaltet“: sonst fällt eine vergessene Einstellung nicht auf", got)
	}
}

// Die Domain darf mit führendem @ eingetragen sein: Ein naheliegender Tippfehler in der .env
// soll die Selbstanmeldung nicht unbemerkt abschalten.
func TestDomainMitFuehrendemAt(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "@philipp-reis-schule.de")

	if !darfSichSelbstAnmelden("nina.berger@philipp-reis-schule.de") {
		t.Error("SELBSTANMELDUNG_DOMAIN mit führendem @ muss genauso wirken")
	}
}

func TestNamenAusAdresse(t *testing.T) {
	faelle := []struct{ email, vorname, nachname string }{
		{"nina.berger@philipp-reis-schule.de", "Nina", "Berger"},
		{"anna.maria.weber@schule.de", "Anna", "Maria.weber"}, // nur am ersten Punkt geteilt
		{"sekretariat@schule.de", "", "Sekretariat"},
		{"j.doe@schule.de", "J", "Doe"},
		{"", "", ""},
	}

	for _, f := range faelle {
		v, n := namenAusAdresse(f.email)
		if v != f.vorname || n != f.nachname {
			t.Errorf("namenAusAdresse(%q) = %q/%q, erwartet %q/%q", f.email, v, n, f.vorname, f.nachname)
		}
	}
}
