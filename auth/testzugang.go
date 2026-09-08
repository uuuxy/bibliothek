package auth

// Außerordentlicher Zugang zum Testen (08.09.2026).
//
// Warum es ihn gibt: Angemeldet wird ausschließlich per IMAP gegen den Schul-Mailserver
// (auth/imap.go). Wer dort kein Postfach hat, kommt nicht hinein — und genau das trifft
// jeden, der das System von außen ansehen soll, ohne Angehöriger der Schule zu sein.
// IMAP_HOST=mock wäre der bequeme Weg, ist aber keiner: Der Mock akzeptiert JEDES
// Passwort für JEDE eingetragene Adresse und öffnet damit alle Konten gleichzeitig.
//
// Dieser Zugang öffnet stattdessen GENAU EINE Adresse mit GENAU EINEM Passwort und lässt
// jede andere Anmeldung unverändert über den Mailserver laufen.
//
// Er hängt allein an zwei Umgebungsvariablen. Sind sie nicht gesetzt, existiert der Weg
// nicht — es gibt keinen Schalter in der Oberfläche, keine Zeile in der Datenbank und
// keinen Zustand, den man versehentlich anlassen kann. Abschalten heißt: Variablen aus
// der .env nehmen und den Container neu starten.

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	testzugangEmailEnv    = "TESTZUGANG_EMAIL"
	testzugangPasswortEnv = "TESTZUGANG_PASSWORT"

	// testzugangMindestlaenge: Das Passwort ist die EINZIGE Schranke vor einem Konto mit
	// Admin-Rechten — hier gibt es keinen zweiten Faktor und keinen Mailserver dahinter,
	// der auch noch zustimmen müsste. Der Brute-Force-Zähler (5 Fehlversuche je
	// E-Mail+IP, 15 Minuten) bremst zwar, aber er bremst nur pro IP. 20 Zeichen sind
	// die Untergrenze, unterhalb derer der Server den Start verweigert.
	testzugangMindestlaenge = 20
)

// TestzugangEmail liefert die freigeschaltete Adresse in Kleinschreibung, oder "".
func TestzugangEmail() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv(testzugangEmailEnv)))
}

func testzugangPasswort() string {
	return os.Getenv(testzugangPasswortEnv)
}

// TestzugangAktiv meldet, ob beide Variablen gesetzt sind. Halb gesetzt zählt nicht als
// aktiv — PruefeTestzugangKonfiguration lässt den Server in dem Fall gar nicht erst hoch.
func TestzugangAktiv() bool {
	return TestzugangEmail() != "" && testzugangPasswort() != ""
}

// PruefeTestzugangKonfiguration validiert die beiden Variablen beim Serverstart.
//
// Jeder Fall hier endet in log.Fatalf, nicht in einer Warnung. Ein halb oder falsch
// konfigurierter Sonderzugang ist die schlechteste aller Lagen: Er sieht von außen aus
// wie ein funktionierender (der Tester bekommt „Anmeldung fehlgeschlagen" und hält es
// für sein Passwort) oder wie ein sicherer (das Passwort ist in Wahrheit zu kurz).
func PruefeTestzugangKonfiguration() error {
	email := TestzugangEmail()
	passwort := testzugangPasswort()

	// Halb gesetzt: eine tote Tür. Wer nur die Adresse einträgt, hat den Zugang in
	// seinem Kopf eingerichtet — und wundert sich über die abgelehnte Anmeldung.
	if email == "" && passwort != "" {
		return fmt.Errorf("%s ist gesetzt, %s nicht — der Testzugang wäre wirkungslos", testzugangPasswortEnv, testzugangEmailEnv)
	}
	if email != "" && passwort == "" {
		return fmt.Errorf("%s ist gesetzt, %s nicht — der Testzugang wäre wirkungslos", testzugangEmailEnv, testzugangPasswortEnv)
	}
	if email == "" {
		return nil // beide leer: der Normalfall, es gibt keinen Testzugang
	}

	if !strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@") {
		return fmt.Errorf("%s=%q ist keine E-Mail-Adresse", testzugangEmailEnv, email)
	}

	// Die Adresse darf nicht zur Schuldomain gehören. Zwei Gründe, beide ernst:
	//
	//   - Sie würde eine Schulidentität tragen. Wer das Testpasswort hat, wäre im
	//     Protokoll von einer echten Person nicht zu unterscheiden.
	//   - Existiert dort ein echtes Postfach, hätte dieselbe Adresse zwei Passwörter:
	//     das des Mailservers und dieses. Der Inhaber des Kontos wüsste nichts davon.
	if d := SelbstanmeldeDomain(); d != "" && strings.HasSuffix(email, "@"+d) {
		return fmt.Errorf("%s=%q liegt auf der Schuldomain @%s — der Testzugang darf keine Schulidentität tragen; eine eigene Adresse verwenden (z. B. test@bibliothek.invalid)", testzugangEmailEnv, email, d)
	}

	if len([]rune(passwort)) < testzugangMindestlaenge {
		return fmt.Errorf("%s ist zu kurz (%d Zeichen, mindestens %d) — es ist die einzige Schranke vor einem Konto mit Admin-Rechten", testzugangPasswortEnv, len([]rune(passwort)), testzugangMindestlaenge)
	}
	if strings.EqualFold(strings.TrimSpace(passwort), email) {
		return errors.New(testzugangPasswortEnv + " ist die Adresse selbst — bitte ein zufälliges Passwort setzen")
	}
	return nil
}

// TestzugangStatus beschreibt den Zustand für das Startprotokoll. Ein aktiver
// Sonderzugang darf nicht schweigend laufen; wer die Logs liest, muss ihn sehen.
func TestzugangStatus() string {
	if !TestzugangAktiv() {
		return fmt.Sprintf("Testzugang abgeschaltet (%s nicht gesetzt) — Anmeldung ausschließlich über den Mailserver", testzugangEmailEnv)
	}
	return fmt.Sprintf("⚠️  TESTZUGANG AKTIV für %s — diese eine Adresse meldet sich ohne Mailserver an, mit den Rechten ihres Kontos. Zum Abschalten %s und %s aus der Umgebung nehmen und neu starten.",
		TestzugangEmail(), testzugangEmailEnv, testzugangPasswortEnv)
}

// istTestzugang prüft, ob diese Anmeldung den Sonderweg nimmt.
//
// Beide Vergleiche laufen über subtle.ConstantTimeCompare, auch der der Adresse: Ein
// Vergleich, der beim ersten abweichenden Byte aussteigt, verrät über die Antwortzeit,
// wie viele Zeichen stimmten. Bei einem Passwort, das nirgends gehasht liegt und ein
// Admin-Konto öffnet, ist das keine theoretische Sorge.
//
// Die Reihenfolge ist Absicht: erst die Adresse, dann das Passwort. Sonst würde jede
// falsche Anmeldung im Haus gegen das Testpasswort geprüft.
func istTestzugang(email, passwort string) bool {
	if !TestzugangAktiv() {
		return false
	}
	erwarteteMail := TestzugangEmail()
	angeboteneMail := strings.ToLower(strings.TrimSpace(email))
	if subtle.ConstantTimeCompare([]byte(angeboteneMail), []byte(erwarteteMail)) != 1 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(passwort), []byte(testzugangPasswort())) == 1
}
