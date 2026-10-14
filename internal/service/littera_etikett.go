package service

// dekodiereLitteraEtikett rechnet aus dem Strichcode eines Etiketts von Littera die
// Mediennummer zurück. Littera druckt sie als Klartext, codiert im Strichcode darunter aber
// eine EAN-13. Gemessen an zwei Büchern der Philipp-Reis-Schule (Aufdruck 58968, Scan
// 5896800039556; Aufdruck 124117, Scan 1241170039561) ist der Aufbau:
//
//	[Mediennummer, rechts mit Nullen auf 8 Stellen aufgefüllt]
//	[Littera-Bibliotheksnummer, 3 Stellen (hier 395)]
//	[Stellenzahl der Mediennummer, 1 Stelle]
//	[EAN-13-Prüfziffer]
//
// Die Stellenzahl am Ende macht die Rückrechnung eindeutig: 58968 und 589680 füllen beide zu
// "58968000" auf, tragen aber 5 und 6 als vorletzte Ziffer.
//
// Die Bibliotheksnummer wird nicht geprüft, weil sie je Schule anders ist. Gegen Fehldeutungen
// (etwa eine gescannte ISBN des Verlags) schützen Polsterung, Stellenzahl und Prüfziffer und
// dazu der Nachschlag in der Datenbank, der nur bei einem vorhandenen Exemplar etwas auslöst.
func dekodiereLitteraEtikett(scan string) (nummer string, ok bool) {
	if len(scan) != 13 {
		return "", false
	}
	for _, c := range scan {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	if !istPruefzifferRichtig(scan) {
		return "", false
	}

	daten := scan[:12]
	laenge := int(daten[11] - '0')
	if laenge < 1 || laenge > 8 {
		return "", false
	}
	nummer = daten[:laenge]
	if nummer[0] == '0' {
		return "", false
	}
	// Zwischen Mediennummer und Bibliotheksnummer steht ausschließlich Polsterung.
	for i := laenge; i < 8; i++ {
		if daten[i] != '0' {
			return "", false
		}
	}
	return nummer, true
}

// istPruefzifferRichtig prüft die Prüfziffer einer EAN-13 (Gewichte 1 und 3).
func istPruefzifferRichtig(scan string) bool {
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
