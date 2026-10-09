package littera

import (
	"strings"
	"testing"

	"bibliothek/pkg/mitteltopf"
)

func TestZuordnungAusVermerk(t *testing.T) {
	faelle := []struct {
		vermerk            string
		bekannt            bool
		wortlaut, eigentum string
	}{
		{"Land Hessen", true, "Land Hessen", mitteltopf.Land},
		// 33 Mal klein geschrieben: derselbe Vermerk, aufgehoben in der üblichen Form.
		{"land hessen", true, "Land Hessen", mitteltopf.Land},
		{"  Land   Hessen ", true, "Land Hessen", mitteltopf.Land},
		{"Hochtaunuskreis", true, "Hochtaunuskreis", mitteltopf.Schultraeger},
		// Bekannt, aber ohne Zuordnung, bis die Schule sie bestätigt (OFFEN.md 4.24).
		{"Philipp-Reis-Schule", true, "Philipp-Reis-Schule", ""},
		{"Bibliothek", true, "Bibliothek", ""},
		{"Förderverein", true, "Förderverein", ""},
		{"Info Schulprojekt", true, "Info Schulprojekt", ""},
		{"Dauerleihgabe", true, "Dauerleihgabe", ""},
		{"", true, "", ""},
		// Freitext außerhalb der Liste kommt nicht mit — auch kein Teilwort.
		{"Erika Mustermann", false, "", ""},
		{"Land", false, "", ""},
		{"Eigentum Land Hessen", false, "", ""},
	}
	for _, f := range faelle {
		z, bekannt := zuordnungAusVermerk(f.vermerk)
		if bekannt != f.bekannt || z.Wortlaut != f.wortlaut || z.Eigentum != f.eigentum {
			t.Errorf("zuordnungAusVermerk(%q) = (%+v, %v), erwartet (%q, %q, %v)",
				f.vermerk, z, bekannt, f.wortlaut, f.eigentum, f.bekannt)
		}
	}
}

// Jede Zuordnung der Liste muss die Bedingung chk_exemplar_eigentum bestehen — sonst bräche
// der Titel samt Exemplaren an der Datenbank ab, und das erst beim einen echten Lauf.
func TestVermerkeLittera_NurGueltigesEigentum(t *testing.T) {
	for schluessel, z := range vermerkeLittera {
		if z.Eigentum != "" && !mitteltopf.Gueltig(z.Eigentum) {
			t.Errorf("%q ordnet %q zu — nicht im Vokabular von chk_exemplar_eigentum", schluessel, z.Eigentum)
		}
		if got, _ := zuordnungAusVermerk(z.Wortlaut); got != z {
			t.Errorf("der aufgehobene Wortlaut %q findet seine eigene Zuordnung nicht", z.Wortlaut)
		}
	}
}

func TestLeseExemplare_Eigentumsvermerk(t *testing.T) {
	const csv = `Buchungsnummer,Titel,Barcode,Sig1,Sig2,Eigentumsvermerk
1,2,"B-1","Ga","Bos"," Land Hessen "
2,2,"B-2","Ga","Bos",""
`
	ex, err := LeseExemplare(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}
	if len(ex) != 2 || ex[0].Eigentumsvermerk != "Land Hessen" || ex[1].Eigentumsvermerk != "" {
		t.Errorf("Eigentumsvermerk nicht gelesen: %+v", ex)
	}
}
