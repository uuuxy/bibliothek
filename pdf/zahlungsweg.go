package pdf

import (
	"fmt"
	"strings"

	"bibliothek/pkg/betrag"
)

// Wohin gezahlt wird, in einem Wortlaut für die Rechnung (`rechnung.go`).
//
// Die Arbeitshilfe zum Erlass vom 17.12.2014 (Az. 674.100.002-00178) sagt für die Lernmittel des
// Landes: „Es darf kein Schulgirokonto, kein anderes Bankkonto und keine Bargeldannahme
// vorgesehen werden."
//
// Zwei Töpfe, zwei Wege. Das Vokabular ist das der Datenbank (`land` / `schultraeger`), die
// Zuordnung dieselbe wie beim Bescheid: Lernmittel gehen an das Land, alles andere an den
// Schulträger. Für das Land steht der Weg fest und kommt aus derselben Einstellung wie im
// Bescheid (`repository.BescheidAngaben`); eine zweite Kontoangabe im selben Haus wäre eine
// zweite Wahrheit über dieselbe Zahlung.
//
// Für den Schulträger ist der Zahlungsweg nicht entschieden: Dort ist „bar gegen Quittung"
// ausdrücklich möglich, aber niemand hat es beschlossen. Solange das so ist, nennt der Brief kein
// Konto und keine Kasse, sondern sagt, dass die Angabe fehlt, mit derselben Zeile, die für die
// Rechnung des Kreises beschlossen ist. Ein Brief, der sich einen Zahlungsweg ausdenkt, schickt
// Geld an die falsche Stelle.
const (
	// ZahlungswegLandSatz nennt die Zahlstelle; darunter stehen die Zeilen der Bankverbindung.
	ZahlungswegLandSatz = "Bitte überweisen Sie den Betrag auf das Konto %s:"
	// ZahlungswegTraegerSatz bleibt offen, siehe oben.
	ZahlungswegTraegerSatz = "Den Zahlungsweg für Bücher der Schülerbücherei nennt Ihnen die Schule."
	// ZahlungswegTraegerFehlt ist die beschlossene Zeile, solange die Bankverbindung des
	// Schulträgers nicht hinterlegt ist.
	ZahlungswegTraegerFehlt = "(Bankverbindung des Schulträgers nicht hinterlegt)"
)

// Zahlungsangaben sind die Werte aus den Einstellungen, mit denen der Brief das Konto des
// Landes benennt. Sie kommen aus repository.BescheidAngaben und tragen deren Vorbelegung
// aus dem Musterschreiben.
type Zahlungsangaben struct {
	Zahlstelle     string
	Bankverbindung string
}

// ZahlungswegBlock ist ein Zahlungsweg, wie er im Brief steht: eine Überschrift (nur wenn ein
// Brief beide Töpfe trägt) und die Zeilen darunter.
type ZahlungswegBlock struct {
	Ueberschrift string
	Zeilen       []string
}

// ZahlungswegZeilen liefert die Zeilen für einen Topf.
func ZahlungswegZeilen(land bool, a Zahlungsangaben) []string {
	if !land {
		return []string{ZahlungswegTraegerSatz, ZahlungswegTraegerFehlt}
	}
	zeilen := []string{fmt.Sprintf(ZahlungswegLandSatz, a.Zahlstelle)}
	for _, z := range strings.Split(a.Bankverbindung, "\n") {
		if z = strings.TrimSpace(z); z != "" {
			zeilen = append(zeilen, z)
		}
	}
	return zeilen
}

// ZahlungswegBloecke teilt einen Brief auf, der Forderungen aus beiden Töpfen trägt.
//
// Das kommt vor: Die Rechnung listet alle offenen Forderungen eines Schülers, und darunter
// können ein Lernmittel des Landes und ein Buch der Schülerbücherei stehen. Zwei Töpfe in einer
// Summe auf ein Konto wären falsch, denn Geld des Landes und Geld des Trägers werden getrennt
// geführt (dieselbe Regel wie beim Bescheid: ein Brief, ein Topf; nur kann dieser Brief sich
// seine Positionen nicht aussuchen).
//
// Trägt der Brief nur einen Topf, bekommt der Block keine Überschrift: Dann geht es um den
// ganzen Betrag, und eine Teilsumme danebenzuschreiben verwirrt nur.
func ZahlungswegBloecke(betragLand, betragTraeger float64, a Zahlungsangaben) []ZahlungswegBlock {
	beide := betragLand > 0 && betragTraeger > 0
	var bloecke []ZahlungswegBlock
	// Beträge von 0,00 EUR bekommen keinen Weg: Da ist nichts zu zahlen. Einen Brief ohne jede
	// Position gibt es nicht, die Tür weist ihn vorher mit 404 ab.
	if betragLand > 0 {
		b := ZahlungswegBlock{Zeilen: ZahlungswegZeilen(true, a)}
		if beide {
			b.Ueberschrift = fmt.Sprintf("Für die Lernmittel des Landes (%s EUR):", betrag.Text(betragLand))
		}
		bloecke = append(bloecke, b)
	}
	if betragTraeger > 0 {
		b := ZahlungswegBlock{Zeilen: ZahlungswegZeilen(false, a)}
		if beide {
			b.Ueberschrift = fmt.Sprintf("Für die Bücher der Schülerbücherei (%s EUR):", betrag.Text(betragTraeger))
		}
		bloecke = append(bloecke, b)
	}
	return bloecke
}
