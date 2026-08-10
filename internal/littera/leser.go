package littera

import (
	"io"
	"strings"
)

// LeserArt sagt, wohin ein Littera-Leser gehört.
//
// Littera führt alle Personen in EINER Tabelle `Leser` — Schüler, Lehrkräfte,
// Praktikanten, das Sekretariat und sogar Sammelkonten der Fachbereiche. Diese
// Anwendung trennt sie: Schüler nach `schueler`, Lehrkräfte nach `benutzer` mit der
// Rolle `kollegium` (hieß bis Migration 069 `lehrer`). Ohne diese Einordnung landeten
// 158 Lehrkräfte in der Schülerdatei.
//
// Die Unterscheidung steht in den Daten selbst und muss nicht geraten werden:
// `Leser.Lesergruppe` → `Leser_UG.Untergruppe`.
type LeserArt int

const (
	// ArtUnbekannt steht für eine nicht auflösbare oder unklare Untergruppe (z. B. „IMPORT",
	// „U-plus", „Im Ausland"). Diese Zeilen werden NICHT still zu Schülern gemacht —
	// sie brauchen eine Entscheidung, bevor Personendaten geschrieben werden.
	ArtUnbekannt LeserArt = iota
	// ArtSchueler umfasst „Schüler" UND „Sekundarstufe II": Die Oberstufenklassen
	// (11T1, 12T3, 13T5) sind eine eigene Untergruppe, aber selbstverständlich Schüler.
	ArtSchueler
	// ArtLehrkraft umfasst „Lehrer" und „Lehrerin".
	ArtLehrkraft
	// ArtAbgegangen sind ehemalige Schüler — Untergruppe „Abgegangen". Gehören als
	// Abgänger in die Schülerdatei, nicht als aktive Schüler.
	ArtAbgegangen
	// ArtSonstige sind Praktikanten, Sekretariat, Fachbereichs-Sammelkonten. Keine Schüler,
	// aber auch keine Lehrkräfte im Sinne des Portals.
	ArtSonstige
)

// Lesergruppe ist eine Zeile aus `Leser_UG` — Klassenbezeichnung plus Art.
type Lesergruppe struct {
	Klasse string // KurzBez: "07H1", "12T3" — die Klasse, wie sie an der Schule heißt
	Art    LeserArt
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
	case "Schüler", "Sekundarstufe II":
		return ArtSchueler
	case "Lehrer", "Lehrerin":
		return ArtLehrkraft
	case "Abgegangen":
		return ArtAbgegangen
	case "Praktikant", "Praktikantin", "Sekretärin":
		return ArtSonstige
	}
	// Sammelkonten der Fachbereiche sind keine Personen.
	if strings.HasPrefix(b, "Fachbereich") {
		return ArtSonstige
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
			Klasse: strings.TrimSpace(z["KurzBez"]),
			Art:    artAusUntergruppe(z["Untergruppe"]),
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
		gruppe := gruppen[strings.TrimSpace(z["Lesergruppe"])]
		leser = append(leser, Leser{
			ID:           id,
			Lesernummer:  strings.TrimSpace(z["Lesernummer"]),
			Vorname:      strings.TrimSpace(z["Vorname"]),
			Nachname:     strings.TrimSpace(z["Nachname"]),
			Klasse:       gruppe.Klasse,
			Art:          gruppe.Art,
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
