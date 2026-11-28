package littera

import (
	"strings"
	"testing"
)

// Werte aus dem echten Export (littera_sav.mdb).
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
,122,"Sekr.","Sekretärin",79
,123,"IMPORT"," IMPORT",1
,129,"U+","U-plus",74
`

func TestArtAusUntergruppe(t *testing.T) {
	gruppen, err := leseLesergruppen(strings.NewReader(gruppenCSV), nil, nil)
	if err != nil {
		t.Fatalf("Gruppen lesen: %v", err)
	}

	faelle := map[string]struct {
		art    LeserArt
		klasse string
		warum  string
	}{
		"22": {ArtSchueler, "07H1", "regulaere Klasse"},
		// Der Fall, den man leicht uebersieht: Die Oberstufe ist eine eigene
		// Untergruppe, aber selbstverstaendlich sind das Schueler.
		"71": {ArtSchueler, "11T1", "Sekundarstufe II sind auch Schueler"},
		"90": {ArtLehrkraft, "Lehrer", "Lehrkraft"},
		"91": {ArtLehrkraft, "Lehrerin", "Lehrkraft"},
		"95": {ArtAbgegangen, "Ab", "ehemalige Schueler"},
		// Keine Schüler, aber Leser mit Ausleihen: ins Kollegium, je mit eigener Art.
		"96":  {ArtPraktikum, "Prakt", "Praktikant ist weder Schueler noch Lehrkraft"},
		"97":  {ArtFachbereich, "FB-Ek", "Fachbereich ist ein Sammelkonto, keine Person"},
		"122": {ArtSekretariat, "Sekr.", "Sekretariat"},
		"129": {ArtUPlus, "U+", "U-plus: Vertretungskraefte"},
		// Schueler auf Zeit im Ausland: Schueler mit der Littera-Klasse.
		"98": {ArtSchueler, "Ausl", "Im Ausland bleibt Schueler"},
		// Ohne Zuordnung heisst ohne Zuordnung — nicht stillschweigend Schueler; der Lauf
		// haelt an (OhneZuordnung). Die Bezeichnung zaehlt ohne Leerraum („ IMPORT").
		"1":   {ArtUnbekannt, "UNDEF", "undefiniert"},
		"123": {ArtUnbekannt, "IMPORT", "Importgruppe ohne Zuordnung"},
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

// Referendare heißen in Hessen LiV (Lehrkraft im Vorbereitungsdienst). Littera führt sie als
// eigene Gruppe; sie gehören ins Kollegium, nicht unter „unklar".
func TestReferendareSindLiV(t *testing.T) {
	for _, bezeichnung := range []string{"Referendar", "Referendarin"} {
		if art := artAusUntergruppe(bezeichnung); art != ArtLiV {
			t.Errorf("Untergruppe %q: Art %d, erwartet ArtLiV", bezeichnung, art)
		}
	}
}

// Buchungsnummer ist der interne Schlüssel (Ziel von Verleih.Leser), Lesernummer die
// Ausweisnummer. Die letzte Zeile hat keine Buchungsnummer — sie faellt heraus.
const leserCSV = `Buchungsnummer,Lesernummer,Vorname,Nachname,Lesergruppe,Geburtsdatum,eMail,Adresse,PLZ,Ort
1,1001,"Anna","Schuelerin",22,"05/03/95 00:00:00","","Hauptstr. 1","61250","Usingen"
2,1002,"Bert","Oberstufe",71,"01/01/93 00:00:00","",,"61250","Usingen"
3,1003,"Clara","Lehrkraft",91,,"c.lehrkraft@schule.de",,,
4,1004,"Dora","Ehemalig",95,,,,,
5,1005,"Erik","Fachbereich",97,,,,,
,1006,"Ohne","Schluessel",22,,,,,
`

func TestLeseLeser_OrdnetEinUndFiltert(t *testing.T) {
	gruppen, err := leseLesergruppen(strings.NewReader(gruppenCSV), nil, nil)
	if err != nil {
		t.Fatalf("Gruppen: %v", err)
	}
	leser, err := leseLeser(strings.NewReader(leserCSV), gruppen)
	if err != nil {
		t.Fatalf("Leser: %v", err)
	}

	// Die Zeile ohne Buchungsnummer faellt raus — ohne internen Schluessel laesst sich
	// keine Ausleihe zuordnen.
	if len(leser) != 5 {
		t.Fatalf("len(leser) = %d, erwartet 5 Leser", len(leser))
	}

	if leser[0].Klasse != "07H1" || leser[0].Art != ArtSchueler {
		t.Errorf("Schuelerin falsch eingeordnet: %+v", leser[0])
	}
	// Beide Nummern muessen getrennt ankommen: ID verknuepft die Ausleihen,
	// Lesernummer wird zum Ausweis-Barcode.
	if leser[0].ID != "1" || leser[0].Lesernummer != "1001" {
		t.Errorf("Schluessel vertauscht: ID=%q Lesernummer=%q", leser[0].ID, leser[0].Lesernummer)
	}
	if leser[1].Art != ArtSchueler {
		t.Errorf("Oberstufenschueler muss ArtSchueler sein, war %d", leser[1].Art)
	}
	if leser[2].Art != ArtLehrkraft {
		t.Errorf("Lehrkraft falsch eingeordnet: %+v", leser[2])
	}

	// der Punkt: Lehrkraefte duerfen nicht in der Schuelermenge landen.
	schueler := nurArt(leser, ArtSchueler)
	if len(schueler) != 2 {
		t.Fatalf("len(schueler) = %d, erwartet 2 Schueler", len(schueler))
	}
	for _, s := range schueler {
		if s.Nachname == "Lehrkraft" {
			t.Error("eine Lehrkraft ist in der Schuelermenge gelandet")
		}
		if s.Nachname == "Fachbereich" {
			t.Error("ein Fachbereichs-Sammelkonto ist in der Schuelermenge gelandet")
		}
	}

	if len(nurArt(leser, ArtLehrkraft)) != 1 {
		t.Error("Lehrkraft-Menge falsch")
	}
	if len(nurArt(leser, ArtAbgegangen)) != 1 {
		t.Error("Abgaenger-Menge falsch")
	}
	// Die Gruppe reist mit: Das Protokoll nennt sie fuer ein Kollegium-Konto, und der Halt
	// bei fehlender Zuordnung nennt Schluessel und Bezeichnung.
	if leser[4].Gruppe != "Fachbereich Erdkunde" || leser[4].GruppeNr != "97" {
		t.Errorf("Gruppe des Fachbereichskontos: %q / %q", leser[4].Gruppe, leser[4].GruppeNr)
	}
}

// TestUnbekannteArtWirdNichtZuSchueler haelt die vorsichtige Vorgabe fest: Was nicht
// eindeutig zugeordnet werden kann, wird nicht stillschweigend zum Schueler gemacht —
// der Personenlauf haelt an, bis es zugeordnet ist (OhneZuordnung).
func TestUnbekannteArtWirdNichtZuSchueler(t *testing.T) {
	gruppen := map[string]lesergruppe{"1": {Klasse: "UNDEF", Bezeichnung: "Undefinierte Untergruppe", Art: ArtUnbekannt}}
	const csv = `Buchungsnummer,Lesernummer,Vorname,Nachname,Lesergruppe
50,2001,"Unklar","Fall",1
`
	leser, err := leseLeser(strings.NewReader(csv), gruppen)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if len(nurArt(leser, ArtSchueler)) != 0 {
		t.Error("unklare Zuordnung darf nicht als Schueler durchgehen")
	}
	if leser[0].Art != ArtUnbekannt {
		t.Errorf("Art muss ArtUnbekannt bleiben, war %d", leser[0].Art)
	}
}

// TestLeserOhneGruppe: Ein Leser, dessen Lesergruppe nicht auflösbar ist, bekommt
// weder Klasse noch Art — und faellt damit aus jeder Schreibmenge heraus.
func TestLeserOhneGruppe(t *testing.T) {
	const csv = `Buchungsnummer,Lesernummer,Vorname,Nachname,Lesergruppe
51,3001,"Ohne","Gruppe",9999
`
	leser, err := leseLeser(strings.NewReader(csv), map[string]lesergruppe{})
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if leser[0].Art != ArtUnbekannt || leser[0].Klasse != "" {
		t.Errorf("nicht aufloesbare Gruppe: %+v", leser[0])
	}
}

// Aufbau der Datenbank der Schule: Die Untergruppe nennt die Klasse, die Art steht in der
// Obergruppe (Leser_OG). Bezeichnungen wie dort, „Undefinerte" so geschrieben.
const obergruppenCSV = `Buchungsnummer,KurzBez,Obergruppe
1,"UNDEF","Undefinierte Obergruppe"
3,"Fö","Förderstufe"
7,"Gym","Gymnasialzweig"
8,"SekII","Sekundarstufe II"
9,"Ab","Abgegangen"
74,"Lehrer","Lehrer"
84,"Neu","Grundschüler / Oberstufe"
`

const gruppenHeuteCSV = `Buchungsnummer,KurzBez,Untergruppe,Obergruppe
1,"UNDEF","Undefinerte Untergruppe",1
27,"05F1","05F1",3
60,"05G1","05G1",7
70,"ET1","11T1",8
218,"-/IKL1","-/IKL1",1
230,"ET8","ET8",1
240,"13T8","13T8",1
250,"NDHS","NDHS",1
95,"Ab","Abgegangen",9
90,"Lehrerin","Lehrerin",74
129,"U+","U-plus",74
300,"04Neu","04Neu",84
`

func leseGruppenHeute(t *testing.T, zuordnung Gruppenzuordnung) (map[string]lesergruppe, error) {
	t.Helper()
	obergruppen, err := leseObergruppen(strings.NewReader(obergruppenCSV))
	if err != nil {
		t.Fatalf("Obergruppen: %v", err)
	}
	return leseLesergruppen(strings.NewReader(gruppenHeuteCSV), obergruppen, zuordnung)
}

func TestArtUndKlasseImHeutigenAufbau(t *testing.T) {
	gruppen, err := leseGruppenHeute(t, nil)
	if err != nil {
		t.Fatalf("Gruppen: %v", err)
	}
	faelle := map[string]struct {
		art    LeserArt
		klasse string
		warum  string
	}{
		"27":  {ArtSchueler, "05F1", "Förderstufe, Klasse aus der Untergruppe"},
		"60":  {ArtSchueler, "05G1", "Gymnasialzweig"},
		"70":  {ArtSchueler, "11T1", "Einführungsphase: die Kurzbezeichnung ET1 nennt keinen Jahrgang"},
		"218": {ArtSchueler, "-/IKL1", "Inklusionsklasse ohne Obergruppe"},
		"230": {ArtSchueler, "ET8", "Einführungsphase ohne Obergruppe"},
		"240": {ArtSchueler, "13T8", "Klasse mit Jahrgang ohne Obergruppe"},
		"250": {ArtUnbekannt, "NDHS", "ein Name ohne Regel hält den Lauf an"},
		"1":   {ArtUnbekannt, "UNDEF", "undefiniert bleibt undefiniert"},
		"95":  {ArtAbgegangen, "Ab", "Abgegangene"},
		"90":  {ArtLehrkraft, "Lehrerin", "Lehrkraft"},
		"129": {ArtUPlus, "U+", "U-plus unter der Obergruppe Lehrer"},
		"300": {ArtSchueler, "04Neu", "Grundschüler über die Obergruppe"},
	}
	for id, erwartet := range faelle {
		g := gruppen[id]
		if g.Art != erwartet.art {
			t.Errorf("Gruppe %s (%s): Art %d, erwartet %d", id, erwartet.warum, g.Art, erwartet.art)
		}
		if g.Klasse != erwartet.klasse {
			t.Errorf("Gruppe %s (%s): Klasse %q, erwartet %q", id, erwartet.warum, g.Klasse, erwartet.klasse)
		}
	}
}

// Die Zuordnung des Laufs gilt vor jeder Regel, auch für eine Klasse, die eine Regel erkennt;
// ein Tippfehler im Gruppennamen und ein Ziel ohne Art halten an, statt nichts zu bewirken.
func TestGruppenzuordnungGiltVorJederRegel(t *testing.T) {
	gruppen, err := leseGruppenHeute(t, Gruppenzuordnung{
		"Undefinerte Untergruppe": "Abgegangen", "13T8": "Abgegangen"})
	if err != nil {
		t.Fatalf("Gruppen: %v", err)
	}
	if gruppen["1"].Art != ArtAbgegangen || gruppen["240"].Art != ArtAbgegangen {
		t.Errorf("Zuordnung nicht angewandt: %+v / %+v", gruppen["1"], gruppen["240"])
	}
	if _, err := leseGruppenHeute(t, Gruppenzuordnung{"Gibt es nicht": "Schüler"}); err == nil {
		t.Error("eine Zuordnung zu einer fehlenden Lesergruppe muss abbrechen")
	}
	if err := (Gruppenzuordnung{"NDHS": "Kinder"}).Pruefe(); err == nil {
		t.Error("ein Ziel, das keine Art benennt, muss abbrechen")
	}
	if err := (Gruppenzuordnung{"NDHS": "Schüler"}).Pruefe(); err != nil {
		t.Errorf("Schüler ist eine Art: %v", err)
	}
}

func TestSperrgrundAus(t *testing.T) {
	faelle := []struct{ am, begruendung, erwartet string }{
		{"03/12/24 10:15:00", " Ausweis  verloren\n", "In Littera gesperrt am 12.03.2024: Ausweis verloren"},
		{"03/12/24 10:15:00", "", "In Littera gesperrt am 12.03.2024"},
		// Die Begründung bleibt nach dem Aufheben stehen; ohne Sperrdatum ist niemand gesperrt.
		{"", "Hat 5 ausständige Hörbücher", ""},
	}
	for _, f := range faelle {
		if got := sperrgrundAus(f.am, f.begruendung); got != f.erwartet {
			t.Errorf("sperrgrundAus(%q, %q) = %q, erwartet %q", f.am, f.begruendung, got, f.erwartet)
		}
	}
}

// Die Spalten heißen in Littera GesperrtAm und SperrenBegründung; ein falscher Name ließe jede
// Sperre still fallen.
func TestLeseLeserLiestDieSperre(t *testing.T) {
	gruppen, err := leseLesergruppen(strings.NewReader(gruppenCSV), nil, nil)
	if err != nil {
		t.Fatalf("Gruppen: %v", err)
	}
	csv := "Buchungsnummer,Lesernummer,Vorname,Nachname,Lesergruppe,GesperrtAm,SperrenBegründung\n" +
		"1,1001,\"Anna\",\"Gesperrt\",22,\"03/12/24 10:15:00\",\"Ausweis verloren\"\n" +
		"2,1002,\"Bert\",\"Frei\",22,,\"alte Begründung\"\n"
	leser, err := leseLeser(strings.NewReader(csv), gruppen)
	if err != nil {
		t.Fatalf("Leser: %v", err)
	}
	if leser[0].Sperrgrund != "In Littera gesperrt am 12.03.2024: Ausweis verloren" || leser[1].Sperrgrund != "" {
		t.Errorf("Sperrgründe: %q / %q", leser[0].Sperrgrund, leser[1].Sperrgrund)
	}
}

// Ein Leser ohne Namen und ohne Ausleihe trägt nichts und kommt nicht mit; hat er eine Ausleihe,
// bleibt er, sonst gälte das Buch als verfügbar.
func TestLeereLeserOhneAusleiheFallenWeg(t *testing.T) {
	leser := []Leser{
		{ID: "1", Vorname: "Anna", Nachname: "Voll"},
		{ID: "2"},
		{ID: "3"},
		{ID: "4", Nachname: "Nurnachname"},
	}
	behalten, leer := ohneLeereLeser(leser, []Ausleihe{{LeserID: "3"}})
	if leer != 1 || len(behalten) != 3 {
		t.Fatalf("leer %d, behalten %d; erwartet 1 und 3", leer, len(behalten))
	}
	for _, l := range behalten {
		if l.ID == "2" {
			t.Error("der leere Leser ohne Ausleihe darf nicht mitkommen")
		}
	}
}
