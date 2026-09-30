package littera

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"bibliothek/pkg/lmf"
)

// Die Schlagworte der Titel (docs/OFFEN.md 4.20, Stufe 2, entschieden am 30.09.2026: alle
// mitnehmen). Littera führt sie in der Tabelle Schlagworte (Buchungsnummer → SuchWort) und
// ordnet sie über Schlag_zuord den Titeln zu — dieselbe Form wie die Verfasser über Personen
// und Personen_Zuordnung (personen.go).
//
// Die Verweise (Verweise_Schlagworte, Verweis_Zu_Schlagworte) werden gezählt, nicht
// übernommen: In der Sicherung von 2010 sind beide Tabellen leer, und wie Littera sie
// verknüpft, ist an echten Daten nicht zu sehen. Geraten wird nicht; stehen in einer Sicherung
// welche, nennt der Lauf die Zahl.

// SchlagwortQuelle ist, was der Export über die Schlagworte sagt.
type SchlagwortQuelle struct {
	// JeTitel bildet Titel.Buchungsnummer → Wörter ab, in Littera-Reihenfolge (Sortierung,
	// dann Erfassung) und so, wie sie dort stehen; Leerraum, Leeres und Doppelte regelt der
	// Schreibpfad (repository.NormalisiereSchlagworte).
	JeTitel map[string][]string
	// Zuordnungen zählt die Zeilen von Schlag_zuord, OhneWort die darunter, deren Wort es in
	// der Tabelle Schlagworte nicht gibt.
	Zuordnungen, OhneWort int
	// VerweisWoerter und VerweisZuordnungen zählen die Zeilen der beiden Verweis-Tabellen.
	VerweisWoerter, VerweisZuordnungen int
}

// Die Interessenkreise (entschieden am 30.09.2026, docs/OFFEN.md 4.20: als Schlagworte
// übernehmen). Littera führt sie als eigene Liste, gedacht als „thematische Gliederung des
// Belletristikbereiches (z.B. Krimi, Heimat)" (Handbuch). An dieser Schule sind es Zielgruppen:
// in der Sicherung von 2010 36 Werte an 8.902 Titeln — Sekundarstufe 1 und 2, Lehrer (1.813
// Titel), Referendare (986), DAZ-Schüler (209). Die Form gleicht den Schlagworten: Tabelle
// Interessenskreis (Buchungsnummer → Interessenskreis), über IntZuMed den Titeln zugeordnet.
// Littera selbst kann jeden Interessenkreis samt aller Titel in ein Schlagwort umwandeln; hier
// geschieht das beim Schreiben (SchreibeSchlagworte), und die Stufe ergibt die Jahrgangsspanne
// (lernmittelUndFach), wie im Katalogisat-Import. An Lesern (IntZuLeser) hat die Schule sie nie
// benutzt; die Tabelle wird nicht gelesen.

// LeseSchlagworte liest die Tabelle `Schlagworte` als Buchungsnummer → Wort.
func LeseSchlagworte(r io.Reader) (map[string]string, error) {
	return leseWortliste(r, "SuchWort")
}

// LeseInteressenkreise liest die Tabelle `Interessenskreis` als Buchungsnummer → Wort.
func LeseInteressenkreise(r io.Reader) (map[string]string, error) {
	return leseWortliste(r, "Interessenskreis")
}

// leseWortliste liest eine Littera-Wortliste (Buchungsnummer → Wort aus der Spalte spalte).
func leseWortliste(r io.Reader, spalte string) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	if err := spaltenDa(zeilen, "Buchungsnummer", spalte); err != nil {
		return nil, err
	}
	woerter := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		if id := strings.TrimSpace(z["Buchungsnummer"]); id != "" {
			woerter[id] = z[spalte]
		}
	}
	return woerter, nil
}

// schlagwortZuordnung ist eine Zeile von Schlag_zuord.
type schlagwortZuordnung struct {
	wort            string
	sortierung, lfd int // lfd: Buchungsnummer der Zuordnung — hält die Erfassungsreihenfolge
}

// SchlagworteJeTitel löst die Schlagworte der Titel über Schlag_zuord auf.
func SchlagworteJeTitel(woerter map[string]string, zuordnungen io.Reader) (SchlagwortQuelle, error) {
	return woerterJeTitel(woerter, zuordnungen, "Schlagwort")
}

// InteressenkreiseJeTitel löst die Interessenkreise der Titel über IntZuMed auf. IntZuMed hat
// keine Spalte Sortierung; die Reihenfolge ist die der Erfassung.
func InteressenkreiseJeTitel(woerter map[string]string, zuordnungen io.Reader) (SchlagwortQuelle, error) {
	return woerterJeTitel(woerter, zuordnungen, "Interessenskreis")
}

// woerterJeTitel löst eine Littera-Zuordnungstabelle (Titel → Wort aus der Spalte spalte) auf.
func woerterJeTitel(woerter map[string]string, zuordnungen io.Reader, spalte string) (SchlagwortQuelle, error) {
	zeilen, err := leseTabelle(zuordnungen)
	if err != nil {
		return SchlagwortQuelle{}, err
	}
	if err := spaltenDa(zeilen, "Buchungsnummer", "Titel", spalte); err != nil {
		return SchlagwortQuelle{}, err
	}
	q := SchlagwortQuelle{JeTitel: map[string][]string{}, Zuordnungen: len(zeilen)}
	jeTitel := map[string][]schlagwortZuordnung{}
	for _, z := range zeilen {
		titelID := strings.TrimSpace(z["Titel"])
		wort, bekannt := woerter[strings.TrimSpace(z[spalte])]
		if titelID == "" || !bekannt {
			q.OhneWort++
			continue
		}
		jeTitel[titelID] = append(jeTitel[titelID], schlagwortZuordnung{
			wort: wort, sortierung: zahlOderNull(z["Sortierung"]), lfd: zahlOderNull(z["Buchungsnummer"]),
		})
	}
	for titelID, liste := range jeTitel {
		sort.SliceStable(liste, func(i, j int) bool {
			if liste[i].sortierung != liste[j].sortierung {
				return liste[i].sortierung < liste[j].sortierung
			}
			return liste[i].lfd < liste[j].lfd
		})
		for _, e := range liste {
			q.JeTitel[titelID] = append(q.JeTitel[titelID], e.wort)
		}
	}
	return q, nil
}

// zaehleZeilen zählt die Datenzeilen einer Tabelle (für die Verweise, die nur gezählt werden).
func zaehleZeilen(r io.Reader) (int, error) {
	zeilen, err := leseTabelle(r)
	return len(zeilen), err
}

// zahlOderNull liest eine Zahl; eine unlesbare kostet nur die Reihenfolge, nicht das Wort.
func zahlOderNull(roh string) int {
	n, err := strconv.Atoi(strings.TrimSpace(roh))
	if err != nil {
		return 0
	}
	return n
}

// spaltenDa prüft, dass die Tabelle die gebrauchten Spalten trägt. leseTabelle liefert für eine
// fehlende Spalte still einen leeren Wert — bei den Schlagworten fiele dann jedes Wort als leer
// weg, und der Lauf meldete trotzdem Erfolg. Eine Tabelle ohne Datenzeilen hat nichts zu prüfen.
func spaltenDa(zeilen []map[string]string, namen ...string) error {
	if len(zeilen) == 0 {
		return nil
	}
	for _, name := range namen {
		if _, da := zeilen[0][name]; !da {
			return fmt.Errorf("spalte %q fehlt", name)
		}
	}
	return nil
}

// lernmittelUndFach liest Lernmittel, Fach und Jahrgang aus der Signatur und nimmt das Fach aus
// den Schlagworten, wo die Signatur keins nennt — dieselbe Reihenfolge wie der
// Katalogisat-Import (kategorisiere in internal/service/import_service.go, seit dem
// 02.09.2026), und wie dort nur, wenn die Schlagworte genau ein Fach nennen: lieber leer als
// falsch (lmf.FachAusSchlagworten). Bis zum 30.09.2026 las die Übernahme nur die Signatur; in
// der Sicherung von 2010 trugen so 258 Titel ein Fach, mit den Schlagworten 2.945 mehr.
// ausSchlagworten sagt, ob das Fach aus den Schlagworten kam.
//
// Die Jahrgangsspanne kommt aus den Interessenkreisen, wo die Signatur keine nennt
// („Sekundarstufe 2" → 11–13, lmf.JahrgangAusZielgruppen über alle Werte) — wie im
// Katalogisat-Import, dessen 070b dieselben Werte trägt. Das Fach kommt nie aus ihnen.
func lernmittelUndFach(signatur string, schlagworte, interessenkreise []string) (lern lernmittelfelder, ausSchlagworten bool) {
	lern = lernmittelAusSignatur(signatur)
	if lern.JahrgangVon == 0 {
		lern.JahrgangVon, lern.JahrgangBis = lmf.JahrgangAusZielgruppen(interessenkreise)
	}
	if lern.Fach != "" {
		return lern, false
	}
	lern.Fach = lmf.FachAusSchlagworten(schlagworte)
	return lern, lern.Fach != ""
}

// ZaehleFachquellen sagt vor dem Lauf, wie viele Titel ihr Fach aus der Signatur bekommen und
// wie viele aus den Schlagworten (lernmittelUndFach) — für den Trockenlauf.
func ZaehleFachquellen(ab *Altbestand) (ausSignatur, ausSchlagworten int) {
	for _, t := range ab.Titel {
		switch lern, sw := lernmittelUndFach(ab.Signaturen[t.ID], ab.Schlagworte.JeTitel[t.ID],
			ab.Interessenkreise.JeTitel[t.ID]); {
		case sw:
			ausSchlagworten++
		case lern.Fach != "":
			ausSignatur++
		}
	}
	return ausSignatur, ausSchlagworten
}
