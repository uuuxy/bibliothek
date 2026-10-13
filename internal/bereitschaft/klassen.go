package bereitschaft

import "bibliothek/internal/ausweis"

// TrageKlassenEin vergleicht die Klassen mit aktiven Schülern mit den Zuordnungen der
// Klassenleitungen und mit den Bücherlisten und trägt in die Lage ein, was nicht
// zusammenpasst. Die drei Listen hängen nur am gleichen Text; reißt er, meldet nichts einen
// Fehler, und die Mahnliste einer Klasse erreicht niemanden. Wer den Bestand nicht lesen
// konnte, ruft die Funktion nicht: Die Felder bleiben nil, und die Prüfung meldet „nicht
// erhoben".
func (l *Lage) TrageKlassenEin(schueler, zuordnungen, buecherlisten []string) {
	l.KlassenOhneLehrkraft = fehlendeEintraege(mitKlassenleitung(schueler), zuordnungen)
	l.VerwaisteZuordnungen = fehlendeEintraege(zuordnungen, schueler)
	l.VerwaisteBuecherliste = fehlendeEintraege(buecherlisten, schueler)
}

// mitKlassenleitung lässt die Klassen übrig, für die die Schule eine Klassenleitung hat: bis
// Jahrgang 10. Die Oberstufe (ET, 12T, 13T) hat keine, und ein Name ohne Ziffer („ABG") ist
// ein Sonderwert.
func mitKlassenleitung(klassen []string) []string {
	mit := []string{}
	for _, k := range klassen {
		if _, jahrgang, lesbar := ausweis.AblaufJahrgang(k); lesbar {
			if jahrgang <= ausweis.AbschlussMittelstufe {
				mit = append(mit, k)
			}
			continue
		}
		// Eine Zahl außerhalb 1 bis 13 („70R1") bleibt gemeldet: Ihre Mahnliste erreicht niemanden.
		if k != "" && k[0] >= '0' && k[0] <= '9' {
			mit = append(mit, k)
		}
	}
	return mit
}

// fehlendeEintraege liefert alle Werte aus menge, die in referenz fehlen. Das Ergebnis ist nie
// nil, damit „geprüft und leer" von „nicht erhoben" (nil) unterscheidbar bleibt.
func fehlendeEintraege(menge, referenz []string) []string {
	bekannt := make(map[string]bool, len(referenz))
	for _, r := range referenz {
		bekannt[r] = true
	}
	fehlt := []string{}
	for _, m := range menge {
		if !bekannt[m] {
			fehlt = append(fehlt, m)
		}
	}
	return fehlt
}
