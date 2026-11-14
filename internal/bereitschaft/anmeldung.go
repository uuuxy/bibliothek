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
	case l.IMAPHost == "" && l.LDAPURL == "":
		b.Stufe = StufeKritisch
		b.Befund = "IMAP_HOST und LDAP_URL sind leer."
		b.Folge = "Niemand kann sich anmelden."
		b.Abhilfe = "Den IMAP-Host der Schule oder den Verzeichnisdienst (LDAP_URL) eintragen."
	case l.LDAPURL == "":
		b.Stufe = StufeOK
		b.Befund = "Anmeldung läuft über " + l.IMAPHost + "."
	case l.IMAPHost == "":
		b.Stufe = StufeOK
		b.Befund = "Anmeldung läuft über den Verzeichnisdienst " + l.LDAPURL + "; Konten mit IMAP kommen nicht hinein."
	default:
		b.Stufe = StufeOK
		b.Befund = "Anmeldung läuft je Konto über " + l.IMAPHost + " oder den Verzeichnisdienst " + l.LDAPURL + "."
	}
	return b
}

// pruefeSelbstanmeldung meldet eine leere SELBSTANMELDUNG_DOMAIN. Ohne sie wird eine
// Lehrkraft ohne Konto auch mit richtigem Passwort abgewiesen und nach fünf Versuchen
// gesperrt, was wie ein Tippfehler aussieht; der Start schreibt es nur ins Log
// (auth.SelbstanmeldungStatus). Sie ist auch die Domain, die der Sammelversand an einen Namen
// ohne „@" hängt. Eine Warnung, weil die Schule die Konten auch von Hand anlegen und volle
// Adressen tippen kann und es nur wissen soll.
func pruefeSelbstanmeldung(l Lage) Befund {
	b := Befund{Bereich: "Selbstanmeldung Kollegium"}
	if l.SelbstanmeldeDomain == "" {
		b.Stufe = StufeWarnung
		b.Befund = "SELBSTANMELDUNG_DOMAIN ist leer — Lehrkräfte können sich nicht selbst anmelden."
		b.Folge = "Eine Lehrkraft ohne Konto bekommt trotz richtigem Passwort „Anmeldung fehlgeschlagen“; jedes Kollegiums-Konto muss unter Benutzer & Rechte von Hand angelegt werden. Im Sammelversand an die Klassenleitungen braucht eine abweichende Adresse die volle Domain."
		b.Abhilfe = "Die Schul-Maildomain eintragen (z. B. SELBSTANMELDUNG_DOMAIN=philipp-reis-schule.de). Neue Konten entstehen dann inaktiv und werden unter Benutzer & Rechte freigeschaltet."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = "Postfächer @" + l.SelbstanmeldeDomain + " melden sich selbst an; neue Konten warten inaktiv auf die Freischaltung."
	return b
}
