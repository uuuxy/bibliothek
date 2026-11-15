package adressabgleich

import (
	"context"
	"errors"
	"os"
	"testing"
)

// Abruf gegen ein echtes Roundcube, etwa ein Wegwerf-Container am Arbeitsplatz:
//
//	ROUNDCUBE_TEST_ADRESSE=http://127.0.0.1:8099 ROUNDCUBE_TEST_BENUTZER=probe \
//	ROUNDCUBE_TEST_PASSWORT=geheim go test -run Roundcube ./internal/adressabgleich/
//
// Ohne die Variablen läuft der Test nicht; den Ablauf prüft dann der Nachbau in webmail_test.go.
// Ein frisches Roundcube hat kein globales Adressbuch, deshalb lädt der Test das persönliche.
func TestHoleAdressbuch_AgainstRealRoundcube(t *testing.T) {
	adresse := os.Getenv("ROUNDCUBE_TEST_ADRESSE")
	if adresse == "" {
		t.Skip("ROUNDCUBE_TEST_ADRESSE nicht gesetzt")
	}
	zugang := Zugang{Adresse: adresse, Benutzer: os.Getenv("ROUNDCUBE_TEST_BENUTZER"), Passwort: os.Getenv("ROUNDCUBE_TEST_PASSWORT")}

	vcf, err := holeAdressbuch(context.Background(), zugang, "0")
	if err != nil {
		t.Fatalf("holeAdressbuch: %v", err)
	}
	eintraege := LeseAdressbuch(vcf)
	t.Logf("%d Einträge im persönlichen Adressbuch", len(eintraege))

	falsch := zugang
	falsch.Passwort += "-falsch"
	if _, err := holeAdressbuch(context.Background(), falsch, "0"); !errors.Is(err, ErrAnmeldungAbgelehnt) {
		t.Errorf("holeAdressbuch mit falschem Passwort = %v, erwartet ErrAnmeldungAbgelehnt", err)
	}
	if _, err := HoleGlobalesAdressbuch(context.Background(), zugang); !errors.Is(err, ErrKeinGlobalesAdressbuch) {
		t.Errorf("HoleGlobalesAdressbuch ohne globales Adressbuch = %v, erwartet ErrKeinGlobalesAdressbuch", err)
	}
}
