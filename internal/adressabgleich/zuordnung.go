package adressabgleich

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Konto ist ein Zugangskonto, wie die Zuordnung es braucht.
type Konto struct {
	ID       string
	Vorname  string
	Nachname string
	Email    string
	Aktiv    bool
	// LDAPKennung ist der Anmeldename im PaedNet; steht er als Kürzel im Adressbuch, gilt der
	// Treffer über das Kürzel vor dem über den Namen.
	LDAPKennung string
}

// Uebernahme ist eine Adresse, die ein Konto bekommt, und der Weg, auf dem sie gefunden wurde.
type Uebernahme struct {
	KontoID     string `json:"konto_id"`
	Name        string `json:"name"`
	Adresse     string `json:"adresse"`
	Weg         string `json:"weg"`
	Platzhalter string `json:"-"`
}

// Uebergangen ist ein Konto mit Platzhalter, das keine Adresse bekommt, und der Grund dafür.
type Uebergangen struct {
	KontoID string `json:"konto_id"`
	Name    string `json:"name"`
	Grund   string `json:"grund"`
}

// Zuordnung ist das Ergebnis von BerechneZuordnung.
type Zuordnung struct {
	Uebernahmen []Uebernahme
	Uebergangen []Uebergangen
	// KontenMitPlatzhalter zählt die aktiven Konten, die eine Adresse bekommen könnten.
	KontenMitPlatzhalter int
}

// Gründe, aus denen ein Konto keine Adresse bekommt.
const (
	grundNichtImAdressbuch    = "nicht im globalen Adressbuch"
	grundNameMehrfach         = "Name im Adressbuch mehrfach"
	grundKuerzelMehrfach      = "Kürzel im Adressbuch mehrfach"
	grundMehrereSchuladressen = "Eintrag mit mehreren Adressen der Schule"
	grundMehrereKonten        = "mehrere Konten mit diesem Namen"
	grundAdresseVergeben      = "Adresse gehört schon einem anderen Konto"
	grundAdresseDoppelt       = "Adresse steht im Adressbuch bei mehreren Einträgen"
	mehrdeutigeAdresse        = "\x00mehrdeutig"
)

// Die Wege zu einem Treffer, vom strengsten zum losesten. Die Stufen des Namens zählen der Reihe
// nach; die erste, die überhaupt einen Eintrag findet, entscheidet.
const (
	wegKuerzel    = "Kürzel"
	stufeGenau    = 0 // Vor- und Nachname gleich
	stufeTolerant = 1 // gleich bis auf Umlaute, Akzente, Bindestriche, Titel und weitere Vornamen
	stufeTeil     = 2 // erster Vorname gleich und ein Teil eines Doppelnamens gleich
	anzahlStufen  = 3
)

var wegJeStufe = [anzahlStufen]string{"Name", "Name in anderer Schreibung", "Teil des Nachnamens"}

// BerechneZuordnung bestimmt, welches Konto welche Adresse bekommt. Die Regeln:
//   - Nur aktive Konten mit einer Platzhalter-Adresse (platzhalterDomain, etwa „@littera.invalid")
//     bekommen eine Adresse; eine echte Adresse ändert der Abgleich nie, denn bei der Anmeldung über
//     den Mailserver ist sie der Anmeldename.
//   - Nur Adressen der Schule (schulDomain) zählen; private Adressen im Adressbuch nicht.
//   - Zuerst zählt das Kürzel: Trägt das Konto eine LDAP-Kennung und steht sie als Kürzel im
//     Adressbuch, ist das der Treffer. Sonst der Name in drei Stufen (stufeGenau bis stufeTeil).
//   - Nur eindeutige Treffer in beide Richtungen: genau ein Eintrag passt zum Konto, und genau ein
//     Konto passt zu diesem Eintrag. Die Adresse trägt noch kein anderes Konto und steht im
//     Adressbuch nur bei diesem Eintrag.
func BerechneZuordnung(eintraege []Eintrag, konten []Konto, schulDomain, platzhalterDomain string) Zuordnung {
	schulDomain = "@" + strings.TrimPrefix(strings.ToLower(strings.TrimSpace(schulDomain)), "@")
	platzhalterDomain = strings.ToLower(platzhalterDomain)

	buch := leseBuch(eintraege, schulDomain)
	kontoNamen := make([]namen, len(konten))
	vergeben := map[string]bool{}
	for i, k := range konten {
		kontoNamen[i] = namenAus(k.Nachname, k.Vorname)
		if !istPlatzhalter(k.Email, platzhalterDomain) {
			vergeben[strings.ToLower(strings.TrimSpace(k.Email))] = true
		}
	}

	var z Zuordnung
	for _, i := range reihenfolgeNachName(konten) {
		k := konten[i]
		if !k.Aktiv || !istPlatzhalter(k.Email, platzhalterDomain) {
			continue
		}
		z.KontenMitPlatzhalter++
		name := strings.TrimSpace(k.Vorname + " " + k.Nachname)
		adresse, weg, grund := buch.finde(k.LDAPKennung, kontoNamen[i], kontoNamen, vergeben)
		if grund != "" {
			z.Uebergangen = append(z.Uebergangen, Uebergangen{KontoID: k.ID, Name: name, Grund: grund})
			continue
		}
		vergeben[adresse] = true
		z.Uebernahmen = append(z.Uebernahmen, Uebernahme{KontoID: k.ID, Name: name, Adresse: adresse, Weg: weg, Platzhalter: k.Email})
	}
	return z
}

// buchEintrag ist ein Eintrag mit Adresse der Schule, vorbereitet für den Vergleich.
type buchEintrag struct {
	namen   namen
	kuerzel string
	adresse string // die Adresse der Schule oder mehrdeutigeAdresse
}

type adressbuch struct {
	eintraege          []buchEintrag
	eintraegeJeAdresse map[string]int
}

func leseBuch(eintraege []Eintrag, schulDomain string) adressbuch {
	b := adressbuch{eintraegeJeAdresse: map[string]int{}}
	for _, e := range eintraege {
		adresse, ok := schuladresse(e.Adressen, schulDomain)
		if !ok {
			continue
		}
		b.eintraege = append(b.eintraege, buchEintrag{namen: namenAus(e.Nachname, e.Vorname), kuerzel: kuerzelSchluessel(e.Kuerzel), adresse: adresse})
		b.eintraegeJeAdresse[adresse]++
	}
	return b
}

// finde sucht den Eintrag zu einem Konto: erst über das Kürzel, dann über den Namen. Es liefert
// Adresse und Weg oder den Grund, aus dem es keine gibt.
func (b adressbuch) finde(kennung string, konto namen, alleKonten []namen, vergeben map[string]bool) (adresse, weg, grund string) {
	if adresse, weg, grund, gefunden := b.findeUeberKuerzel(kennung, vergeben); gefunden {
		return adresse, weg, grund
	}
	if !konto.vollstaendig() {
		return "", "", grundNichtImAdressbuch
	}
	for stufe := range anzahlStufen {
		treffer := b.passendeEintraege(stufe, konto)
		switch {
		case len(treffer) == 0:
			continue
		case len(treffer) > 1:
			return "", "", grundNameMehrfach
		}
		adresse, weg, grund = b.pruefeAdresse(treffer[0].adresse, wegJeStufe[stufe], vergeben)
		if grund == "" && zaehlePassendeKonten(stufe, treffer[0].namen, alleKonten) > 1 {
			return "", "", grundMehrereKonten
		}
		return adresse, weg, grund
	}
	return "", "", grundNichtImAdressbuch
}

// findeUeberKuerzel sucht die LDAP-Kennung unter den Kürzeln; gefunden ist false, wenn sie dort
// nicht steht und der Name entscheiden soll. Mehrere Konten mit derselben Kennung gibt es nicht:
// Die LDAP-Kennung ist eindeutig.
func (b adressbuch) findeUeberKuerzel(kennung string, vergeben map[string]bool) (adresse, weg, grund string, gefunden bool) {
	k := kuerzelSchluessel(kennung)
	if k == "" {
		return "", "", "", false
	}
	var treffer []buchEintrag
	for _, e := range b.eintraege {
		if e.kuerzel == k {
			treffer = append(treffer, e)
		}
	}
	switch len(treffer) {
	case 0:
		return "", "", "", false
	case 1:
		adresse, weg, grund = b.pruefeAdresse(treffer[0].adresse, wegKuerzel, vergeben)
		return adresse, weg, grund, true
	}
	return "", "", grundKuerzelMehrfach, true
}

// passendeEintraege liefert die Einträge mit Namen, die auf dieser Stufe zum Konto passen.
func (b adressbuch) passendeEintraege(stufe int, konto namen) []buchEintrag {
	var treffer []buchEintrag
	for _, e := range b.eintraege {
		if e.namen.vollstaendig() && passt(stufe, konto, e.namen) {
			treffer = append(treffer, e)
		}
	}
	return treffer
}

func (b adressbuch) pruefeAdresse(adresse, weg string, vergeben map[string]bool) (string, string, string) {
	switch {
	case adresse == mehrdeutigeAdresse:
		return "", "", grundMehrereSchuladressen
	case b.eintraegeJeAdresse[adresse] > 1:
		return "", "", grundAdresseDoppelt
	case vergeben[adresse]:
		return "", "", grundAdresseVergeben
	}
	return adresse, weg, ""
}

// zaehlePassendeKonten zählt die Konten, die auf dieser Stufe zum Eintrag passen. Eine lose Stufe
// zählt die strengeren mit: Passt ein anderes Konto genauer, gehört der Eintrag nicht sicher dem
// gesuchten.
func zaehlePassendeKonten(stufe int, eintrag namen, konten []namen) int {
	n := 0
	for _, k := range konten {
		if k.vollstaendig() && passt(stufe, k, eintrag) {
			n++
		}
	}
	return n
}

// namen sind die vergleichbaren Formen eines Namens.
type namen struct {
	genau    string   // klein, Leerraum zusammengezogen, Unicode in einer Form
	nachname string   // tolerant und ohne Leerraum: „Janicka-Ahlemeyer" wie „Janicka Ahlemeyer"
	vorname  string   // erster Vorname, tolerant
	teile    []string // Teile des Nachnamens, tolerant, ohne Namenszusätze wie „von"
}

func namenAus(nachname, vorname string) namen {
	toleranterNachname := tolerant(nachname)
	vornamen := strings.Fields(tolerant(vorname))
	n := namen{genau: glaette(nachname) + "\x1f" + glaette(vorname), nachname: strings.ReplaceAll(toleranterNachname, " ", "")}
	if len(vornamen) > 0 {
		n.vorname = vornamen[0]
	}
	for _, teil := range strings.Fields(toleranterNachname) {
		if !istNamenszusatz(teil) {
			n.teile = append(n.teile, teil)
		}
	}
	return n
}

func (n namen) vollstaendig() bool { return n.nachname != "" && n.vorname != "" }

func passt(stufe int, konto, eintrag namen) bool {
	switch stufe {
	case stufeGenau:
		return konto.genau == eintrag.genau
	case stufeTolerant:
		return konto.vorname == eintrag.vorname && konto.nachname == eintrag.nachname
	case stufeTeil:
		return konto.vorname == eintrag.vorname && teilenEinenTeil(konto.teile, eintrag.teile)
	}
	return false
}

func teilenEinenTeil(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

func glaette(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(norm.NFC.String(s))), " ")
}

var umlaute = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "-", " ", ".", " ", ",", " ", "'", "", "’", "")

// tolerant macht einen Namen unabhängig von Schreibweisen: Umlaute ausgeschrieben, Akzente weg,
// Bindestriche und Punkte als Leerraum, Titel wie „Dr." vorn entfernt.
func tolerant(s string) string {
	s = umlaute.Replace(strings.ToLower(norm.NFC.String(s)))
	var b strings.Builder
	for _, z := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, z) {
			b.WriteRune(z)
		}
	}
	felder := strings.Fields(b.String())
	for len(felder) > 0 && istTitel(felder[0]) {
		felder = felder[1:]
	}
	return strings.Join(felder, " ")
}

func istTitel(wort string) bool {
	switch wort {
	case "dr", "prof", "med", "phil", "rer", "nat", "dipl", "ing", "hc":
		return true
	}
	return false
}

func istNamenszusatz(wort string) bool {
	switch wort {
	case "von", "van", "vom", "zu", "zur", "de", "del", "der", "den", "di", "da", "du", "la", "le", "ten", "ter":
		return true
	}
	return false
}

func kuerzelSchluessel(kuerzel string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(kuerzel)))
}

// schuladresse liefert die Adresse der Schule eines Eintrags: mehrdeutigeAdresse, wenn er zwei
// verschiedene trägt, und false, wenn er keine trägt.
func schuladresse(adressen []string, schulDomain string) (string, bool) {
	eigene := map[string]bool{}
	for _, a := range adressen {
		a = strings.ToLower(strings.TrimSpace(a))
		if strings.HasSuffix(a, schulDomain) && len(a) > len(schulDomain) {
			eigene[a] = true
		}
	}
	switch len(eigene) {
	case 0:
		return "", false
	case 1:
		for a := range eigene {
			return a, true
		}
	}
	return mehrdeutigeAdresse, true
}

func istPlatzhalter(email, platzhalterDomain string) bool {
	return platzhalterDomain != "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), platzhalterDomain)
}

// reihenfolgeNachName liefert die Indizes der Konten nach Namen sortiert, damit Liste und Mail in
// derselben Ordnung stehen.
func reihenfolgeNachName(konten []Konto) []int {
	reihe := make([]int, len(konten))
	for i := range reihe {
		reihe[i] = i
	}
	sort.SliceStable(reihe, func(a, b int) bool {
		return glaette(konten[reihe[a]].Nachname+" "+konten[reihe[a]].Vorname) < glaette(konten[reihe[b]].Nachname+" "+konten[reihe[b]].Vorname)
	})
	return reihe
}
