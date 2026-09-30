package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// schlagwortStichwoerterMax kappt die Kandidaten eines Satzes. Die DNB liefert bis zu 98
// Verlagswörter je Titel (gemessen am 23.09.2026); die Kappung hält nur eine unerwartet
// lange Antwort von der Datenbank fern.
const schlagwortStichwoerterMax = 300

// SchlagworteAusStichwoertern gleicht die Stichwörter eines DNB-Satzes (Gattungsbegriffe und
// Verlagswörter, inventur.MetadatenErgebnis.Stichwoerter) gegen die eigene Liste ab —
// der Schlagwort-Vorschlag beim Bestellen per ISBN (docs/OFFEN.md 4.20). Vorgeschlagen wird
// nur, was es schon gibt: ein Schlagwort, das mindestens ein Titel trägt, oder ein Verweis
// darauf, aufgelöst zu seinem Ziel („Science Fiction" → „Science-Fiction"). Dieselbe Regel
// wie die Vorschlagsliste beim Tippen (SchlagwortVorschlaege: ein Wort ohne Titel ist meist
// ein Tippfehler, der nicht weiterleben soll) und dieselbe Auflösung wie der Schreibpfad
// (coalesce(verweis_auf, id)).
//
// Verglichen wird das ganze Wort ohne Rücksicht auf Groß- und Kleinschreibung, nach
// derselben Normalform wie beim Speichern (schlagwortNormalform: Leerraum zusammengezogen,
// Umlaute zusammengesetzt) — kein Teilstring: „Krieg" soll nicht „Kriegsende" vorschlagen.
// Die Antwort ist alphabetisch und ohne Doppelte; leer, wenn nichts passt. Geschrieben wird
// nichts: Das Ja gibt ein Mensch.
func SchlagworteAusStichwoertern(ctx context.Context, q DBQueryer, stichwoerter []string) ([]string, error) {
	treffer, err := listenZiele(ctx, q, stichwoerter)
	if err != nil {
		return nil, fmt.Errorf("schlagwort-vorschlag aus stichwörtern: %w", err)
	}
	var woerter []string
	for _, ziel := range treffer {
		woerter = append(woerter, ziel)
	}
	return alphabetischOhneDoppelte(woerter), nil
}

// Normdatenbegriff ist ein Schlagwort der Gemeinsamen Normdatei aus einem DNB-Satz (MARC 600
// bis 651 mit $2 gnd; inventur.MetadatenErgebnis.Normdaten, docs/OFFEN.md 4.25). Formen sind
// die Schreibweisen, unter denen er in der Liste stehen kann; Anzeige ist die, in der er als
// neues Wort angeboten wird. Die DNB schreibt eine Person „Kafka, Franz", die Littera-Liste
// „Kafka <Franz>" — im Katalogisat vom Juni 2026 stehen 2.059 Wörter in dieser Form und eines
// mit Komma; ein neues Wort in der DNB-Form stünde als zweite Person daneben.
type Normdatenbegriff struct {
	Anzeige string
	Formen  []string
}

// SchlagworteAusNormdaten gleicht die Normdatei-Begriffe eines DNB-Satzes gegen die eigene Liste
// ab (entschieden am 30.09.2026, docs/OFFEN.md 4.25). Steht ein Begriff in einer seiner Formen in
// der Liste — als Wort mit Titeln oder als Verweis darauf —, kommt er zu vorhanden, aufgelöst zu
// seinem Ziel wie beim Stichwort-Vorschlag. Sonst kommt er zu neu, in der Anzeige-Form: Die
// Normdatei ist ein festes Vokabular ohne Werbewörter, anders als die Verlagswörter, die nur
// über die Liste kommen (gemessen am 30.09.2026 an 56 DNB-Sätzen des Katalogisats: 86 Begriffe,
// 56 davon schon in der Liste, die übrigen wie „Judo", „Oslo", „Schulstress"). Ein Wort, das zu
// lang für die Liste wäre, wird nicht angeboten. Beide Antworten sind alphabetisch, ohne
// Doppelte und nie nil; eingetragen wird nur, was jemand anklickt.
func SchlagworteAusNormdaten(ctx context.Context, q DBQueryer, begriffe []Normdatenbegriff) (vorhanden, neu []string, err error) {
	var formen []string
	for _, b := range begriffe {
		formen = append(formen, b.Formen...)
	}
	treffer, err := listenZiele(ctx, q, formen)
	if err != nil {
		return nil, nil, fmt.Errorf("schlagwort-vorschlag aus normdaten: %w", err)
	}
	var neuRoh []string
	for _, b := range begriffe {
		ziel := ""
		for _, form := range b.Formen {
			if z, da := treffer[schlagwortNormalform(form)]; da {
				ziel = z
				break
			}
		}
		if ziel != "" {
			vorhanden = append(vorhanden, ziel)
			continue
		}
		if wort := schlagwortNormalform(b.Anzeige); wort != "" && utf8.RuneCountInString(wort) <= SchlagwortMaxZeichen {
			neuRoh = append(neuRoh, wort)
		}
	}
	vorhanden = alphabetischOhneDoppelte(vorhanden)
	schonDa := make(map[string]bool, len(vorhanden))
	for _, w := range vorhanden {
		schonDa[strings.ToLower(w)] = true
	}
	for _, w := range alphabetischOhneDoppelte(neuRoh) {
		if !schonDa[strings.ToLower(w)] {
			neu = append(neu, w)
		}
	}
	if neu == nil {
		neu = []string{}
	}
	return vorhanden, neu, nil
}

// listenZiele sucht die Kandidaten in der eigenen Liste und liefert je getroffenem Kandidaten
// (in schlagwortNormalform) das Wort, das vorgeschlagen wird: ein Wort mit Titeln oder das Ziel
// eines Verweises darauf. Das ganze Wort, ohne Rücksicht auf Groß- und Kleinschreibung — der
// Vergleich geschieht in der Datenbank mit lower(), wie der eindeutige Index, damit Go und SQL
// nicht verschieden kleinschreiben.
func listenZiele(ctx context.Context, q DBQueryer, roh []string) (map[string]string, error) {
	kandidaten := make([]string, 0, len(roh))
	for _, r := range roh {
		if wort := schlagwortNormalform(r); wort != "" {
			kandidaten = append(kandidaten, wort)
		}
		if len(kandidaten) == schlagwortStichwoerterMax {
			break
		}
	}
	treffer := map[string]string{}
	if len(kandidaten) == 0 {
		return treffer, nil
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT k, ziel.wort
		FROM unnest($1::text[]) AS k
		JOIN schlagworte s ON lower(s.wort) = lower(k)
		JOIN schlagworte ziel ON ziel.id = coalesce(s.verweis_auf, s.id)
		WHERE EXISTS (SELECT 1 FROM titel_schlagworte ts WHERE ts.schlagwort_id = ziel.id)`, kandidaten)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kandidat, ziel string
		if err := rows.Scan(&kandidat, &ziel); err != nil {
			return nil, err
		}
		treffer[kandidat] = ziel
	}
	return treffer, rows.Err()
}

// SchlagwortVorschlagAusDNB ist der ganze Schlagwort-Vorschlag zu einem DNB-Satz: liste sind die
// Wörter der eigenen Liste, die der Satz als Gattung, Verlagswort oder Normdatei-Schlagwort nennt
// (SchlagworteAusStichwoertern, SchlagworteAusNormdaten), neu die Normdatei-Wörter, die die Liste
// noch nicht kennt. Die eine Regel für beide Türen: das Anlegen beim Bestellen (POST
// /api/buecher/aus-isbn) und den Vorschlag im Buchformular und beim Nachbestellen (GET
// /api/schlagworte/dnb-vorschlag, entschieden am 30.09.2026). Beide Listen nie nil.
func SchlagwortVorschlagAusDNB(ctx context.Context, q DBQueryer, stichwoerter []string, normdaten []Normdatenbegriff) (liste, neu []string, err error) {
	ausStichwoertern, err := SchlagworteAusStichwoertern(ctx, q, stichwoerter)
	if err != nil {
		return nil, nil, err
	}
	ausNormdaten, neu, err := SchlagworteAusNormdaten(ctx, q, normdaten)
	if err != nil {
		return nil, nil, err
	}
	return SchlagworteZusammen(ausStichwoertern, ausNormdaten), neu, nil
}

// SchlagworteZusammen führt Vorschlagslisten zu einer zusammen: alphabetisch, ohne Doppelte.
func SchlagworteZusammen(listen ...[]string) []string {
	var alle []string
	for _, l := range listen {
		alle = append(alle, l...)
	}
	return alphabetischOhneDoppelte(alle)
}

// alphabetischOhneDoppelte sortiert ohne Rücksicht auf Groß- und Kleinschreibung und lässt
// Doppelte fallen; das erste gewinnt. Nie nil.
func alphabetischOhneDoppelte(woerter []string) []string {
	aus := []string{}
	gesehen := map[string]bool{}
	for _, w := range woerter {
		if k := strings.ToLower(w); !gesehen[k] {
			gesehen[k] = true
			aus = append(aus, w)
		}
	}
	sort.SliceStable(aus, func(i, j int) bool { return strings.ToLower(aus[i]) < strings.ToLower(aus[j]) })
	return aus
}
