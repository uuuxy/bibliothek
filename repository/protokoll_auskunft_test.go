package repository

import "testing"

// Jeder Schlüssel, den ein Protokolleintrag neben der Kennung eines Lesers oder eines Kontos
// trägt, ist für das Blatt der Auskunft eingeordnet: mit Bezeichnung (protokollAngaben) oder als
// Kennung, buchendes Konto oder Merker (protokollOhneAngabe). Ohne Einordnung stünde der Schlüssel
// mit seinem Namen auf dem Blatt, bei einer neuen Kennung samt ihrem Wert.
//
// Blind für: dieselben Formen wie die Ratsche der Tilgung (sammleStellenMit); die Einträge an der
// Leserzeile selbst (Papierkorb, Zusammenführen), die keine Kennung in details tragen; die
// Schlüssel verschachtelter Objekte. Dafür steht der Test am echten Weg,
// api.TestDsgvoBlatt_ProtokollzeilenInWorten.
func TestProtokollSchluessel_FuerDieAuskunftEingeordnet(t *testing.T) {
	for _, merkmal := range []string{"schueler_id", "ziel_id"} {
		stellen := sammleStellenMit(t, merkmal)
		if len(stellen) < 3 {
			t.Fatalf("nur %d Maps mit %s gefunden — der Detektor sieht sie nicht mehr", len(stellen), merkmal)
		}
		for _, s := range stellen {
			if _, ausgenommen := keinProtokoll[s.ort]; ausgenommen {
				continue
			}
			for _, k := range s.schluessel {
				if _, mitAngabe := protokollAngabeJeSchluessel[k]; mitAngabe || ProtokollOhneAngabe(k) {
					continue
				}
				t.Errorf("%s (%s): Schlüssel %q neben %s ist für die Auskunft nicht eingeordnet. Nennt er etwas "+
					"über die Person oder den Vorgang, bekommt er eine Bezeichnung in protokollAngaben "+
					"(repository/protokoll_auskunft.go); eine Kennung des Programms oder das buchende Konto "+
					"steht mit Grund in protokollOhneAngabe.", s.ort, s.art, k, merkmal)
			}
		}
	}
}

func TestProtokollSchluessel_AuskunftsListenSindSauber(t *testing.T) {
	gesehen := map[string]bool{}
	for _, a := range protokollAngaben {
		if gesehen[a.Schluessel] {
			t.Errorf("%q steht zweimal in protokollAngaben", a.Schluessel)
		}
		gesehen[a.Schluessel] = true
		if a.Bezeichnung == "" {
			t.Errorf("%q hat keine Bezeichnung", a.Schluessel)
		}
		if ProtokollOhneAngabe(a.Schluessel) {
			t.Errorf("%q steht in protokollAngaben und in protokollOhneAngabe", a.Schluessel)
		}
	}
	ohne := map[string]bool{}
	for _, o := range protokollOhneAngabe {
		if ohne[o.Schluessel] {
			t.Errorf("%q steht zweimal in protokollOhneAngabe", o.Schluessel)
		}
		ohne[o.Schluessel] = true
		if o.Grund == "" {
			t.Errorf("%q steht ohne Grund in protokollOhneAngabe", o.Schluessel)
		}
	}
	// Was die Tilgung als Wert der Person führt, steht auf dem Blatt. ziel_id ist die Kennung des
	// Kontos; das Blatt nennt sie beim Konto.
	for _, k := range protokollSchluesselMitPersonenbezug {
		if k != "ziel_id" && !gesehen[k] {
			t.Errorf("%q trägt nach protokollSchluesselMitPersonenbezug einen Wert der Person, hat aber keine "+
				"Bezeichnung für die Auskunft", k)
		}
	}
	if a, _ := ProtokollAngabeZu("barcode_id", true, "UPDATE"); a.Bezeichnung != "Ausweisnummer" {
		t.Errorf("barcode_id an der Leserzeile heißt %q, erwartet „Ausweisnummer“", a.Bezeichnung)
	}
	if a, _ := ProtokollAngabeZu("barcode_id", false, "DELETE"); a.Bezeichnung != "Buchnummer" {
		t.Errorf("barcode_id an einem Buch heißt %q, erwartet „Buchnummer“", a.Bezeichnung)
	}
	for aktion, eigene := range protokollBezeichnungJeAktion {
		for k, bezeichnung := range eigene {
			if !gesehen[k] {
				t.Errorf("protokollBezeichnungJeAktion[%s] nennt %q, den protokollAngaben nicht kennt", aktion, k)
			}
			if a, _ := ProtokollAngabeZu(k, false, aktion); a.Bezeichnung != bezeichnung {
				t.Errorf("%s im Eintrag %s heißt %q, erwartet %q", k, aktion, a.Bezeichnung, bezeichnung)
			}
		}
	}
}
