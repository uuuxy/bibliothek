package bereitschaft

import (
	"strings"

	"bibliothek/internal/pkg/bestelllink"
)

func pruefeBestelllink(l Lage) Befund {
	b := Befund{Bereich: "Bestell-Bestätigungslink"}
	// Geprüft wird der Link, wie er in die Mail kommt: Ohne Schema setzt bestelllink.Adresse
	// https davor, ein eingetragenes http:// lässt es stehen.
	if strings.HasPrefix(bestelllink.Adresse(l.OeffentlicheAdresse, "token"), "http://") {
		b.Stufe = StufeWarnung
		b.Befund = "Links verweisen unverschlüsselt auf " + l.OeffentlicheAdresse
		b.Folge = "Der Bestätigungslink trägt sein Geheimnis in der Adresse. Öffnet der Lieferant ihn, " +
			"kann es unverschlüsselt durchs Netz gehen, und wer mitliest, kann die Bestellung ansehen " +
			"und an seiner Stelle bestätigen."
		b.Abhilfe = "Einstellungen → Erreichbarkeit & Alarme: die öffentliche Adresse mit https:// eintragen"
		return b
	}
	if l.OeffentlicheAdresse != "" {
		b.Stufe = StufeOK
		b.Befund = "Links verweisen auf " + l.OeffentlicheAdresse
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = "Keine öffentliche Adresse hinterlegt"
	b.Folge = "Bestellmails gehen raus, aber ohne den Bestätigungslink, " +
		"um dessentwillen es sie gibt. Der Lieferant kann nicht bestätigen."
	b.Abhilfe = "Einstellungen → Erreichbarkeit & Alarme: öffentliche Adresse der Anwendung eintragen"
	return b
}

func pruefeMailversand(l Lage) Befund {
	b := Befund{Bereich: "Mailversand im Mahnwesen"}
	// Die Ausnahmeschalter zuerst: Sie schwächen den Versand auch dann, wenn sonst alles
	// eingerichtet ist, und gerade dann sähe der Bereich ohne sie „ok" aus.
	var schalter []string
	if l.SMTPKlartextErlaubt {
		schalter = append(schalter, "SMTP_ALLOW_PLAINTEXT")
	}
	if l.SMTPUnsicheresTLSErlaubt {
		schalter = append(schalter, "SMTP_ALLOW_INSECURE_TLS")
	}
	if len(schalter) > 0 {
		b.Stufe = StufeWarnung
		b.Befund = "Der Mailversand läuft mit abgeschwächter Absicherung: " +
			strings.Join(schalter, " und ") + " steht auf true"
		b.Folge = "SMTP_ALLOW_PLAINTEXT schickt Mails ohne Verschlüsselung, wenn der Server kein STARTTLS " +
			"anbietet; SMTP_ALLOW_INSECURE_TLS prüft das Zertifikat des Mailservers nicht. Mahnungen " +
			"tragen Schülernamen und Elternadressen."
		b.Abhilfe = "Die Zeile aus der .env entfernen und den Stack neu starten, außer der Mailserver " +
			"der Schule verlangt die Ausnahme wirklich, etwa als altes Relay oder mit selbstsigniertem Zertifikat"
		return b
	}
	if l.SMTPHost != "" {
		b.Stufe = StufeOK
		b.Befund = "Versand über " + l.SMTPHost
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = "Kein SMTP-Server in den Einstellungen"
	b.Folge = "Mahnungen und Bestellmails können nicht zugestellt werden"
	b.Abhilfe = "Einstellungen → Mail: Postausgang ausfüllen und mit dem Testversand prüfen"
	return b
}
