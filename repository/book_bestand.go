package repository

// Der Bestand eines Titels — EINE Definition für alle Abfragen, die ihn nennen.
//
// Anlass: Protokoll des Medienzentrums vom 16.09.2026, Punkt 4 („Bücher, zu denen es
// keine Exemplare gibt, tauchen in der Trefferliste auf"). Die Theke zeigte solche Titel
// ohne jeden Hinweis; der Katalog schrieb immerhin „Keine Exemplare".
//
// Die beiden Prädikate sind WÖRTLICH die der Klassenbuch-Abfrage
// (inventur/datenbank_klassen.go), und das ist der Zweck dieser Datei: Zwei Auslegungen
// von „verfügbar" ergäben zwei Zahlen über denselben Titel, und an der Theke entscheidet
// diese Zahl, ob jemand losläuft und im Regal sucht.
//
//   - gesamt:     im Bestand, nicht ausgesondert, nicht mehr im Zulauf
//     (bestellstatus IS NULL — ein bestelltes Buch steht noch nicht im Regal).
//   - verfuegbar: davon die ausleihbaren, die gerade niemand hat.
const (
	// SQLBestandGesamt zählt die Exemplare eines Titels, die im Bestand stehen.
	// Einzusetzen in eine SELECT-Liste; der Titel muss als `b` gebunden sein.
	SQLBestandGesamt = `(SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = b.id AND e.ist_ausgesondert = false AND e.bestellstatus IS NULL)`

	// SQLBestandVerfuegbar zählt davon die, die jemand sofort mitnehmen könnte.
	SQLBestandVerfuegbar = `(SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = b.id AND e.ist_ausgesondert = false AND e.ist_ausleihbar = true
		  AND NOT EXISTS (SELECT 1 FROM ausleihen a
		                  WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL))`
)

// scanBookTitleMitBestand scannt die Standard-Spaltenliste und danach die beiden
// Bestandszahlen. Getrennt von scanBookTitle, weil die übrigen Abfragen sie nicht
// mitliefern — und ein Titel mit nil-Bestand sagt „nicht gezählt", nicht „keine da".
func scanBookTitleMitBestand(row Scanner, zusatz ...any) (*BookTitle, error) {
	var gesamt, verfuegbar int
	t, err := scanBookTitleMitZusatz(row, append([]any{&gesamt, &verfuegbar}, zusatz...)...)
	if err != nil {
		return nil, err
	}
	t.Bestand = &gesamt
	t.Verfuegbar = &verfuegbar
	return t, nil
}
