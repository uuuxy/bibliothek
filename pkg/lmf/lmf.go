// Package lmf kennt die Schreibweisen, an denen Littera ein Lernmittel erkennen lässt, und
// leitet aus Signatur, Titel, Schlagwörtern und Zielgruppe Fach und Jahrgang ab. Lernmittel
// sind die Schulbücher, die die Schule dem Schüler für ein Schuljahr leiht
// (Lernmittelfreiheit, „LMF").
//
// Ob ein Titel ein Lernmittel ist, sagt das Feld buecher_titel.ist_lernmittel, und die Regeln
// lesen nur dieses Feld. Mit dem Text befasst sich das Paket für den Weg aus Littera:
//
//   - HatKennung erkennt Litteras Schreibweise „LMF …" in einem Feldwert; der Import setzt
//     daraus das Feld.
//   - Zerlege liest aus „LMF Bio 7" Fach und Jahrgang. Die Signatur eines Schulbuchs war in
//     Littera keine Regaladresse, sondern eine Einordnung von Hand, und die übernimmt der
//     Import, statt am Titel zu raten.
//   - FachAusText, FachAusSchlagworten und JahrgangAusZielgruppen schätzen, wo keine solche
//     Signatur steht. Sie stehen an einer Stelle, damit jeder Weg des Imports dasselbe Fach
//     unter demselben Namen anlegt.
package lmf

import (
	"regexp"
	"strings"
)

// kennung trifft „lmf" am Anfang eines Feldwerts, gefolgt von Leerzeichen oder Bindestrich:
// „LMF-Deutsch", „LMF - Deutsch" und „LMF Deutsch", aber nicht „LMFP-Roman" oder „lmfao". Der
// Bindestrich steht am Ende der Zeichenklasse, damit er als Zeichen gilt. Migration 093 trägt
// dasselbe Muster als SQL; ein Test dieses Pakets hält beide gleich.
var kennung = regexp.MustCompile(`(?i)^lmf[ -]`)

// HatKennung meldet, ob ein Feldwert (Titel, Signatur) Litteras Schreibweise für Lernmittel
// trägt.
func HatKennung(wert string) bool {
	return kennung.MatchString(strings.TrimSpace(wert))
}

// VermerkMuster trifft „LMF" als eigenes Wort zwischen Leerzeichen oder Schrägstrichen:
// „LMF", „LMF/Bibliothek", „Buch LMF Ma 6". So schreibt die Schule den Vermerk in Litteras
// Standortangaben; „Filmfest" und „Elmf" treffen nicht.
const VermerkMuster = `(?i)(^|[\s/])LMF([\s/]|$)`

var vermerk = regexp.MustCompile(VermerkMuster)

// HatVermerk meldet, ob ein Standortvermerk den Bestand der Lernmittelfreiheit nennt.
func HatVermerk(wert string) bool {
	return vermerk.MatchString(wert)
}

// Die Fächer in der einen Schreibweise, unter der Import und Schätzung ein Fach in der
// Systematik anlegen (buecher_titel.subject verweist darauf). Legten zwei Wege „Mathe" und
// „Mathematik" an, führte die Systematik dasselbe Fach zweimal.
const (
	FachMathematik   = "Mathematik"
	FachDeutsch      = "Deutsch"
	FachEnglisch     = "Englisch"
	FachFranzoesisch = "Französisch"
	FachLatein       = "Latein"
	FachSpanisch     = "Spanisch"
	FachGeschichte   = "Geschichte"
	FachPoWi         = "Politik und Wirtschaft"
	FachErdkunde     = "Erdkunde"
	FachBiologie     = "Biologie"
	FachChemie       = "Chemie"
	FachPhysik       = "Physik"
	FachMusik        = "Musik"
	FachKunst        = "Kunst"
	FachReligion     = "Religion"
	FachEthik        = "Ethik"
	FachPhilosophie  = "Philosophie"
	FachInformatik   = "Informatik"
	FachSport        = "Sport"
	FachArbeitslehre = "Arbeitslehre"
	FachDarstSpiel   = "Darstellendes Spiel"
	FachNaWi         = "Naturwissenschaften"
)

// fachKuerzel ordnet die Kürzel aus Litteras Signaturen („LMF Bio 7") und aus dem früheren
// Signatur-Vorschlag der Maske („LMF M") den Fächern zu. Die Schlüssel sind klein geschrieben,
// weil Littera „PoWi" und „Powi" nebeneinander führt.
var fachKuerzel = map[string]string{
	"ma": FachMathematik, "m": FachMathematik,
	"deu": FachDeutsch, "d": FachDeutsch,
	"eng": FachEnglisch, "e": FachEnglisch,
	"fra": FachFranzoesisch, "f": FachFranzoesisch,
	"lat": FachLatein, "l": FachLatein,
	"spa": FachSpanisch,
	"ges": FachGeschichte, "g": FachGeschichte,
	"powi": FachPoWi, "powie": FachPoWi,
	"erd": FachErdkunde, "erdat": FachErdkunde, "ek": FachErdkunde,
	"bio": FachBiologie,
	"che": FachChemie, "ch": FachChemie,
	"phy": FachPhysik, "ph": FachPhysik,
	"mus": FachMusik,
	"ku":  FachKunst, "kun": FachKunst,
	"rel": FachReligion, "re": FachReligion,
	"eth":  FachEthik,
	"phil": FachPhilosophie,
	"inf":  FachInformatik, "info": FachInformatik,
	"spo": FachSport, "sposi": FachSport, "sposii": FachSport,
	"arb":  FachArbeitslehre,
	"dsp":  FachDarstSpiel,
	"nawi": FachNaWi,
}

// Zerlegung ist, was die Signatur eines Lernmittels aus Littera über den Titel sagt.
type Zerlegung struct {
	// Fach ist leer, wenn das Kürzel unbekannt ist („LMF Pusch"); es bleibt dann offen und
	// wird nicht geraten.
	Fach string
	// JahrgangVon und JahrgangBis: „7" ist 7 bis 7, „12/13" und „1213" sind 12 bis 13, ohne
	// Zahl 0 und 0. Nur 5 bis 13 zählen: In „LMF Arb 1" ist die 1 der Band, kein Jahrgang.
	JahrgangVon, JahrgangBis int
}

var (
	kuerzelAmAnfang = regexp.MustCompile(`^[A-Za-zÄÖÜäöüß]+`)
	jahrgangsZahlen = regexp.MustCompile(`\d{1,2}`)
)

// Zerlege liest Fach und Jahrgang aus einer Signatur mit LMF-Kennung. ok ist false, wenn die
// Signatur keine Kennung trägt; sie sagt dann nichts über Fach und Jahrgang, und der Rufer
// lässt die Felder unverändert.
func Zerlege(signatur string) (z Zerlegung, ok bool) {
	rest := strings.TrimSpace(signatur)
	if !kennung.MatchString(rest) {
		return Zerlegung{}, false
	}
	rest = strings.TrimLeft(rest[3:], " -")

	if k := kuerzelAmAnfang.FindString(rest); k != "" {
		z.Fach = fachKuerzel[strings.ToLower(k)]
		if z.Fach == "" {
			// „LMF Deutsch 5" nennt das Fach im Klartext. Es geht denselben Weg wie ein Titel,
			// damit das Fach nicht von der Schreibweise der Signatur abhängt.
			z.Fach = FachAusText(k)
		}
		rest = rest[len(k):]
	}
	// „LMF Deu 7 / Bie": Hinter dem Schrägstrich steht das Kürzel des Titels; es trägt keine
	// Zahl und fällt weg.
	if i := strings.Index(rest, "/"); i >= 0 && !strings.ContainsAny(rest[i:], "0123456789") {
		rest = rest[:i]
	}
	z.JahrgangVon, z.JahrgangBis = jahrgangsSpanne(rest)
	return z, true
}

// jahrgangsSpanne liest den kleinsten und den größten Jahrgang aus dem Rest der Signatur;
// ohne Zahl zwischen 5 und 13 bleibt es bei 0 und 0.
func jahrgangsSpanne(rest string) (von, bis int) {
	for _, zahl := range jahrgangsZahlen.FindAllString(rest, -1) {
		j := int(zahl[0] - '0')
		if len(zahl) == 2 {
			j = j*10 + int(zahl[1]-'0')
		}
		if j < 5 || j > 13 {
			continue
		}
		if von == 0 || j < von {
			von = j
		}
		if j > bis {
			bis = j
		}
	}
	return von, bis
}

// stichwort ist ein Eintrag der Liste, mit der FachAusText ein Fach schätzt. Die Liste ist ein
// Slice und keine Map, weil bei zwei Treffern die Reihenfolge entscheidet und Go eine Map in
// wechselnder Folge durchläuft.
type stichwort struct{ wort, fach string }

var stichwoerter = []stichwort{
	{"natur und technik", FachNaWi}, {"naturwissenschaft", FachNaWi},
	{"mathematik", FachMathematik}, {"mathe", FachMathematik}, {"algebra", FachMathematik}, {"geometrie", FachMathematik},
	{"deutsch", FachDeutsch},
	{"englisch", FachEnglisch}, {"english", FachEnglisch}, {"grammar", FachEnglisch},
	{"französisch", FachFranzoesisch}, {"franzoesisch", FachFranzoesisch}, {"franzosisch", FachFranzoesisch},
	{"français", FachFranzoesisch}, {"francais", FachFranzoesisch},
	{"latein", FachLatein}, {"spanisch", FachSpanisch},
	{"geschichte", FachGeschichte}, {"histor", FachGeschichte},
	{"biologie", FachBiologie}, {"chemie", FachChemie}, {"physik", FachPhysik},
	{"erdkunde", FachErdkunde}, {"geographie", FachErdkunde}, {"geografie", FachErdkunde},
	{"politik", FachPoWi}, {"powi", FachPoWi}, {"sozialkunde", FachPoWi},
	{"informatik", FachInformatik}, {"musik", FachMusik}, {"kunst", FachKunst},
	{"religion", FachReligion}, {"ethik", FachEthik}, {"philosophie", FachPhilosophie},
	{"arbeitslehre", FachArbeitslehre}, {"sport", FachSport},
}

// FachAusText schätzt das Fach aus freiem Text (Titel, Untertitel). Der erste Treffer in der
// Reihenfolge der Liste gewinnt; ohne Treffer "".
func FachAusText(text string) string {
	klein := strings.ToLower(text)
	for _, s := range stichwoerter {
		if strings.Contains(klein, s.wort) {
			return s.fach
		}
	}
	return ""
}

// schlagwortFach ordnet Litteras Schlagwörter (MAB 710) einem Fach zu, wenn sie ihm genau
// gleichen: „Deutsch für Ausländer" ist kein Deutschunterricht, „Englische Literatur" kein
// Englischbuch.
var schlagwortFach = map[string]string{
	"mathematik": FachMathematik, "mathe": FachMathematik, "deutsch": FachDeutsch, "deutschunterricht": FachDeutsch,
	"englisch": FachEnglisch, "englischunterricht": FachEnglisch,
	"französisch": FachFranzoesisch, "französischunterricht": FachFranzoesisch,
	"latein": FachLatein, "lateinunterricht": FachLatein, "spanisch": FachSpanisch,
	"geschichte": FachGeschichte, "geschichtsunterricht": FachGeschichte,
	"biologie": FachBiologie, "biologieunterricht": FachBiologie,
	"chemie": FachChemie, "chemieunterricht": FachChemie,
	"physik": FachPhysik, "physikunterricht": FachPhysik,
	"erdkunde": FachErdkunde, "geographie": FachErdkunde, "geografie": FachErdkunde,
	"politik": FachPoWi, "politik und wirtschaft": FachPoWi, "sozialkunde": FachPoWi,
	"informatik": FachInformatik, "musik": FachMusik, "musikunterricht": FachMusik,
	"kunst": FachKunst, "kunstunterricht": FachKunst,
	"religion": FachReligion, "religionsunterricht": FachReligion,
	"ethik": FachEthik, "philosophie": FachPhilosophie, "sport": FachSport,
	"arbeitslehre": FachArbeitslehre, "darstellendes spiel": FachDarstSpiel,
}

// FachExakt liefert das Fach, wenn der Text als Ganzes ein Fach benennt („Mathe", „Deutsch",
// „Politik und Wirtschaft"), sonst "". Sie ist für Spalten gedacht, die ein Fach tragen können,
// aber nicht müssen: Litteras Kategorie enthält auch Standorttexte wie „Buch Che 11/Che 175",
// und die Suche nach einem enthaltenen Wort (FachAusText) machte daraus ein Chemiebuch.
func FachExakt(text string) string {
	klein := strings.ToLower(strings.TrimSpace(text))
	if klein == "" {
		return ""
	}
	if f, ok := schlagwortFach[klein]; ok {
		return f
	}
	for _, f := range alleFaecher {
		if strings.ToLower(f) == klein {
			return f
		}
	}
	return ""
}

var alleFaecher = []string{
	FachMathematik, FachDeutsch, FachEnglisch, FachFranzoesisch, FachLatein, FachSpanisch,
	FachGeschichte, FachPoWi, FachErdkunde, FachBiologie, FachChemie, FachPhysik, FachMusik,
	FachKunst, FachReligion, FachEthik, FachPhilosophie, FachInformatik, FachSport,
	FachArbeitslehre, FachDarstSpiel, FachNaWi,
}

// FachAusSchlagworten liefert das Fach, wenn die Schlagwörter genau ein Fach nennen. Nennen
// sie zwei („Deutsch" und „Englisch" an einem zweisprachigen Band), bleibt das Fach offen.
func FachAusSchlagworten(schlagwoerter []string) string {
	fach := ""
	for _, sw := range schlagwoerter {
		f, bekannt := schlagwortFach[strings.ToLower(strings.TrimSpace(sw))]
		if !bekannt {
			continue
		}
		if fach != "" && fach != f {
			return ""
		}
		fach = f
	}
	return fach
}

// jahrgangAusZielgruppe übersetzt eine Zielgruppe aus Littera (MAB 070b) in die Jahrgänge der
// kooperativen Gesamtschule mit Oberstufe. Ohne erkennbare Stufe 0 und 0.
func jahrgangAusZielgruppe(zielgruppe string) (von, bis int) {
	z := strings.ToLower(strings.TrimSpace(zielgruppe))
	switch {
	case z == "":
		return 0, 0
	case strings.Contains(z, "1 u. 2"), strings.Contains(z, "1 u 2"), strings.Contains(z, "1 und 2"),
		strings.Contains(z, "i u. ii"), strings.Contains(z, "i und ii"):
		return 5, 13
	case strings.Contains(z, "sekundarstufe 2"), strings.Contains(z, "sek ii"), strings.Contains(z, "sekundarstufe ii"),
		strings.Contains(z, "oberstufe"):
		return 11, 13
	case strings.Contains(z, "sekundarstufe 1"), strings.Contains(z, "sek i"), strings.Contains(z, "sekundarstufe i"):
		return 5, 10
	case strings.Contains(z, "eingangsstufe"), strings.Contains(z, "förderstufe"):
		return 5, 6
	}
	return 0, 0
}

// JahrgangAusZielgruppen liefert die Jahrgänge aus allen Zielgruppen eines Titels; Littera
// führt bis zu fünf, etwa „Sekundarstufe 2" und „Lehrer". Die Spanne reicht vom kleinsten bis
// zum größten erkannten Jahrgang: „Sekundarstufe 1" und „Sekundarstufe 2" ergeben 5 bis 13.
// Eine Gruppe ohne Stufe („Lehrer", „DAZ-Schüler") trägt nichts bei.
func JahrgangAusZielgruppen(zielgruppen []string) (von, bis int) {
	for _, z := range zielgruppen {
		v, b := jahrgangAusZielgruppe(z)
		if v == 0 {
			continue
		}
		if von == 0 || v < von {
			von = v
		}
		if b > bis {
			bis = b
		}
	}
	return von, bis
}
