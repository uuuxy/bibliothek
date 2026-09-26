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
//   - im Zulauf:  bestellt, noch nicht eingetroffen. Ohne diese dritte Zahl sagte die
//     Trefferliste über einen Titel, dessen Exemplare alle unterwegs sind, „Keine
//     Exemplare" — und schickte den Kollegen ins Regal (docs/OFFEN.md 5.5).
const (
	// SQLBestandSelect projiziert die drei von SQLBestandLateral erzeugten Spalten
	// (gesamt, verfuegbar, im_zulauf). Die JOIN-Ergebnisse müssen als `bestand`
	// verfügbar sein.
	SQLBestandSelect = `COALESCE(bestand.gesamt, 0), COALESCE(bestand.verfuegbar, 0), COALESCE(bestand.im_zulauf, 0)`
)

// SQLBestandLateral bündelt die Zählung von gesamt, verfügbar und im_zulauf in einem einzigen
// LEFT JOIN LATERAL auf die Exemplar-Tabelle. Das vermeidet drei teure, korrelierte Sub-Selects
// je Titel-Zeile in Listen wie der Buch-Suche.
func SQLBestandLateral(titelAlias string) string {
	return `LEFT JOIN LATERAL (
		SELECT
			COUNT(*) FILTER (WHERE e.bestellstatus IS NULL) AS gesamt,
			COUNT(*) FILTER (WHERE e.ist_ausleihbar = true AND
				NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)) AS verfuegbar,
			COUNT(*) FILTER (WHERE e.bestellstatus IS NOT NULL) AS im_zulauf
		FROM buecher_exemplare e
		WHERE e.titel_id = ` + titelAlias + `.id AND e.ist_ausgesondert = false
	) bestand ON true`
}

// SQLFilterImZulauf ist dieselbe Grenze als COUNT-FILTER für Abfragen, die die Exemplare
// als `e` joinen (Katalogliste, Klassenbücher) statt je Titel zu zählen.
const SQLFilterImZulauf = `COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NOT NULL)`

// SQLTitelHatExemplar sagt, ob ein Titel überhaupt ein Exemplar hat, das nicht
// ausgesondert ist — im Regal, verliehen oder im Zulauf. Titel ohne ein solches Exemplar
// erscheinen seit dem 22.09.2026 in keinem Katalog mehr (Antwort der Schule auf Punkt 4 des
// Protokolls: „Ein Titel/Werk ohne (verliehene oder verfügbare) Exemplare im Bestand sollte
// aus unserer Sicht nicht im Katalog erscheinen"). Der Zulauf zählt als vorhanden: Die
// Bücher kommen, und wer die Bestellung sieht, soll den Titel finden.
//
// Der Titel bleibt in der Tabelle — er verschwindet aus der Sicht, nicht aus dem Bestand,
// sonst legt ihn jemand ein zweites Mal an. Die Verwaltung erreicht ihn über die Sicht
// „Ohne Exemplare" derselben Liste (inventur.ListBooks) und die Bestellliste.
//
// titelAlias ist der Alias der Titeltabelle in der umgebenden Abfrage (`b`, `bt`).
func SQLTitelHatExemplar(titelAlias string) string {
	return `EXISTS (SELECT 1 FROM buecher_exemplare hx
		WHERE hx.titel_id = ` + titelAlias + `.id AND hx.ist_ausgesondert = false)`
}

// scanBookTitleMitBestand scannt die Standard-Spaltenliste und danach die beiden
// Bestandszahlen. Getrennt von scanBookTitle, weil die übrigen Abfragen sie nicht
// mitliefern — und ein Titel mit nil-Bestand sagt „nicht gezählt", nicht „keine da".
func scanBookTitleMitBestand(row Scanner, zusatz ...any) (*BookTitle, error) {
	var gesamt, verfuegbar, imZulauf int
	t, err := scanBookTitleMitZusatz(row, append([]any{&gesamt, &verfuegbar, &imZulauf}, zusatz...)...)
	if err != nil {
		return nil, err
	}
	t.Bestand = &gesamt
	t.Verfuegbar = &verfuegbar
	t.ImZulauf = &imZulauf
	return t, nil
}
