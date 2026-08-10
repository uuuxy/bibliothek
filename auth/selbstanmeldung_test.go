package auth

import "testing"

// Die Domain-Prüfung ist die einzige Schranke zwischen „hat irgendein Postfach" und
// „steht im Bibliothekssystem". Sie muss die naheliegenden Umgehungen aushalten —
// deshalb steht hier eine Liste von Angriffsformen und nicht nur ein Positivfall.
func TestDarfSichSelbstAnmelden(t *testing.T) {
	t.Setenv(selbstanmeldeDomainEnv, "philipp-reis-schule.de")

	faelle := []struct {
		email  string
		erlaub bool
		warum  string
	}{
		{"peter.flasch@philipp-reis-schule.de", true, "regulärer Fall"},
		{"Peter.Flasch@Philipp-Reis-Schule.DE", true, "Groß-/Kleinschreibung ist egal"},
		{"a@fremde-schule.de", false, "fremde Domain"},
		{"a@boesephilipp-reis-schule.de", false, "Suffix-Falle: endet auf die Domain, ist sie aber nicht"},
		{"a@philipp-reis-schule.de.angreifer.net", false, "Domain steckt nur mittendrin"},
		{"a@angreifer.net/philipp-reis-schule.de", false, "Contains-Falle"},
		{"peter.flasch@philipp-reis-schule.de@angreifer.net", false, "zweites @ hängt hinten dran"},
		{"ohne-at-zeichen", false, "gar keine Domain"},
		{"@philipp-reis-schule.de", false, "leerer Adressteil"},
	}

	for _, f := range faelle {
		if got := darfSichSelbstAnmelden(f.email); got != f.erlaub {
			t.Errorf("%s: darfSichSelbstAnmelden(%q) = %v, erwartet %v", f.warum, f.email, got, f.erlaub)
		}
	}
}

// Ohne gesetzte Domain ist die Selbstanmeldung AUS — auch für die Adresse, die sonst
// erlaubt wäre. Sichere Vorgabe: Wer die Einstellung nicht kennt, schaltet sie nicht
// versehentlich ein.
func TestSelbstanmeldungIstOhneEinstellungAus(t *testing.T) {
	t.Setenv(selbstanmeldeDomainEnv, "")

	if darfSichSelbstAnmelden("peter.flasch@philipp-reis-schule.de") {
		t.Error("ohne SELBSTANMELDUNG_DOMAIN darf sich niemand selbst anlegen")
	}
	if got := SelbstanmeldungStatus(); got == "" {
		t.Error("der Zustand muss protokollierbar sein, sonst fällt eine vergessene Einstellung nicht auf")
	}
}

// Das @ darf auch mit führendem Zeichen konfiguriert sein — ein naheliegender Tippfehler
// in der .env soll die Funktion nicht stillschweigend abschalten.
func TestDomainMitFuehrendemAt(t *testing.T) {
	t.Setenv(selbstanmeldeDomainEnv, "@philipp-reis-schule.de")

	if !darfSichSelbstAnmelden("peter.flasch@philipp-reis-schule.de") {
		t.Error("SELBSTANMELDUNG_DOMAIN mit führendem @ muss genauso wirken")
	}
}

func TestNamenAusAdresse(t *testing.T) {
	faelle := []struct{ email, vorname, nachname string }{
		{"peter.flasch@philipp-reis-schule.de", "Peter", "Flasch"},
		{"anna.maria.weber@schule.de", "Anna", "Maria.weber"}, // nur am ERSTEN Punkt geteilt
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
