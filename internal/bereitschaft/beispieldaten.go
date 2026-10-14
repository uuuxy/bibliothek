package bereitschaft

import (
	"strconv"
	"strings"
)

func pruefeDemodaten(l Lage) Befund {
	b := Befund{Bereich: "Demo-Daten"}
	if l.DemoSchueler == 0 && l.DemoExemplare == 0 {
		b.Stufe = StufeOK
		b.Befund = "Keine Demo-Datensätze im Bestand."
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = strconv.Itoa(l.DemoSchueler) + " Demo-Schüler und " +
		strconv.Itoa(l.DemoExemplare) + " Demo-Exemplare im Bestand."
	b.Folge = "Statistik, Mahnwesen und Bestellbedarf mischen echte Zahlen mit Fiktion. " +
		"Demo-Eltern-Adressen enden auf example.invalid, ein Mahnlauf erreicht sie nie."
	// Nicht der Block am Anfang von seed_demo.sql: Der bricht auf einem benutzten System an
	// Schäden ab (RESTRICT) und lässt Bescheide und Inventur-Verluste ohne Bezug stehen.
	b.Abhilfe = "Vor dem Echtstart scripts/entferne_demo_daten.sql ausführen: erst ohne Schalter " +
		"(Vorschau, zeigt jede Verflechtung mit echten Daten), dann mit -v ausfuehren=ja. " +
		"Der Aufruf steht im Kopf der Datei."
	return b
}

// pruefeBeispielLieferanten: Vom 30.05. bis zum 07.09.2026 legte der erste Programmstart
// drei Beispiel-Lieferanten an — damals gewollt, damit das Bestellwesen ohne Vorarbeit
// bedienbar war. Das Problem war nie die Absicht, sondern der Ort: Startdaten im
// Boot-Pfad sind auf einer echten Anlage von echten Händlern nicht zu unterscheiden. Wer
// einen davon im Bestellformular wählte, bekam eine Historie mit „gesendet“ — und keine
// Bücher. Kritisch, nicht Warnung: Der Schaden ist still und trifft eine echte Bestellung.
func pruefeBeispielLieferanten(l Lage) Befund {
	b := Befund{Bereich: "Lieferanten"}
	switch {
	case l.BeispielLieferanten == nil:
		b.Stufe = StufeWarnung
		b.Befund = "Lieferantenliste nicht lesbar — nicht geprüft."
		b.Folge = "Ob noch Beispiel-Lieferanten aus den alten Startdaten im Bestand stehen, ist offen."
		b.Abhilfe = abhilfeDBNeuLaden
	case len(l.BeispielLieferanten) > 0:
		b.Stufe = StufeKritisch
		b.Befund = "Beispiel-Lieferanten aus den alten Startdaten im Bestand: " +
			strings.Join(l.BeispielLieferanten, ", ") + "."
		b.Folge = "Eine Bestellung an diesen Eintrag geht wirklich an die Beispiel-Adresse, mit einer " +
			"Kundennummer, die es nicht gibt. Die Historie meldet „gesendet“, die Bücher kommen nie."
		b.Abhilfe = "Einstellungen → Lieferanten: Eintrag löschen oder mit den echten Daten des Händlers überschreiben."
	default:
		b.Stufe = StufeOK
		b.Befund = "Keine Beispiel-Lieferanten im Bestand."
	}
	return b
}
