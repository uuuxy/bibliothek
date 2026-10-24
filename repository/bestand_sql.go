package repository

// Der Bestand eines Titels: eine Definition für alle Abfragen, die ihn nennen. Ohne die Zahlen
// stünde ein Titel ohne Exemplar in der Trefferliste wie jeder andere.
//
// Die beiden Prädikate sind wörtlich die der Klassenbuch-Abfrage
// (repository/klassensaetze.go), und das ist der Zweck dieser Datei: Zwei Auslegungen von
// „verfügbar" ergäben zwei Zahlen über denselben Titel, und an der Theke entscheidet diese
// Zahl, ob jemand losläuft und im Regal sucht.
//
//   - gesamt:     im Bestand, nicht ausgesondert, nicht mehr im Zulauf
//     (bestellstatus IS NULL — ein bestelltes Buch steht noch nicht im Regal).
//   - verfuegbar: davon die ausleihbaren, die gerade niemand hat.
//   - im Zulauf:  bestellt, noch nicht eingetroffen. Ohne diese dritte Zahl sagte die
//     Trefferliste über einen Titel, dessen Exemplare alle unterwegs sind, „Keine
//     Exemplare" — und schickte den Kollegen ins Regal.
const (
	// SQLExemplarImBestand ist die Grenze von „gesamt" selbst, für Abfragen über die
	// Exemplare als `e`. Die Buchmaske zeigt diese Zahl im Feld „Aktueller Bestand", und die
	// Bestandskorrektur dahinter (TitelAbfragen.gleicheBestandAb) zählt und sondert mit derselben
	// Grenze aus: Zählte sie die bestellten Exemplare mit, hielte sie die Zahl der Maske für
	// eine Verringerung.
	SQLExemplarImBestand = `e.ist_ausgesondert = false AND e.bestellstatus IS NULL`

	// SQLBestandGesamt zählt die Exemplare eines Titels, die im Bestand stehen.
	// Einzusetzen in eine SELECT-Liste; der Titel muss als `b` gebunden sein.
	SQLBestandGesamt = `(SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = b.id AND ` + SQLExemplarImBestand + `)`

	// SQLBestandVerfuegbar zählt davon die, die jemand sofort mitnehmen könnte.
	SQLBestandVerfuegbar = `(SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = b.id AND e.ist_ausgesondert = false AND e.ist_ausleihbar = true
		  AND NOT EXISTS (SELECT 1 FROM ausleihen a
		                  WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL))`

	// SQLBestandImZulauf zählt die bestellten, noch nicht eingetroffenen Exemplare.
	// Dieselbe Grenze wie SQLBestandGesamt, nur die andere Seite von bestellstatus: Beide
	// zusammen sind alle nicht ausgesonderten Exemplare (sqlTitelHatExemplar).
	SQLBestandImZulauf = `(SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = b.id AND e.ist_ausgesondert = false AND e.bestellstatus IS NOT NULL)`
)

// sqlFilterImZulauf ist dieselbe Grenze als COUNT-FILTER für Abfragen, die die Exemplare
// als `e` joinen (Katalogliste, Klassenbücher) statt je Titel zu zählen.
const sqlFilterImZulauf = `COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NOT NULL)`

// sqlTitelHatExemplar sagt, ob ein Titel überhaupt ein Exemplar hat, das nicht ausgesondert
// ist: im Regal, verliehen oder im Zulauf. Ein Titel ohne ein solches Exemplar erscheint in
// keinem Katalog. Der Zulauf zählt als vorhanden: Die Bücher kommen, und wer die Bestellung
// sieht, soll den Titel finden.
//
// Der Titel bleibt in der Tabelle: Er verschwindet aus der Sicht, nicht aus dem Bestand, sonst
// legt ihn jemand ein zweites Mal an. Die Verwaltung erreicht ihn über die Sicht „Ohne
// Exemplare" derselben Liste (TitelAbfragen.LadeTitelListe) und die Bestellliste.
//
// titelAlias ist der Alias der Titeltabelle in der umgebenden Abfrage (`b`, `bt`).
func sqlTitelHatExemplar(titelAlias string) string {
	return `EXISTS (SELECT 1 FROM buecher_exemplare hx
		WHERE hx.titel_id = ` + titelAlias + `.id AND hx.ist_ausgesondert = false)`
}
