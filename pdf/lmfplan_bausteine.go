package pdf

// lmfplan_bausteine.go — aus Abschnitten werden Bausteine, aus Bausteinen Spalten. Der
// Satzspiegel und seine Begründung stehen in lmfplan_satz.go.

import (
	"math"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/breakline"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// lmfSatzbau baut die Bausteine eines Versuchs — Maße, Satzart und Messer stehen fest.
type lmfSatzbau struct {
	masse      lmfMasse
	modus      lmfModus
	messer     *lmfMesser
	abschnitte []LmfPlanAbschnitt
}

// stil ist der Textstil, mit dem gesetzt UND gemessen wird — eine Quelle für beides.
func (b lmfSatzbau) stil(groesse float64, s fontstyle.Type, a align.Type) props.Text {
	return props.Text{Size: groesse, Style: s, Align: a, BreakLineStrategy: breakline.EmptySpaceStrategy}
}

// einheiten rechnet Textzeilen in Einheitszeilen um, mindestens eine.
func (b lmfSatzbau) einheiten(zeilen int, groesse float64) int {
	n := int(math.Ceil(float64(zeilen) * groesse * lmfPunktInMm * lmfLuft / b.masse.einheit))
	return max(n, 1)
}

// breiterBaustein ist eine Zeile über die volle Spaltenbreite (Überschriften).
func (b lmfSatzbau) breiterBaustein(text string, groesse float64, s fontstyle.Type, art lmfArt, i int) lmfBaustein {
	stil := b.stil(groesse, s, align.Center)
	zeilen := b.messer.zeilen(text, groesse, s, b.modus.spaltenBreite())
	return lmfBaustein{
		zellen:    []lmfZelle{{breite: lmfGrid, text: text, stil: stil}},
		zeilen:    b.einheiten(zeilen, groesse),
		art:       art,
		abschnitt: i,
	}
}

// fuenfSpalter setzt eine Tabellenzeile: fünf Zellen im Raster der Satzart. Die letzte
// Zelle bekommt rechts Luft — zweispaltig ist das der Steg zwischen den Tabellen.
func (b lmfSatzbau) fuenfSpalter(texte [5]string, fett [5]bool, art lmfArt, i int) lmfBaustein {
	groesse := b.zeilenGroesse(texte, fett)
	zellen := make([]lmfZelle, 0, 5)
	textZeilen := 1
	for k, t := range texte {
		stil := b.stil(groesse, lmfSchnitt(fett[k]), align.Left)
		stil.Right = b.luft(k)
		zellen = append(zellen, lmfZelle{breite: b.modus.zellen[k], text: t, stil: stil})
		textZeilen = max(textZeilen, b.messer.zeilen(t, groesse, stil.Style, b.platz(k)))
	}
	return lmfBaustein{zellen: zellen, zeilen: b.einheiten(textZeilen, groesse), art: art, abschnitt: i}
}

// zeilenGroesse ist die Schriftgröße einer Tabellenzeile: die kleinste, die eine ihrer
// Zellen braucht, damit kein Wort in die Nachbarspalte ragt — für alle fünf Zellen
// dieselbe, sonst stünde eine Zeile in zwei Größen.
func (b lmfSatzbau) zeilenGroesse(texte [5]string, fett [5]bool) float64 {
	groesse := b.masse.basis
	for k, t := range texte {
		groesse = min(groesse, b.messer.passendeGroesse(t, b.masse.basis, lmfSchnitt(fett[k]), b.platz(k)))
	}
	return groesse
}

// luft ist der Abstand rechts einer Zelle: zwischen zwei Spalten ein Haarraum, hinter der
// letzten der Steg zur Nachbartabelle. Ohne ihn stößt „Wochentag" an „Datum".
func (b lmfSatzbau) luft(k int) float64 {
	if k == len(b.modus.zellen)-1 {
		return b.modus.steg * b.masse.faktor
	}
	return lmfZellenLuft * b.masse.faktor
}

// platz ist die Breite, die einer Zelle für Text bleibt.
func (b lmfSatzbau) platz(k int) float64 {
	return b.modus.zellenBreite(k) - b.luft(k)
}

// lmfSchnitt übersetzt „fett?" in den Schriftschnitt.
func lmfSchnitt(fett bool) fontstyle.Type {
	if fett {
		return fontstyle.Bold
	}
	return fontstyle.Normal
}

// tabellenkopf ist die Zeile „Wochentag | Datum | Stunde | Klassen | Besonderheiten".
func (b lmfSatzbau) tabellenkopf(i int) lmfBaustein {
	kopf := [5]string{"Wochentag", "Datum", "Stunde", "Klassen", "Besonderheiten"}
	if b.modus.kurzerTag {
		kopf[0], kopf[2] = "Tag", "Std."
	}
	return b.fuenfSpalter(kopf, [5]bool{true, true, true, true, true}, lmfKopf, i)
}

// datenzeile ist ein Termin. Der Wochentag steht zweispaltig abgekürzt — „Donnerstag"
// bräuchte dort mehr Platz als die halbe Tabelle.
func (b lmfSatzbau) datenzeile(z LmfPlanZeile, i int) lmfBaustein {
	tag := Wochentag(z.Datum)
	if b.modus.kurzerTag {
		tag = WochentagKurz(z.Datum)
	}
	return b.fuenfSpalter(
		[5]string{tag, z.Datum.Format(dateFormatDE), stundeText(z.Stunde), z.Klassen, z.Vermerk},
		[5]bool{false, false, false, true, false}, lmfDaten, i)
}

// bausteine setzt den ganzen Plan in Bausteine: je Abschnitt Überschrift, Erklärungssatz,
// Sortierhinweis, Tabellenkopf, Termine — und zwischen zwei Abschnitten eine Leerzeile.
// Alle Abschnittsköpfe bekommen dieselbe Höhe: sonst begänne die eine Tabelle eine Zeile
// höher als die daneben, nur weil ihr Erklärungssatz kürzer umbricht.
func (b lmfSatzbau) bausteine() []lmfBaustein {
	koepfe := make([][]lmfBaustein, len(b.abschnitte))
	hoehe := 0
	for i, a := range b.abschnitte {
		if len(a.Zeilen) == 0 {
			continue
		}
		koepfe[i] = b.abschnittskopf(i, a)
		hoehe = max(hoehe, lmfHoehe(koepfe[i]))
	}
	var alle []lmfBaustein
	for i, a := range b.abschnitte {
		if len(a.Zeilen) == 0 {
			continue
		}
		if len(alle) > 0 {
			alle = append(alle, lmfBaustein{zeilen: 1, art: lmfAbstand, abschnitt: i})
		}
		alle = append(alle, lmfAufHoehe(koepfe[i], hoehe, b.modus.spalten)...)
		for _, z := range a.Zeilen {
			alle = append(alle, b.datenzeile(z, i))
		}
	}
	return alle
}

// abschnittskopf ist alles, was über der Tabelle eines Abschnitts steht.
func (b lmfSatzbau) abschnittskopf(i int, a LmfPlanAbschnitt) []lmfBaustein {
	kopf := []lmfBaustein{b.breiterBaustein(a.Titel, b.masse.titel, fontstyle.Bold, lmfKopf, i)}
	if a.Untertitel != "" {
		kopf = append(kopf, b.breiterBaustein(a.Untertitel, b.masse.unter, fontstyle.Italic, lmfKopf, i))
	}
	return append(kopf,
		b.breiterBaustein(lmfSortierhinweis, b.masse.basis, fontstyle.Bold, lmfKopf, i),
		b.tabellenkopf(i))
}

// lmfAufHoehe streckt einen Abschnittskopf auf die gemeinsame Höhe — die Luft wächst
// zwischen Erklärungssatz und Sortierhinweis, nicht zwischen Kopf und erster Zeile.
// Einspaltig steht kein Kopf neben einem anderen: dort wäre das nur Leerraum.
func lmfAufHoehe(kopf []lmfBaustein, hoehe, spalten int) []lmfBaustein {
	fehlt := hoehe - lmfHoehe(kopf)
	if spalten < 2 || len(kopf) < 3 || fehlt <= 0 {
		return kopf
	}
	gestreckt := append([]lmfBaustein(nil), kopf...)
	gestreckt[len(gestreckt)-3].zeilen += fehlt
	return gestreckt
}

// fortsetzung ist der Kopf, mit dem ein in der Mitte umbrochener Abschnitt in der nächsten
// Spalte weitergeht — ohne ihn stünden dort Termine ohne Überschrift und ohne Tabellenkopf.
func (b lmfSatzbau) fortsetzung(i int) []lmfBaustein {
	titel := b.abschnitte[i].Titel + " (Fortsetzung)"
	return []lmfBaustein{
		b.breiterBaustein(titel, b.masse.titel, fontstyle.Bold, lmfKopf, i),
		b.tabellenkopf(i),
	}
}
