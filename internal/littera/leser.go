package littera

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// LeserArt sagt, wohin ein Littera-Leser gehört.
//
// Littera führt alle Personen in einer Tabelle `Leser` — Schüler, Lehrkräfte,
// Praktikanten, das Sekretariat und sogar Sammelkonten der Fachbereiche. Diese
// Anwendung trennt sie: Schüler nach `schueler`, alle anderen ins Kollegium, jeweils mit
// ihrer Art an der Leserzeile (leser.art, zielArt). Ohne diese Einordnung landeten
// 158 Lehrkräfte in der Schülerdatei.
//
// Die Unterscheidung steht in den Daten selbst und muss nicht geraten werden:
// `Leser.Lesergruppe` → `Leser_UG.Untergruppe`.
type LeserArt int

const (
	// ArtUnbekannt steht für eine Untergruppe, die keiner Art zugeordnet ist („Undefinierte
	// Untergruppe", „IMPORT"), und für eine Lesergruppe, die in Leser_UG fehlt. Mit solchen
	// Lesern hält der Personenlauf an, bevor er etwas schreibt (OhneZuordnung): Ausgelassen
	// fehlten ihre Ausleihen, und die Bücher stünden als verfügbar im Regal. Zugeordnet wird
	// in Littera selbst oder hier in artAusUntergruppe.
	ArtUnbekannt LeserArt = iota
	// ArtSchueler umfasst „Schüler" und „Sekundarstufe II": Die Oberstufenklassen
	// (11T1, 12T3, 13T5) sind eine eigene Untergruppe, aber selbstverständlich Schüler.
	// Ebenso „Im Ausland": Sie kommen mit der Littera-Klasse („AUS"), die richtige trägt der
	// LUSD-Import nach; das Abgangsjahr ist der Rückfall für eine Klasse ohne Jahrgang
	// (personenlauf.abgangsjahr).
	ArtSchueler
	// ArtLehrkraft umfasst „Lehrer" und „Lehrerin".
	ArtLehrkraft
	// ArtAbgegangen sind ehemalige Schüler — Untergruppe „Abgegangen". Gehören als
	// Abgänger in die Schülerdatei, nicht als aktive Schüler.
	ArtAbgegangen
	// ArtPraktikum sind Praktikanten: keine Schüler, aber Leser mit Ausleihen. Sie kommen ins
	// Kollegium, werden also nicht gemahnt, und bekommen kein Konto (leserart.HatKonto).
	ArtPraktikum
	// ArtLiV sind Referendare, in Hessen LiV (Lehrkraft im Vorbereitungsdienst). Sie gehören ins
	// Kollegium mit der Art „liv".
	ArtLiV
	// ArtSekretariat ist das Sekretariat: Kollegium mit Konto, wie eine Lehrkraft.
	ArtSekretariat
	// ArtUPlus sind die Vertretungskräfte aus „U-plus": Kollegium mit Konto, wie eine
	// Lehrkraft.
	ArtUPlus
	// ArtFachbereich sind die Sammelkonten der Fachbereiche („Fachbereich Erdkunde"): keine
	// Person, sondern ein Konto, das die Kollegen des Fachs benutzen. Kollegium ohne Konto.
	ArtFachbereich
)

// zielArt ist die Art der Leserzeile, in der ein Littera-Leser landet (chk_leser_art); "" für
// einen Leser ohne Zuordnung, den der Lauf nicht schreibt.
func (a LeserArt) zielArt() string {
	switch a {
	case ArtSchueler, ArtAbgegangen:
		return "schueler"
	case ArtLehrkraft:
		return "lehrkraft"
	case ArtLiV:
		return "liv"
	case ArtPraktikum:
		return "praktikum"
	case ArtSekretariat:
		return "sekretariat"
	case ArtUPlus:
		return "uplus"
	case ArtFachbereich:
		return "fachbereich"
	}
	return ""
}

// lesergruppe ist eine Zeile aus `Leser_UG` — Klassenbezeichnung plus Art.
type lesergruppe struct {
	Klasse      string // die Klasse, wie sie an der Schule heißt: "07H1", "12T3" (klasseDerGruppe)
	Bezeichnung string // Untergruppe, wie Littera sie führt: „Schüler", „05F1", „Fachbereich Erdkunde"
	Art         LeserArt
}

// Gruppenzuordnung ordnet eine Littera-Untergruppe einer Art zu, benannt wie in Littera
// („Undefinierte Untergruppe" → „Schüler"). Sie stellt nach, was die Bücherei sonst in Littera
// zuordnen müsste, und gilt vor jeder Regel (artDerGruppe).
type Gruppenzuordnung map[string]string

// Pruefe meldet ein Ziel, das keine Art benennt; die Zuordnung soll den Lauf nicht still unter
// „ohne Zuordnung" stehen lassen.
func (z Gruppenzuordnung) Pruefe() error {
	for gruppe, ziel := range z {
		if artAusUntergruppe(ziel) == ArtUnbekannt {
			return fmt.Errorf("zuordnung „%s=%s“: „%s“ ist keine Art; Arten wie in Littera sind Schüler, "+
				"Sekundarstufe II, Abgegangen, Lehrer, Referendar, Praktikant, Sekretärin, U-plus und Fachbereich",
				gruppe, ziel, ziel)
		}
	}
	return nil
}

// Leser ist eine Person aus dem Littera-Altbestand.
//
// Zwei Nummern mit verschiedenen Aufgaben:
//
//   - ID = `Buchungsnummer` ist der interne Schlüssel. Darauf zeigt `Verleih.Leser` —
//     gemessen: 15.615 von 15.615 Ausleihen treffen darüber, über `Lesernummer` nur 792.
//   - Lesernummer ist die Ausweisnummer, die der Schüler in der Hand hält. Sie gehört
//     nach `schueler.barcode_id`, taugt aber nicht zum Verknüpfen der Ausleihen.
//
// Wer die beiden verwechselt, hängt die Ausleihen an die falschen Personen — und das
// fällt erst auf, wenn ein Schüler die Bücher eines anderen zurückbringen soll.
type Leser struct {
	ID           string // Buchungsnummer — Ziel des Fremdschlüssels in Verleih
	Lesernummer  string // Ausweisnummer → schueler.barcode_id
	Vorname      string
	Nachname     string
	Klasse       string
	Art          LeserArt
	Gruppe       string // Littera-Untergruppe (Lesergruppe.Bezeichnung), leer ohne Eintrag in Leser_UG
	GruppeNr     string // Leser.Lesergruppe — der Schlüssel in Leser_UG
	Geburtsdatum string // Rohform wie exportiert
	EMail        string
	Strasse      string
	PLZ          string
	Ort          string
	Sperrgrund   string // leer, wenn Littera den Leser nicht sperrt (sperrgrundAus)
}

// sperrgrundAus ist der Grund der Sperre, wie das Programm ihn an der Akte zeigt; leer, wenn
// Littera den Leser nicht sperrt. Gesperrt ist, wer ein Sperrdatum trägt: Das Häkchen „Leser
// sperren" setzt es samt Benutzer (GesperrtAm, GesperrtVon), Aufheben nimmt beide weg. Die
// Begründung bleibt nach dem Aufheben stehen und zählt deshalb allein nicht.
func sperrgrundAus(gesperrtAm, begruendung string) string {
	if strings.TrimSpace(gesperrtAm) == "" {
		return ""
	}
	grund := "In Littera gesperrt"
	if am, ok := datumAus(gesperrtAm); ok {
		grund += " am " + am.Format("02.01.2006")
	}
	if b := strings.Join(strings.Fields(begruendung), " "); b != "" {
		grund += ": " + b
	}
	return grund
}

// artAusUntergruppe ordnet die Untergruppen-Bezeichnung einer Art zu.
//
// Bewusst über die Bezeichnung und nicht über die Nummer: Die Nummern sind
// installationsspezifisch vergeben (Fachbereiche liegen zwischen den Klassen), die
// Bezeichnungen sind sprechend und über Littera-Installationen hinweg stabil.
func artAusUntergruppe(bezeichnung string) LeserArt {
	b := strings.TrimSpace(bezeichnung)
	switch b {
	case "Schüler", "Sekundarstufe II", "Im Ausland":
		return ArtSchueler
	case "Lehrer", "Lehrerin":
		return ArtLehrkraft
	case "Referendar", "Referendarin":
		return ArtLiV
	case "Abgegangen":
		return ArtAbgegangen
	case "Praktikant", "Praktikantin":
		return ArtPraktikum
	case "Sekretärin":
		return ArtSekretariat
	case "U-plus":
		return ArtUPlus
	}
	// Sammelkonten der Fachbereiche sind keine Personen.
	if strings.HasPrefix(b, "Fachbereich") {
		return ArtFachbereich
	}
	return ArtUnbekannt
}

// artAusObergruppe ordnet die Obergruppe einer Art zu. Seit Littera die Klasse in der Untergruppe
// führt („05F1"), steht die Art nur noch hier („Gymnasialzweig", „Lehrer"); die Bezeichnungen
// sind dieselben wie in der Sicherung von 2010, nur „Hauptschulzweig" heißt heute „Hauptschule".
func artAusObergruppe(bezeichnung string) LeserArt {
	b := strings.TrimSpace(bezeichnung)
	// „Grundschüler / Oberstufe“ über den Anfang, wie Littera sie führt.
	if strings.HasPrefix(b, "Grundschüler") {
		return ArtSchueler
	}
	switch b {
	case "Förderstufe", "Hauptschule", "Hauptschulzweig", "Realschulzweig", "Gymnasialzweig",
		"Sekundarstufe II", "Im Ausland":
		return ArtSchueler
	case "Abgegangen":
		return ArtAbgegangen
	case "Lehrer":
		return ArtLehrkraft
	case "Referendar":
		return ArtLiV
	case "Praktikant":
		return ArtPraktikum
	case "Fachbereich":
		return ArtFachbereich
	case "Sekretärin":
		return ArtSekretariat
	}
	return ArtUnbekannt
}

// einfuehrungsphase und inklusionsklasse erkennen Klassen ohne Jahrgang im Namen: „ET8" ist eine
// Klasse der Einführungsphase (Jahrgang 11), „IKL1" eine Inklusionsklasse; beide sind Schüler.
var (
	einfuehrungsphase = regexp.MustCompile(`(?i)^ET(\d|X|\s|WEG)`)
	inklusionsklasse  = regexp.MustCompile(`(?i)IKL`)
)

// istKlassenname erkennt eine Klasse am Namen der Untergruppe: Jahrgang und Zweig („05G6",
// „13T8"), die Einführungsphase („ET8") und eine Inklusionsklasse („IKL1", „-/IKL1"). So kommt
// eine Klasse an, deren Obergruppe in Littera leer blieb („Undefinierte Obergruppe").
func istKlassenname(untergruppe string) bool {
	u := strings.TrimSpace(untergruppe)
	if einfuehrungsphase.MatchString(u) || inklusionsklasse.MatchString(u) {
		return true
	}
	treffer := klassenMuster.FindStringSubmatch(u)
	if treffer == nil {
		return false
	}
	_, ok := abschlussJahrgang(treffer[2])
	return ok
}

// artDerGruppe ordnet eine Lesergruppe einer Art zu, in dieser Reihenfolge: die Zuordnung des
// Laufs, die Untergruppe als Art („Schüler", so bis 2010), die Obergruppe (heute steht in der
// Untergruppe die Klasse) und zuletzt der Name als Klasse. Was keiner Regel folgt, bleibt
// ArtUnbekannt und hält den Lauf an (OhneZuordnung).
func artDerGruppe(untergruppe, obergruppe string, zuordnung Gruppenzuordnung) LeserArt {
	if ziel, ok := zuordnung[untergruppe]; ok {
		return artAusUntergruppe(ziel)
	}
	if art := artAusUntergruppe(untergruppe); art != ArtUnbekannt {
		return art
	}
	if art := artAusObergruppe(obergruppe); art != ArtUnbekannt {
		return art
	}
	if istKlassenname(untergruppe) {
		return ArtSchueler
	}
	return ArtUnbekannt
}

// klasseDerGruppe ist die Klasse der Gruppe: die Untergruppe, wenn sie eine Klasse benennt
// („11T1"), sonst die Kurzbezeichnung (bis 2010 „07H1" neben der Untergruppe „Schüler"). Heute
// trägt die Kurzbezeichnung der Einführungsphase „ET1" ohne Jahrgang; „11T1" steht in der
// Untergruppe.
func klasseDerGruppe(untergruppe, kurzBez string) string {
	if istKlassenname(untergruppe) {
		return strings.TrimSpace(untergruppe)
	}
	return strings.TrimSpace(kurzBez)
}

// leseObergruppen liest `Leser_OG` als Schlüssel → Bezeichnung der Obergruppe.
func leseObergruppen(r io.Reader) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	obergruppen := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		if id := strings.TrimSpace(z["Buchungsnummer"]); id != "" {
			obergruppen[id] = strings.TrimSpace(z["Obergruppe"])
		}
	}
	return obergruppen, nil
}

// leseLesergruppen liest `Leser_UG` als Schlüssel → Klasse + Art. Eine Zuordnung, deren
// Untergruppe es nicht gibt, ist ein Tippfehler und bricht ab, statt nichts zu bewirken.
func leseLesergruppen(r io.Reader, obergruppen map[string]string, zuordnung Gruppenzuordnung) (map[string]lesergruppe, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	gruppen := make(map[string]lesergruppe, len(zeilen))
	gesehen := map[string]bool{}
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue
		}
		untergruppe := strings.TrimSpace(z["Untergruppe"])
		gesehen[untergruppe] = true
		gruppen[id] = lesergruppe{
			Klasse:      klasseDerGruppe(untergruppe, z["KurzBez"]),
			Bezeichnung: untergruppe,
			Art:         artDerGruppe(untergruppe, obergruppen[strings.TrimSpace(z["Obergruppe"])], zuordnung),
		}
	}
	var fehlend []string
	for name := range zuordnung {
		if !gesehen[name] {
			fehlend = append(fehlend, name)
		}
	}
	if len(fehlend) > 0 {
		sort.Strings(fehlend)
		return nil, fmt.Errorf("zuordnung nennt Lesergruppen, die in Leser_UG fehlen: %s", strings.Join(fehlend, ", "))
	}
	return gruppen, nil
}

// leseLeser liest die Tabelle `Leser` und ordnet jede Person über die Lesergruppe ein.
//
// Die Klasse steht nicht am Leser: Sie kommt aus Leser_UG (klasseDerGruppe). Die naheliegende
// Tabelle `LeserSchueler` (mit Jahrgang, Abgang, Klassenlehrer) ist im Altbestand
// vollständig leer — 705 Zeilen, alle Werte 0. Wer darauf baut, baut auf nichts.
func leseLeser(r io.Reader, gruppen map[string]lesergruppe) ([]Leser, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}

	leser := make([]Leser, 0, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue // ohne internen Schlüssel lässt sich keine Ausleihe zuordnen
		}
		gruppeNr := strings.TrimSpace(z["Lesergruppe"])
		gruppe := gruppen[gruppeNr]
		leser = append(leser, Leser{
			ID:           id,
			Lesernummer:  strings.TrimSpace(z["Lesernummer"]),
			Vorname:      strings.TrimSpace(z["Vorname"]),
			Nachname:     strings.TrimSpace(z["Nachname"]),
			Klasse:       gruppe.Klasse,
			Art:          gruppe.Art,
			Gruppe:       gruppe.Bezeichnung,
			GruppeNr:     gruppeNr,
			Geburtsdatum: strings.TrimSpace(z["Geburtsdatum"]),
			EMail:        strings.TrimSpace(z["eMail"]),
			Strasse:      strings.TrimSpace(z["Adresse"]),
			PLZ:          strings.TrimSpace(z["PLZ"]),
			Ort:          strings.TrimSpace(z["Ort"]),
			Sperrgrund:   sperrgrundAus(z["GesperrtAm"], z["SperrenBegründung"]),
		})
	}
	return leser, nil
}

// nurArt filtert die Leser einer Art heraus — der Schreibpfad bekommt so genau die
// Menge, die in seine Zieltabelle gehört, statt selbst sortieren zu müssen.
func nurArt(leser []Leser, art LeserArt) []Leser {
	var treffer []Leser
	for _, l := range leser {
		if l.Art == art {
			treffer = append(treffer, l)
		}
	}
	return treffer
}
