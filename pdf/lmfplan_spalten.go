package pdf

// lmfplan_spalten.go — die Suche nach dem Satz, der auf ein Blatt passt, und das Setzen
// der Einheitszeilen. Warum überhaupt gesucht wird: lmfplan_satz.go.

import (
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/core"
)

// lmfHoehe ist die Höhe einer Bausteinfolge in Einheitszeilen.
func lmfHoehe(bs []lmfBaustein) int {
	h := 0
	for _, b := range bs {
		h += b.zeilen
	}
	return h
}

// lmfKette liefert die Bausteine ab i, die zusammen umbrechen müssen: eine Folge von
// Köpfen zieht die erste Datenzeile mit — eine Überschrift allein am Spaltenende ist ein
// Versprechen, das die Spalte nicht mehr einlöst.
func lmfKette(bs []lmfBaustein, i int) []lmfBaustein {
	if bs[i].art != lmfKopf {
		return bs[i : i+1]
	}
	j := i
	for j < len(bs) && bs[j].art == lmfKopf {
		j++
	}
	if j < len(bs) {
		j++
	}
	return bs[i:j]
}

// lmfSatz ist der gewählte Satz: Maße, Satzart und der Inhalt, auf die Spalten verteilt.
type lmfSatz struct {
	masse   lmfMasse
	modus   lmfModus
	spalten [][]lmfBaustein
}

// lmfSatzWaehlen probiert die Satzarten in ihrer Reihenfolge und je Satzart die Faktoren
// von voller Größe abwärts — der erste Versuch, der auf ein Blatt passt, gewinnt. Der
// letzte Modus geht bis 0,15 hinunter (rund 700 Termine); scheitert auch der, setzen wir
// ihn trotzdem: ein winziger Plan ist besser als ein zerrissener.
func lmfSatzWaehlen(abschnitte []LmfPlanAbschnitt) lmfSatz {
	messer := lmfNeuerMesser()
	var letzter lmfSatz
	for _, mo := range lmfModi {
		for f := 1.0; f >= mo.minFaktor-1e-9; f -= lmfSchritt {
			satz, ok := lmfVersuch(abschnitte, mo, f, messer)
			letzter = satz
			if ok {
				return satz
			}
		}
	}
	return letzter
}

// lmfVersuch verteilt den Plan bei gegebener Satzart und Größe auf die Spalten. Gesucht
// ist die KLEINSTE Spaltenhöhe, mit der alles untergebracht ist: so sind die Spalten
// gleich lang statt eine voll und eine halb leer.
func lmfVersuch(abschnitte []LmfPlanAbschnitt, mo lmfModus, faktor float64, messer *lmfMesser) (lmfSatz, bool) {
	bau := lmfSatzbau{masse: lmfMasseFuer(faktor), modus: mo, messer: messer, abschnitte: abschnitte}
	satz := lmfSatz{masse: bau.masse, modus: mo}
	bausteine := bau.bausteine()
	frei := lmfSeiteHoehe - lmfRandOben - lmfRandUnten - bau.masse.kopfHoehe() - bau.masse.fussHoehe() - lmfSicherheit
	maxKap := int(frei / bau.masse.einheit)
	von := (lmfHoehe(bausteine) + mo.spalten - 1) / mo.spalten
	for kap := max(von, 1); kap <= maxKap; kap++ {
		if verteilt, ok := lmfVerteile(bau, bausteine, kap); ok {
			satz.spalten = verteilt
			return satz, true
		}
	}
	satz.spalten = lmfNotverteilung(bau, bausteine, maxKap)
	return satz, false
}

// lmfVerteile füllt die Spalten der Reihe nach, höchstens kap Einheitszeilen je Spalte.
// Bricht ein Abschnitt mitten in seinen Terminen um, beginnt die neue Spalte mit seinem
// Fortsetzungskopf.
func lmfVerteile(bau lmfSatzbau, bausteine []lmfBaustein, kap int) ([][]lmfBaustein, bool) {
	spalten := make([][]lmfBaustein, bau.modus.spalten)
	aktuell, frei := 0, kap
	for i := 0; i < len(bausteine); {
		if bausteine[i].art == lmfAbstand {
			// Ein Abstand trennt zwei Abschnitte. Er bricht nie um: am Spaltenanfang
			// wäre er ein Loch, am Spaltenende unsichtbar.
			if len(spalten[aktuell]) > 0 && bausteine[i].zeilen <= frei {
				spalten[aktuell] = append(spalten[aktuell], bausteine[i])
				frei -= bausteine[i].zeilen
			}
			i++
			continue
		}
		kette := lmfKette(bausteine, i)
		hoehe := lmfHoehe(kette)
		if hoehe > frei {
			if aktuell+1 >= bau.modus.spalten {
				return nil, false
			}
			aktuell, frei = aktuell+1, kap
			if bausteine[i].art == lmfDaten {
				kopf := bau.fortsetzung(bausteine[i].abschnitt)
				spalten[aktuell] = append(spalten[aktuell], kopf...)
				frei -= lmfHoehe(kopf)
			}
			if hoehe > frei {
				return nil, false
			}
		}
		spalten[aktuell] = append(spalten[aktuell], kette...)
		frei -= hoehe
		i += len(kette)
	}
	return spalten, true
}

// lmfNotverteilung ist der Ausweg, wenn selbst die kleinste Schrift nicht reicht (jenseits
// von rund 700 Terminen): Dann steht alles untereinander in der ersten Spalte, und maroto
// bricht wieder auf mehrere Seiten um — sichtbar, statt dass Termine verschwinden.
func lmfNotverteilung(bau lmfSatzbau, bausteine []lmfBaustein, kap int) [][]lmfBaustein {
	verteilt, ok := lmfVerteile(bau, bausteine, kap)
	if ok {
		return verteilt
	}
	spalten := make([][]lmfBaustein, bau.modus.spalten)
	spalten[0] = bausteine
	return spalten
}

// linien zerlegt eine Spalte in Einheitszeilen: die erste trägt den Inhalt, die weiteren
// bleiben leer, damit ein umbrochener Text sichtbar hineinfließen kann.
func lmfLinien(bausteine []lmfBaustein) [][]lmfZelle {
	var linien [][]lmfZelle
	for _, b := range bausteine {
		linien = append(linien, b.zellen)
		for i := 1; i < b.zeilen; i++ {
			linien = append(linien, nil)
		}
	}
	return linien
}

// reihen setzt den Satz in maroto-Zeilen: je Einheitszeile eine Reihe, in ihr Spalte für
// Spalte die Zellen — leere Spalten füllen ihr Raster auf, sonst verrutscht die nächste.
func (s lmfSatz) reihen() []core.Row {
	linien := make([][][]lmfZelle, len(s.spalten))
	hoehe := 0
	for i, sp := range s.spalten {
		linien[i] = lmfLinien(sp)
		hoehe = max(hoehe, len(linien[i]))
	}
	reihen := make([]core.Row, 0, hoehe)
	for i := range hoehe {
		r := row.New(s.masse.einheit)
		for _, sp := range linien {
			if i >= len(sp) || sp[i] == nil {
				r.Add(col.New(lmfGrid))
				continue
			}
			for _, z := range sp[i] {
				if z.text == "" {
					r.Add(col.New(z.breite))
					continue
				}
				r.Add(col.New(z.breite).Add(text.New(z.text, z.stil)))
			}
		}
		reihen = append(reihen, r)
	}
	return reihen
}
