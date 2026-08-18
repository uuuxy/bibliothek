package service

// Littera druckt auf die Buchetiketten die Mediennummer als Klartext, codiert im
// Strichcode darunter aber eine EAN-13. Gemessen am 18.08.2026 an zwei Büchern
// der Philipp-Reis-Schule (Aufdruck 58968 → Scan 5896800039556, Aufdruck 124117
// → Scan 1241170039561) ist der Aufbau:
//
//	[Mediennummer, rechts mit Nullen auf 8 Stellen aufgefüllt]
//	[Littera-Bibliotheksnummer, 3 Stellen (hier 395)]
//	[Stellenzahl der Mediennummer, 1 Stelle]
//	[EAN-13-Prüfziffer]
//
// Die Stellenzahl am Ende macht die Rückrechnung eindeutig: 58968 und 589680
// füllen beide zu "58968000" auf, tragen aber 5 bzw. 6 als vorletzte Ziffer.
// Ohne diese Übersetzung liefe jeder Scan eines Littera-Etiketts ins Leere,
// obwohl das Buch unter seiner Mediennummer längst im System steht.
//
// Die Bibliotheksnummer wird bewusst NICHT geprüft: Sie ist je Schule anders,
// und der eigentliche Wächter gegen Fehldeutungen (z. B. eine gescannte
// Verlags-ISBN) ist die Kombination aus Null-Polsterung, Stellenzahl und
// Prüfziffer — plus der Nachschlag in der Datenbank, der nur bei einem
// tatsächlich existierenden Exemplar eine Aktion auslöst.
func dekodiereLitteraEtikett(scan string) (nummer string, ok bool) {
	if len(scan) != 13 {
		return "", false
	}
	for _, c := range scan {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	if !ean13PruefzifferStimmt(scan) {
		return "", false
	}

	payload := scan[:12]
	laenge := int(payload[11] - '0')
	if laenge < 1 || laenge > 8 {
		return "", false
	}
	nummer = payload[:laenge]
	if nummer[0] == '0' {
		return "", false
	}
	// Zwischen Mediennummer und Bibliotheksnummer steht ausschließlich Polsterung.
	for i := laenge; i < 8; i++ {
		if payload[i] != '0' {
			return "", false
		}
	}
	return nummer, true
}

// ean13PruefzifferStimmt prüft die Standard-EAN-13-Prüfziffer (Gewichte 1/3).
func ean13PruefzifferStimmt(scan string) bool {
	summe := 0
	for i := 0; i < 12; i++ {
		ziffer := int(scan[i] - '0')
		if i%2 == 1 {
			ziffer *= 3
		}
		summe += ziffer
	}
	erwartet := (10 - summe%10) % 10
	return int(scan[12]-'0') == erwartet
}
