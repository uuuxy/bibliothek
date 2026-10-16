package lusd

import "testing"

func TestNormiereKopf(t *testing.T) {
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
		if got := normiereKopf(in); got != want {
			t.Errorf("normiereKopf(%q) = %q; erwartet %q", in, got, want)
		}
	}
}

func TestOrdneSpaltenZu_ExportMitPraefix(t *testing.T) {
	// "Individueller Bericht" mit Tabellen-Präfixen (Fall A).
	headers := []string{"Schueler_Vorname", "Schueler_Nachname", "Klassen_Klassenbezeichnung", "lusd_id", "Ansprechpartner_Email"}

	hm, err := ordneSpaltenZu(headers)
	if err != nil {
		t.Fatalf("Präfix-Header sollen erkannt werden, Fehler: %v", err)
	}
	if hm[spalteVorname] != 0 || hm[spalteNachname] != 1 || hm[spalteKlasse] != 2 || hm[spalteID] != 3 {
		t.Errorf("Pflichtspalten falsch gemappt: %v", hm)
	}
	if hm[spalteElternEmail] != 4 {
		t.Errorf("Ansprechpartner_Email soll auf eltern_email mappen: %v", hm)
	}
}

func TestOrdneSpaltenZu_OhneLUSDIDIstErlaubt(t *testing.T) {
	// Der Export der Schule hat keine Schüler-ID (LANIS-Klassenliste: Nachname;Vorname;
	// Klasse) — die Kopfzeile ist gültig, den Modus bestimmt der Parser (Nur-Name).
	headers := []string{"vorname", "nachname", "klasse", "strasse"}

	if _, err := ordneSpaltenZu(headers); err != nil {
		t.Fatalf("Kopfzeile ohne LUSD-ID muss gültig sein, war: %v", err)
	}
}

func TestLeseCSV_PraefixKoepfeMitAnschrift(t *testing.T) {
	// Kompletter Durchlauf mit Präfix-Headern und Umlaut-Adresse.
	csv := "lusd_id;Schueler_Vorname;Schueler_Nachname;Klasse;Straße;Hausnummer;PLZ;Wohnort;Email\n" +
		"L1;Max;Mustermann;5a;Hauptstraße;12;63500;Seligenstadt;eltern@example.de\n"

	rows, _, err := leseCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	got := rows[0]
	if got.Vorname != "Max" || got.Strasse != "Hauptstraße" || got.PLZ != "63500" ||
		got.Ort != "Seligenstadt" || got.ElternEmail != "eltern@example.de" {
		t.Errorf("Präfix-Export mit Adresse falsch geparst: %+v", got)
	}
}

// Der dritte Export-Stil: eine Klassenliste mit LUSD-Feldkürzeln (kein
// LUSD-Export) trägt Tabellen-KÜRZEL — SLR_ für Schüler, KLA_ für Klasse. Genau diese Kopfzeile, Wort für Wort;
// die Zeilen darunter sind erfunden.
func TestOrdneSpaltenZu_KuerzelDerSchule(t *testing.T) {
	headers := []string{
		"SLR_Nachname", "SLR_Vorname", "SLR_Strasse", "SLR_PLZ", "SLR_ORT",
		"KLA_Klassennamen", "KLA_KlassenlehrerKuerzel", "KLA_KlassenlehrerName",
		"SLR_Schulzweig", "Widerspruch zur Empf. ZK 1.HJ", "SLR_Begründungstext",
	}
	m, err := ordneSpaltenZu(headers)
	if err != nil {
		t.Fatalf("die Kopfzeile im SLR_/KLA_-Kürzelstil wird abgewiesen: %v", err)
	}
	erwartet := map[string]int{
		spalteNachname: 0, spalteVorname: 1, spalteStrasse: 2, spaltePLZ: 3, spalteOrt: 4, spalteKlasse: 5,
	}
	for col, idx := range erwartet {
		if m[col] != idx {
			t.Errorf("%s: Spalte %d erwartet, %d bekommen", col, idx, m[col])
		}
	}
	if _, ok := m[spalteID]; ok {
		t.Error("eine LUSD-ID gibt es in diesem Export nicht — darf nicht erraten werden")
	}
	if _, ok := m[spalteGeburtsdatum]; ok {
		t.Error("kein Geburtsdatum in dieser Datei — der Import muss in den Nur-Name-Modus")
	}
}

// Der echte LUSD-Bericht „All_Inklusiv" (ein Demo-Export der Schule) schreibt die
// Postleitzahl als „Schueler_Postleitzahl", weder „PLZ" noch „Schueler_PLZ". Ohne diese
// Schreibweise käme jeder Schüler aus diesem Bericht still ohne Postleitzahl an.
func TestOrdneSpaltenZu_AllInklusivBericht(t *testing.T) {
	headers := []string{
		"Schueler_Schluessel", "Schueler_Nachname", "Schueler_Vorname", "Schueler_Geburtsdatum",
		"Klassen_Klassenbezeichnung", "Schueler_Straße", "Schueler_Postleitzahl", "Schueler_Ort",
	}
	m, err := ordneSpaltenZu(headers)
	if err != nil {
		t.Fatalf("Header-Map: %v", err)
	}
	for _, col := range []string{spalteNachname, spalteVorname, spalteGeburtsdatum, spalteKlasse, spalteStrasse, spaltePLZ, spalteOrt} {
		if _, ok := m[col]; !ok {
			t.Errorf("Spalte %q nicht zugeordnet", col)
		}
	}
	if _, ok := m[spalteID]; ok {
		t.Errorf("Schueler_Schluessel (Name,Vorname,Datum) darf NICHT als LUSD-ID gelten")
	}
}
