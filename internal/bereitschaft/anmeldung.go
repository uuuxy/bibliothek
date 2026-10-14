package bereitschaft

func pruefeAnmeldung(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Anmeldung"}
	switch {
	case l.IMAPHost == "mock" && echt:
		b.Stufe = StufeKritisch
		b.Befund = "IMAP_HOST steht auf \"mock\"."
		b.Folge = "Die Anmeldung akzeptiert JEDES Passwort für jede hinterlegte E-Mail-Adresse."
		b.Abhilfe = "Den echten IMAP-Host der Schule eintragen."
	case l.IMAPHost == "mock":
		b.Stufe = StufeOK
		b.Befund = befundNichtImEchtbetrieb("Mock-Anmeldung", l.Umgebung)
	case l.IMAPHost == "":
		b.Stufe = StufeKritisch
		b.Befund = "IMAP_HOST ist leer."
		b.Folge = "Niemand kann sich anmelden."
		b.Abhilfe = "Den IMAP-Host der Schule eintragen."
	default:
		b.Stufe = StufeOK
		b.Befund = "Anmeldung läuft über " + l.IMAPHost + "."
	}
	return b
}

// pruefeSelbstanmeldung meldet eine leere SELBSTANMELDUNG_DOMAIN. Ohne sie wird eine
// Lehrkraft ohne Konto auch mit richtigem Passwort abgewiesen und nach fünf Versuchen
// gesperrt, was wie ein Tippfehler aussieht; der Start schreibt es nur ins Log
// (auth.SelbstanmeldungStatus). Eine Warnung, weil die Schule die Konten auch von Hand
// anlegen kann und es nur wissen soll.
func pruefeSelbstanmeldung(l Lage) Befund {
	b := Befund{Bereich: "Selbstanmeldung Kollegium"}
	if l.SelbstanmeldeDomain == "" {
		b.Stufe = StufeWarnung
		b.Befund = "SELBSTANMELDUNG_DOMAIN ist leer — Lehrkräfte können sich nicht selbst anmelden."
		b.Folge = "Eine Lehrkraft ohne Konto bekommt trotz richtigem Passwort „Anmeldung fehlgeschlagen“; jedes Kollegiums-Konto muss unter Benutzer & Rechte von Hand angelegt werden."
		b.Abhilfe = "Die Schul-Maildomain eintragen (z. B. SELBSTANMELDUNG_DOMAIN=philipp-reis-schule.de). Neue Konten entstehen dann inaktiv und werden unter Benutzer & Rechte freigeschaltet."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = "Postfächer @" + l.SelbstanmeldeDomain + " melden sich selbst an; neue Konten warten inaktiv auf die Freischaltung."
	return b
}
