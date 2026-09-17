package api

// Die Abschnitte eines Bestandsnachweises — EIN Ort für Reihenfolge und Überschriften.
//
// Zugangs- und Abgangsbuch zeigen ihre Zeilen getrennt nach Topf: Die Finanzen von Land und
// Schulträger werden getrennt geführt (docs/mittel_konzept.md 1.1/1.2). Beide Bücher gibt es
// zweimal — auf dem Bildschirm und als Blatt.
//
// Genau daran ist das Abgangsbuch beim ersten Wurf auseinandergelaufen: Das Blatt schrieb
// „Lernmittelfreiheit (Land)" (mittelBeschriftung), die Oberfläche „Lernmittel (Land)" — zwei
// Wörter für denselben Topf auf demselben Nachweis. Seit dem 17.09.2026 baut deshalb der
// SERVER die Abschnitte samt Überschrift, und der Ausdruck und der Bildschirm zeigen dieselbe
// Liste.

// Abschnitt ist ein Topf-Block eines Bestandsnachweises.
type Abschnitt[T any] struct {
	// Topf ist der gespeicherte Wert ('land', 'schultraeger', '' = ohne Zuordnung).
	Topf string `json:"topf"`
	// Titel ist die Überschrift, wie ein Mensch sie liest.
	Titel  string `json:"titel"`
	Zeilen []T    `json:"zeilen"`
}

// abschnitteAus gruppiert Zeilen nach Topf — in der Reihenfolge, die in diesem Programm
// überall gilt (Lernmittel zuerst, dann Schülerbücherei, zuletzt ohne Zuordnung).
//
// Die beiden echten Töpfe stehen IMMER da, auch leer: Auf dem Bildschirm sagt „kein Zugang in
// diesem Zeitraum" etwas, ein fehlender Abschnitt sähe nach einem Filterfehler aus. Der
// Ausdruck überspringt die leeren — ein Blatt zum Abheften soll keine Überschriften ohne
// Inhalt tragen.
//
// „Ohne Zuordnung" erscheint nur, wenn es solche Zeilen gibt. Beim Abgangsbuch kann es sie
// nicht geben (jeder Titel ist entweder Lernmittel oder nicht), beim Zugangsbuch schon: Ein
// Exemplar, das ohne Bestellung entstanden ist, trägt keinen Beleg darüber, aus welchem Geld
// es bezahlt wurde — und geraten wird das nicht.
func abschnitteAus[T any](zeilen []T, topfVon func(T) string) []Abschnitt[T] {
	nach := map[string][]T{}
	for _, z := range zeilen {
		topf := topfVon(z)
		nach[topf] = append(nach[topf], z)
	}
	aus := make([]Abschnitt[T], 0, len(mittelReihenfolge))
	for _, topf := range mittelReihenfolge {
		if topf == "" && len(nach[topf]) == 0 {
			continue
		}
		zeilenDesTopfs := nach[topf]
		if zeilenDesTopfs == nil {
			zeilenDesTopfs = []T{}
		}
		aus = append(aus, Abschnitt[T]{Topf: topf, Titel: mittelBeschriftung(topf), Zeilen: zeilenDesTopfs})
	}
	return aus
}
