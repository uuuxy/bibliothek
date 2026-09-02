// Package lmf bündelt das Wissen über Lernmittel (Lernmittelfreiheit, „LMF"):
// Schulbücher, die die Schule dem Schüler für ein Schuljahr leiht.
//
// Seit Migration 093 ist „Lernmittel" ein Feld am Titel (buecher_titel.ist_lernmittel).
// Die Regeln — Schuljahresfrist, Ausleihlimit, öffentlicher Katalog, Löschfrist,
// Bestellbedarf, Massenverlängerung — lesen diese Spalte und nichts sonst. Was
// dieses Paket noch mit dem TEXT zu tun hat, betrifft allein den Weg aus Littera:
//
//   - HatKennung erkennt Litteras Konvention „LMF …" in einem Feldwert. Der Import
//     setzt daraus das Feld; Migration 093 hat den Altbestand damit befüllt.
//   - Zerlege liest aus „LMF Bio 7" Fach und Jahrgang. Die Schulbücher tragen kein
//     Rückenetikett, ihre Littera-Signatur war nie eine Regaladresse, sondern nur
//     eine Einordnung — die eure Bibliothekarin schon vor Jahren von Hand getroffen
//     hat. Wir nutzen ihre Arbeit, statt am Titel zu raten.
//   - FachAusText, FachAusSchlagworten, JahrgangAusZielgruppe sind die Heuristiken
//     für alles, was keine LMF-Signatur hat. Vorher gab es zwei Stichwortlisten
//     (ISBN-Lookup: „Mathe", Excel-Import: „Mathematik"), die dieselbe Schule mit
//     zwei Fächern für dasselbe Fach zurückließen.
package lmf

import (
	"regexp"
	"strings"
)

// kennung matcht "lmf" am Anfang eines Feldwerts, gefolgt von einem Trenner
// (Leerzeichen oder Bindestrich). Deckt "LMF-Deutsch", "LMF - Deutsch" und
// "LMF Deutsch" ab, aber bewusst NICHT "LMFP-Roman" oder "lmfao" — nach dem Kürzel
// muss ein Trenner stehen. Der Bindestrich steht am Ende der Zeichenklasse, damit er
// literal (keine Range) ist. Migration 093 trägt dasselbe Muster als SQL; der Test
// dieses Pakets hält beide gegeneinander.
var kennung = regexp.MustCompile(`(?i)^lmf[ -]`)

// HatKennung meldet, ob ein Feldwert (Titel, Signatur) Litteras LMF-Konvention trägt.
func HatKennung(wert string) bool {
	return kennung.MatchString(strings.TrimSpace(wert))
}

// Fächer — die kanonischen Bezeichnungen, unter denen Import und Heuristiken ein Fach
// in der Systematik registrieren (buecher_titel.subject ist FK darauf, Migration 078).
// Eine Schreibweise je Fach: Vorher legte der ISBN-Lookup „Mathe" an und der
// Excel-Import „Mathematik", und die Systematik führte beides nebeneinander.
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

// fachKuerzel ordnet die Fachkürzel aus Littera-Signaturen („LMF Bio 7") und aus dem
// alten Signatur-Vorschlag der Maske („LMF M") den Fächern zu — kleingeschrieben,
// weil Littera „PoWi" und „Powi" nebeneinander kennt. Gezählt am Katalogisat vom
// Juni 2026: 474 LMF-Signaturen, 25 verschiedene Kürzel.
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

// Zerlegung ist, was eine Littera-Lernmittelsignatur über das Buch sagt.
type Zerlegung struct {
	// Fach ist die kanonische Bezeichnung; leer, wenn das Kürzel unbekannt ist
	// („LMF Pusch") — dann bleibt das Fach offen, statt geraten zu werden.
	Fach string
	// JahrgangVon/JahrgangBis: „7" ist 7–7, „12/13" und „1213" sind 12–13, ohne Zahl 0/0.
	// Nur 5–13 zählen: Littera schreibt bei Arbeitslehre „LMF Arb 1" — das ist Band 1,
	// kein erster Jahrgang, den es an der Schule gar nicht gibt.
	JahrgangVon, JahrgangBis int
}

var (
	kuerzelAmAnfang = regexp.MustCompile(`^[A-Za-zÄÖÜäöüß]+`)
	jahrgangsZahlen = regexp.MustCompile(`\d{1,2}`)
)

// Zerlege liest Fach und Jahrgang aus einer Signatur mit LMF-Kennung. ok ist false,
// wenn die Signatur keine Lernmittelsignatur ist — dann sagt sie nichts über Fach
// oder Jahrgang, und der Aufrufer lässt die Felder in Ruhe.
func Zerlege(signatur string) (z Zerlegung, ok bool) {
	rest := strings.TrimSpace(signatur)
	if !kennung.MatchString(rest) {
		return Zerlegung{}, false
	}
	rest = strings.TrimLeft(rest[3:], " -")

	if k := kuerzelAmAnfang.FindString(rest); k != "" {
		z.Fach = fachKuerzel[strings.ToLower(k)]
		if z.Fach == "" {
			// „LMF Deutsch 5": Klartext statt Kürzel — derselbe Stichwort-Weg wie beim Titel,
			// damit das Fach nicht davon abhängt, wie jemand die Signatur einst tippte.
			z.Fach = FachAusText(k)
		}
		rest = rest[len(k):]
	}
	// „LMF Deu 7 / Bie" (Access-Pfad): das Titelkürzel hinter dem Schrägstrich trägt
	// keine Jahrgangszahl, fällt aber auch nicht ins Gewicht — Zahlen stehen davor.
	if i := strings.Index(rest, "/"); i >= 0 && !strings.ContainsAny(rest[i:], "0123456789") {
		rest = rest[:i]
	}
	for _, zahl := range jahrgangsZahlen.FindAllString(rest, -1) {
		j := int(zahl[0] - '0')
		if len(zahl) == 2 {
			j = j*10 + int(zahl[1]-'0')
		}
		if j < 5 || j > 13 {
			continue
		}
		if z.JahrgangVon == 0 || j < z.JahrgangVon {
			z.JahrgangVon = j
		}
		if j > z.JahrgangBis {
			z.JahrgangBis = j
		}
	}
	return z, true
}

// stichwort ist ein Eintrag der Titel-Heuristik. Ein Slice, keine Map: Die Reihenfolge
// entscheidet bei Mehrfachtreffern, und eine Map-Iteration ist in Go absichtlich
// ungeordnet — die alte Lookup-Liste lieferte für „Mathematik und Physik" mal das
// eine, mal das andere.
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

// FachAusText rät das Fach aus freiem Text (Titel, Untertitel). Der erste Treffer in
// der Reihenfolge der Liste gewinnt; ohne Treffer "".
func FachAusText(text string) string {
	lower := strings.ToLower(text)
	for _, s := range stichwoerter {
		if strings.Contains(lower, s.wort) {
			return s.fach
		}
	}
	return ""
}

// schlagwortFach ordnet Littera-Schlagwörter (MAB 710) exakt einem Fach zu. Exakt,
// nicht enthalten: „Deutsch für Ausländer" ist kein Deutschunterricht, „Englische
// Literatur" kein Englischbuch.
var schlagwortFach = map[string]string{
	"mathematik": FachMathematik, "deutsch": FachDeutsch, "deutschunterricht": FachDeutsch,
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

// FachAusSchlagworten liefert das Fach, wenn die Schlagwörter GENAU EIN Fach nennen.
// Nennen sie zwei („Deutsch" und „Englisch" an einem zweisprachigen Band), bleibt das
// Fach offen — lieber leer als falsch.
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

// JahrgangAusZielgruppe übersetzt Litteras Zielgruppe (MAB 070b) in die Jahrgangs-
// spanne der kooperativen Gesamtschule mit Oberstufe. Ohne erkennbare Stufe 0/0.
func JahrgangAusZielgruppe(zielgruppe string) (von, bis int) {
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
