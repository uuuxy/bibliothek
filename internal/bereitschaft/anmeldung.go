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

// pruefeSelbstanmeldung: Ohne SELBSTANMELDUNG_DOMAIN bekommt eine Lehrkraft mit
// RICHTIGEM Passwort „invalid email or password“ und nach fünf Versuchen eine Sperre —
// eine tote Tür, die wie ein Tippfehler aussieht. Der Serverstart protokolliert den
// Zustand zwar (auth.SelbstanmeldungStatus), aber ein Log liest niemand, der gerade
// eine Lehrkraft am Telefon hat. Warnung, nicht kritisch: Die Schule KANN alle 160
// Konten von Hand anlegen — sie soll nur wissen, dass sie das gerade tut.
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
