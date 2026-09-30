package littera

import (
	"io"
	"strings"
)

// LeserArt sagt, wohin ein Littera-Leser gehört.
//
// Littera führt alle Personen in EINER Tabelle `Leser` — Schüler, Lehrkräfte,
// Praktikanten, das Sekretariat und sogar Sammelkonten der Fachbereiche. Diese
// Anwendung trennt sie: Schüler nach `schueler`, alle anderen ins Kollegium, jeweils mit
// ihrer Art an der Leserzeile (leser.art, ZielArt). Ohne diese Einordnung landeten
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
	// in Littera selbst oder hier in artAusUntergruppe (Entscheidung vom 28.09.2026).
	ArtUnbekannt LeserArt = iota
	// ArtSchueler umfasst „Schüler" UND „Sekundarstufe II": Die Oberstufenklassen
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
	// ArtPraktikum sind Praktikanten: keine Schüler, aber Entleiher. Sie kommen ins
	// Kollegium, werden also nicht gemahnt, und bekommen kein Konto (repository.ArtMitKonto).
	// Bis zum 28.09.2026 übernahm der Lauf die Sonderkonten nicht, und mit ihnen fehlten ihre
	// Ausleihen; bis zum 30.09.2026 kamen sie als „lehrkraft" an, und ihre Littera-Gruppe
	// stand nur im Protokoll (docs/OFFEN.md 5.18).
	ArtPraktikum
	// ArtLiV sind Referendare — in Hessen LiV, Lehrkraft im Vorbereitungsdienst. Sie gehören ins
	// Kollegium mit der Art „liv" (Migration 119). Bis zum 15.09.2026 fielen sie unter
	// ArtUnbekannt und wurden nicht übernommen.
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

// ZielArt ist die Art der Leserzeile, in der ein Littera-Leser landet (chk_leser_art,
// Migration 153); "" für einen Leser ohne Zuordnung, den der Lauf nicht schreibt.
func (a LeserArt) ZielArt() string {
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

// Lesergruppe ist eine Zeile aus `Leser_UG` — Klassenbezeichnung plus Art.
type Lesergruppe struct {
	Klasse      string // KurzBez: "07H1", "12T3" — die Klasse, wie sie an der Schule heißt
	Bezeichnung string // Untergruppe, wie Littera sie führt: „Schüler", „Fachbereich Erdkunde"
	Art         LeserArt
}

// Leser ist eine Person aus dem Littera-Altbestand.
//
// ACHTUNG, zwei Nummern mit verschiedenen Aufgaben:
//
//   - ID = `Buchungsnummer` ist der INTERNE Schlüssel. Darauf zeigt `Verleih.Leser` —
//     gemessen: 15.615 von 15.615 Ausleihen treffen darüber, über `Lesernummer` nur 792.
//   - Lesernummer ist die Ausweisnummer, die der Schüler in der Hand hält. Sie gehört
//     nach `schueler.barcode_id`, taugt aber NICHT zum Verknüpfen der Ausleihen.
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
}

// artAusUntergruppe ordnet die Untergruppen-Bezeichnung einer Art zu.
//
// Bewusst über die BEZEICHNUNG und nicht über die Nummer: Die Nummern sind
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

// LeseLesergruppen liest `Leser_UG` als Schlüssel → Klasse + Art.
func LeseLesergruppen(r io.Reader) (map[string]Lesergruppe, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	gruppen := make(map[string]Lesergruppe, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue
		}
		gruppen[id] = Lesergruppe{
			Klasse:      strings.TrimSpace(z["KurzBez"]),
			Bezeichnung: strings.TrimSpace(z["Untergruppe"]),
			Art:         artAusUntergruppe(z["Untergruppe"]),
		}
	}
	return gruppen, nil
}

// LeseLeser liest die Tabelle `Leser` und ordnet jede Person über die Lesergruppe ein.
//
// Die Klasse steht NICHT am Leser: Sie kommt aus Leser_UG.KurzBez. Die naheliegende
// Tabelle `LeserSchueler` (mit Jahrgang, Abgang, Klassenlehrer) ist im Altbestand
// vollständig leer — 705 Zeilen, alle Werte 0. Wer darauf baut, baut auf nichts.
func LeseLeser(r io.Reader, gruppen map[string]Lesergruppe) ([]Leser, error) {
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
		})
	}
	return leser, nil
}

// NurArt filtert die Leser einer Art heraus — der Schreibpfad bekommt so genau die
// Menge, die in seine Zieltabelle gehört, statt selbst sortieren zu müssen.
func NurArt(leser []Leser, art LeserArt) []Leser {
	var treffer []Leser
	for _, l := range leser {
		if l.Art == art {
			treffer = append(treffer, l)
		}
	}
	return treffer
}
