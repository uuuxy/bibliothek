package littera

import (
	"strings"
	"testing"
)

// Werte aus dem echten Export (littera_sav.mdb, 04.08.2026).
const gruppenCSV = `Buchungsdatum,Buchungsnummer,KurzBez,Untergruppe,Obergruppe
,1,"UNDEF","Undefinierte Untergruppe",1
,22,"07H1","Schüler",5
,71,"11T1","Sekundarstufe II",7
,90,"Lehrer","Lehrer",1
,91,"Lehrerin","Lehrerin",1
,95,"Ab","Abgegangen",1
,96,"Prakt","Praktikant",1
,97,"FB-Ek","Fachbereich Erdkunde",1
,98,"Ausl","Im Ausland",1
`

func TestArtAusUntergruppe(t *testing.T) {
	gruppen, err := LeseLesergruppen(strings.NewReader(gruppenCSV))
	if err != nil {
		t.Fatalf("Gruppen lesen: %v", err)
	}

	faelle := map[string]struct {
		art    LeserArt
		klasse string
		warum  string
	}{
		"22": {ArtSchueler, "07H1", "regulaere Klasse"},
		// Der Fall, den man leicht uebersieht: Die Oberstufe ist eine EIGENE
		// Untergruppe, aber selbstverstaendlich sind das Schueler.
		"71": {ArtSchueler, "11T1", "Sekundarstufe II sind auch Schueler"},
		"90": {ArtLehrkraft, "Lehrer", "Lehrkraft"},
		"91": {ArtLehrkraft, "Lehrerin", "Lehrkraft"},
		"95": {ArtAbgegangen, "Ab", "ehemalige Schueler"},
		"96": {ArtSonstige, "Prakt", "Praktikant ist weder Schueler noch Lehrkraft"},
		"97": {ArtSonstige, "FB-Ek", "Fachbereich ist ein Sammelkonto, keine Person"},
		// Unklar heisst unklar — NICHT stillschweigend Schueler.
		"98": {ArtUnbekannt, "Ausl", "unklare Untergruppe braucht eine Entscheidung"},
		"1":  {ArtUnbekannt, "UNDEF", "undefiniert"},
	}

	for id, erwartet := range faelle {
		g := gruppen[id]
		if g.Art != erwartet.art {
			t.Errorf("Gruppe %s (%s): Art %d, erwartet %d", id, erwartet.warum, g.Art, erwartet.art)
		}
		if g.Klasse != erwartet.klasse {
			t.Errorf("Gruppe %s: Klasse %q, erwartet %q", id, g.Klasse, erwartet.klasse)
		}
	}
}

const leserCSV = `Buchungsnummer,Lesernummer,Vorname,Nachname,Lesergruppe,Geburtsdatum,eMail,Adresse,PLZ,Ort
1,1001,"Anna","Schuelerin",22,"05/03/95 00:00:00","","Hauptstr. 1","61250","Usingen"
2,1002,"Bert","Oberstufe",71,"01/01/93 00:00:00","",,"61250","Usingen"
3,1003,"Clara","Lehrkraft",91,,"c.lehrkraft@schule.de",,,
4,1004,"Dora","Ehemalig",95,,,,,
5,1005,"Erik","Fachbereich",97,,,,,
6,,"Ohne","Nummer",22,,,,,
`

func TestLeseLeser_OrdnetEinUndFiltert(t *testing.T) {
	gruppen, err := LeseLesergruppen(strings.NewReader(gruppenCSV))
	if err != nil {
		t.Fatalf("Gruppen: %v", err)
	}
	leser, err := LeseLeser(strings.NewReader(leserCSV), gruppen)
	if err != nil {
		t.Fatalf("Leser: %v", err)
	}

	// Die Zeile ohne Lesernummer faellt raus — ohne Schluessel keine Ausleihzuordnung.
	if len(leser) != 5 {
		t.Fatalf("erwartet 5 Leser, waren %d", len(leser))
	}

	if leser[0].Klasse != "07H1" || leser[0].Art != ArtSchueler {
		t.Errorf("Schuelerin falsch eingeordnet: %+v", leser[0])
	}
	if leser[1].Art != ArtSchueler {
		t.Errorf("Oberstufenschueler muss ArtSchueler sein, war %d", leser[1].Art)
	}
	if leser[2].Art != ArtLehrkraft {
		t.Errorf("Lehrkraft falsch eingeordnet: %+v", leser[2])
	}

	// DER Punkt: Lehrkraefte duerfen nicht in der Schuelermenge landen.
	schueler := NurArt(leser, ArtSchueler)
	if len(schueler) != 2 {
		t.Fatalf("erwartet 2 Schueler, waren %d", len(schueler))
	}
	for _, s := range schueler {
		if s.Nachname == "Lehrkraft" {
			t.Error("eine Lehrkraft ist in der Schuelermenge gelandet")
		}
		if s.Nachname == "Fachbereich" {
			t.Error("ein Fachbereichs-Sammelkonto ist in der Schuelermenge gelandet")
		}
	}

	if len(NurArt(leser, ArtLehrkraft)) != 1 {
		t.Error("Lehrkraft-Menge falsch")
	}
	if len(NurArt(leser, ArtAbgegangen)) != 1 {
		t.Error("Abgaenger-Menge falsch")
	}
}

// TestUnbekannteArtWirdNichtZuSchueler haelt die vorsichtige Vorgabe fest: Was nicht
// eindeutig zugeordnet werden kann, wird NICHT stillschweigend zum Schueler gemacht.
// Bei Personendaten ist eine ausgelassene Zeile das kleinere Uebel als eine falsch
// einsortierte.
func TestUnbekannteArtWirdNichtZuSchueler(t *testing.T) {
	gruppen := map[string]Lesergruppe{"98": {Klasse: "Ausl", Art: ArtUnbekannt}}
	const csv = `Lesernummer,Vorname,Nachname,Lesergruppe
2001,"Unklar","Fall",98
`
	leser, err := LeseLeser(strings.NewReader(csv), gruppen)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if len(NurArt(leser, ArtSchueler)) != 0 {
		t.Error("unklare Zuordnung darf nicht als Schueler durchgehen")
	}
	if leser[0].Art != ArtUnbekannt {
		t.Errorf("Art muss ArtUnbekannt bleiben, war %d", leser[0].Art)
	}
}

// TestLeserOhneGruppe: Ein Leser, dessen Lesergruppe nicht auflösbar ist, bekommt
// weder Klasse noch Art — und faellt damit aus jeder Schreibmenge heraus.
func TestLeserOhneGruppe(t *testing.T) {
	const csv = `Lesernummer,Vorname,Nachname,Lesergruppe
3001,"Ohne","Gruppe",9999
`
	leser, err := LeseLeser(strings.NewReader(csv), map[string]Lesergruppe{})
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if leser[0].Art != ArtUnbekannt || leser[0].Klasse != "" {
		t.Errorf("nicht aufloesbare Gruppe: %+v", leser[0])
	}
}
