package auth

// Die Sätze, die Anmeldung und Sperre an die Oberfläche schicken. Die Anmeldung zeigt sie
// unverändert an; sie folgen deshalb den Regeln für Meldungen (docs/STIL.md, Abschnitt 11).
const (
	// MeldungNichtAngemeldet gilt für jede Anfrage ohne gültige Anmeldung, auch an den Türen der
	// Themen, die die Angaben der Sitzung nicht finden.
	MeldungNichtAngemeldet = "Nicht angemeldet. Bitte neu anmelden."
	// MeldungZuVieleFehlversuche gilt für beide Bremsen der Anmeldung: je E-Mail und Adresse hier,
	// je Adresse am Server (api/anmelde_bremse.go).
	MeldungZuVieleFehlversuche = "Zu viele fehlgeschlagene Anmeldungen. Bitte 15 Minuten warten."
	// MeldungPruefungGestoert geht mit 503 hinaus, wenn sich die Anmeldung nicht prüfen ließ
	// (ErrPruefungGestoert).
	MeldungPruefungGestoert = "Die Anmeldung ließ sich nicht prüfen. Bitte erneut versuchen."
	// MeldungSitzungGesperrt geht mit 423 hinaus: Die Anmeldung gilt, ist nach Inaktivität aber
	// gesperrt (ErrSitzungGesperrt).
	MeldungSitzungGesperrt = "Gesperrt wegen Inaktivität. Bitte das Passwort eingeben."

	meldungEmailFehlt            = "Bitte die E-Mail-Adresse eingeben"
	meldungPasswortFehlt         = "Bitte das Passwort eingeben"
	meldungAnmeldungFalsch       = "E-Mail-Adresse oder Passwort stimmt nicht"
	meldungPasswortFalsch        = "Das Passwort stimmt nicht"
	meldungAnmeldedienstGestoert = "Die Anmeldung ist gerade gestört. Bitte später erneut versuchen."
	meldungZugangBeantragt       = "Zugang beantragt. Die Bibliothek muss ihn noch freischalten."
	meldungKontoDeaktiviert      = "Konto deaktiviert. Bitte an die Bibliothek wenden."
)
