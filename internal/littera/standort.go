package littera

import (
	"sort"
	"strings"

	"bibliothek/pkg/lmf"
)

// Standorte stehen in Littera an zwei Stellen, und beide kommen ans Exemplar
// (buecher_exemplare.standort):
//
//   - `Exemplar.Sonderstandort`: Das Exemplar steht nicht an seinem Platz nach der Signatur
//     („Videoschrank", „Lehrerschrank"). Gemessen an der Sicherung von 2010: 1.609 von
//     61.520 Exemplaren, 99 verschiedene Werte.
//   - ein Vermerk am Titel. Littera hat dafür kein Feld; die Bibliothek trägt ihn als Person
//     an der Stelle des dritten Verfassers ein („LMF", „Bibliothek Klassensatz Regal 11",
//     „Schulseelsorge"). Im Export der Titelliste ist das MAB 108a, in `Personen_Zuordnung`
//     die vierte Stelle von `Flags`: 2010 standen dort 8.730 Zuordnungen mit 50 Namen.
//
// Der Vermerk am Titel gilt für jedes Exemplar des Titels ohne eigenen Sonderstandort.

// stelleDritterVerfasser ist die Stelle in `Personen_Zuordnung.Flags`, von 0 gezählt.
const stelleDritterVerfasser = 3

// Standortregel nennt, was an beiden Stellen dasteht und trotzdem kein Standort ist: ein
// Verfasser ohne Komma an der dritten Stelle („Louise Carleton-Gertsch") oder ein
// Sonderstandort, in den Littera beim Rundlauf einer Zeitschrift Nummer und Namen des
// letzten Lesers schreibt. Der Trockenlauf listet alle Werte, die Ausnahmen kommen vom
// Aufrufer (Schalter -kein-standort).
type Standortregel struct {
	KeinStandort map[string]bool
}

// istVermerk meldet, ob der Eintrag einer Verfasser-Zuordnung ein Standortvermerk ist: einer
// der fünf seit jeher bekannten an jeder Stelle, sonst ein Eintrag der dritten Stelle, der
// nicht in Katalogform („Nachname, Vorname") dasteht.
func (r Standortregel) istVermerk(name string, dritteStelle bool) bool {
	if name == "" || r.KeinStandort[name] {
		return false
	}
	return bestandsmarken[name] || (dritteStelle && !strings.Contains(name, ","))
}

// anDritterStelle liest die Stelle aus `Flags`. mdb-export lässt am Ende der Kette
// gelegentlich ein Zeichen weg; gezählt wird deshalb von vorn.
func anDritterStelle(flags string) bool {
	flags = strings.TrimSpace(flags)
	return len(flags) > stelleDritterVerfasser && flags[stelleDritterVerfasser] == '1'
}

// vermerkeInErfassungsreihenfolge sind die Standortvermerke eines Titels, ohne Dubletten:
// „U plus" steht oft zugleich als erster und als dritter Verfasser da.
func (r Standortregel) vermerkeInErfassungsreihenfolge(liste []autorZuordnung, personen map[string]string) []string {
	sort.Slice(liste, func(i, j int) bool { return liste[i].lfd < liste[j].lfd })

	var vermerke []string
	gesehen := map[string]bool{}
	for _, z := range liste {
		name := personen[z.personID]
		if gesehen[name] || !r.istVermerk(name, z.dritteStelle) {
			continue
		}
		gesehen[name] = true
		vermerke = append(vermerke, name)
	}
	return vermerke
}

// ohneVermerke nimmt aus einer freien Verfasserangabe die Teile, die an irgendeinem Titel
// als Standortvermerk erkannt sind. Der Freitext kennt keine Stellen; ohne diesen Schritt
// bliebe „Schulseelsorge" dort als Verfasser stehen, wo die Zuordnung keinen Namen liefert.
func ohneVermerke(autor string, vermerke map[string]bool) string {
	if autor == "" || len(vermerke) == 0 {
		return autor
	}
	var behalten []string
	for _, teil := range strings.Split(autor, ";") {
		if teil = strings.TrimSpace(teil); teil != "" && !vermerke[teil] {
			behalten = append(behalten, teil)
		}
	}
	return strings.Join(behalten, "; ")
}

// wendeStandortregelAn räumt die Sonderstandorte der Exemplare: Was der Aufrufer ausnimmt,
// kommt nicht mit.
func (r Standortregel) wendeStandortregelAn(exemplare []Exemplar) {
	for i := range exemplare {
		if r.KeinStandort[exemplare[i].Sonderstandort] {
			exemplare[i].Sonderstandort = ""
		}
	}
}

// StandortVon ist der Standort, mit dem ein Exemplar ankommt: sein eigener Sonderstandort,
// sonst der Vermerk seines Titels. vomTitel sagt, welche der beiden Stellen ihn liefert.
func StandortVon(e Exemplar, vermerkeDesTitels []string) (standort string, vomTitel bool) {
	if e.Sonderstandort != "" {
		return e.Sonderstandort, false
	}
	return strings.Join(vermerkeDesTitels, "; "), len(vermerkeDesTitels) > 0
}

// nenntLernmittel meldet, ob ein Vermerk am Titel den Bestand der Lernmittelfreiheit nennt
// („LMF", „LMF/Bibliothek"). Der Import der Titelliste liest denselben Vermerk aus MAB 108a
// (service.bookTitleAusFelder); ohne ihn kämen Lernmittel ohne LMF-Signatur als
// Bibliotheksbücher an.
func nenntLernmittel(vermerke []string) bool {
	for _, v := range vermerke {
		if lmf.HatVermerk(v) {
			return true
		}
	}
	return false
}

// Standortzahl ist ein Wert mit der Zahl der Titel oder Exemplare, die ihn tragen.
type Standortzahl struct {
	Wert   string
	Anzahl int
}

// StandortBilanz zählt für den Trockenlauf, was als Standort ankäme: die Vermerke am Titel
// und die Sonderstandorte der Exemplare, je häufigster Wert zuerst. Wer die Liste liest,
// sieht, was kein Standort ist, und nimmt es mit -kein-standort aus.
type StandortBilanz struct {
	Vermerke             []Standortzahl // je Vermerk: Titel
	Sonderstandorte      []Standortzahl // je Wert: Exemplare
	ExemplareEigen       int            // Exemplare mit eigenem Sonderstandort
	ExemplareVomTitel    int            // Exemplare, die den Vermerk ihres Titels bekommen
	LernmittelNurVermerk int            // Titel, die nur der Vermerk zum Lernmittel macht
}

// ZaehleStandorte rechnet die StandortBilanz aus dem gelesenen Altbestand.
func ZaehleStandorte(ab *Altbestand) StandortBilanz {
	var b StandortBilanz
	vermerke := map[string]int{}
	for titelID, liste := range ab.Standortvermerke {
		for _, v := range liste {
			vermerke[v]++
		}
		if nenntLernmittel(liste) && !lernmittelAusSignatur(ab.Signaturen[titelID]).IstLernmittel {
			b.LernmittelNurVermerk++
		}
	}
	sonder := map[string]int{}
	for _, e := range ab.Exemplare {
		switch _, vomTitel := StandortVon(e, ab.Standortvermerke[e.TitelID]); {
		case e.Sonderstandort != "":
			sonder[e.Sonderstandort]++
			b.ExemplareEigen++
		case vomTitel:
			b.ExemplareVomTitel++
		}
	}
	b.Vermerke, b.Sonderstandorte = nachHaeufigkeit(vermerke), nachHaeufigkeit(sonder)
	return b
}

func nachHaeufigkeit(zahlen map[string]int) []Standortzahl {
	liste := make([]Standortzahl, 0, len(zahlen))
	for wert, n := range zahlen {
		liste = append(liste, Standortzahl{Wert: wert, Anzahl: n})
	}
	sort.Slice(liste, func(i, j int) bool {
		if liste[i].Anzahl != liste[j].Anzahl {
			return liste[i].Anzahl > liste[j].Anzahl
		}
		return liste[i].Wert < liste[j].Wert
	})
	return liste
}
