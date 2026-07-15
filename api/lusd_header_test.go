package api

import "testing"

func TestNormalizeHeader(t *testing.T) {
	cases := map[string]string{
		"Vorname":                    "vorname",
		"  Nachname  ":               "nachname",
		"Schüler_Vorname":            "schuelervorname",
		"Schueler_Vorname":           "schuelervorname",
		"Straße":                     "strasse",
		"PLZ":                        "plz",
		"E-Mail":                     "email",
		"Ansprechpartner_Email":      "ansprechpartneremail",
		"Klassen_Klassenbezeichnung": "klassenklassenbezeichnung",
	}
	for in, want := range cases {
		if got := normalizeHeader(in); got != want {
			t.Errorf("normalizeHeader(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestLusdHeaderMap_PraefixExportStyle(t *testing.T) {
	// "Individueller Bericht" mit Tabellen-Präfixen (Fall A).
	headers := []string{"Schueler_Vorname", "Schueler_Nachname", "Klassen_Klassenbezeichnung", "lusd_id", "Ansprechpartner_Email"}

	hm, err := lusdHeaderMap(headers)
	if err != nil {
		t.Fatalf("Präfix-Header sollen erkannt werden, Fehler: %v", err)
	}
	if hm[lusdColVorname] != 0 || hm[lusdColNachname] != 1 || hm[lusdColKlasse] != 2 || hm[lusdColID] != 3 {
		t.Errorf("Pflichtspalten falsch gemappt: %v", hm)
	}
	if hm[lusdColElternEmail] != 4 {
		t.Errorf("Ansprechpartner_Email soll auf eltern_email mappen: %v", hm)
	}
}

func TestLusdHeaderMap_MissingLusdIDStillErrors(t *testing.T) {
	// Ohne stabile LUSD-ID lässt sich kein Abgleich fahren → Pflichtspalte.
	headers := []string{"vorname", "nachname", "klasse", "strasse"}

	if _, err := lusdHeaderMap(headers); err == nil {
		t.Fatal("fehlende LUSD-ID soll einen Fehler liefern")
	}
}

func TestParseLUSDCSV_PraefixHeadersWithAddress(t *testing.T) {
	// Kompletter Durchlauf mit Präfix-Headern und Umlaut-Adresse.
	csv := "lusd_id;Schueler_Vorname;Schueler_Nachname;Klasse;Straße;Hausnummer;PLZ;Wohnort;Email\n" +
		"L1;Max;Mustermann;5a;Hauptstraße;12;63500;Seligenstadt;eltern@example.de\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	got := rows[0]
	if got.Vorname != "Max" || got.Strasse != "Hauptstraße" || got.PLZ != "63500" ||
		got.Ort != "Seligenstadt" || got.ElternEmail != "eltern@example.de" {
		t.Errorf("Präfix-Export mit Adresse falsch geparst: %+v", got)
	}
}
