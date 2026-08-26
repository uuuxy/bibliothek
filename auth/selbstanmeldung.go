package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"bibliothek/db"
)

// Selbstanmeldung: Wer sich mit einem Schul-Mailkonto anmeldet und im System noch nicht
// existiert, bekommt einen INAKTIVEN Eintrag. Anmelden kann er sich damit nicht — der
// Login lehnt inaktive Konten ab. Die Bibliothek sieht die Anfrage und schaltet frei.
//
// Warum überhaupt: Das Kollegium umfasst rund 160 Personen. Alle vorab von Hand
// anzulegen ist Arbeit, die niemand macht — und ohne Konto kommt keine Lehrkraft an die
// Klassensatz-Reservierung. Die Selbstanmeldung dreht das um: Ein Eintrag entsteht nur
// für die, die das Programm auch benutzen wollen.
//
// Warum trotzdem eine Freischaltung: IMAP beantwortet „wer bist du", nicht „darfst du
// rein". Der Mailserver war nie als Rechtequelle gedacht. Die Entscheidung bleibt bei
// der Schule; sie kostet nur einen Klick statt eines Formulars.

// selbstanmeldeDomainEnv nennt die Domain, deren Mailkonten sich selbst anmelden dürfen
// (z. B. "philipp-reis-schule.de"). NICHT gesetzt = Selbstanmeldung abgeschaltet.
//
// Bewusst eine ausdrückliche Domain und kein Schalter: Ohne Einschränkung entschiede der
// Mailserver darüber, wer ins Bibliothekssystem darf. Solange dort nur Kollegium liegt,
// wäre das vertretbar — aber das ist eine Annahme über eine fremde Anlage, und Annahmen
// dieser Art veralten still. Die Domain macht die Bedingung sichtbar und prüfbar.
const selbstanmeldeDomainEnv = "SELBSTANMELDUNG_DOMAIN"

// SelbstanmeldeDomain liefert die freigegebene Domain in Kleinschreibung, oder "" wenn
// die Selbstanmeldung abgeschaltet ist.
func SelbstanmeldeDomain() string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(
		os.Getenv(selbstanmeldeDomainEnv), "@")))
}

// SelbstanmeldungStatus beschreibt den Zustand für das Startprotokoll.
//
// Es gibt diese Funktion, damit die Abschaltung nicht STILL passiert: Eine Einstellung,
// die man vergessen hat, sieht sonst genauso aus wie eine, die funktioniert — und die
// Lehrkraft bekommt in beiden Fällen nur „Anmeldung fehlgeschlagen".
func SelbstanmeldungStatus() string {
	if d := SelbstanmeldeDomain(); d != "" {
		return fmt.Sprintf("Selbstanmeldung aktiv für @%s — neue Konten entstehen inaktiv "+
			"und müssen freigeschaltet werden", d)
	}
	return fmt.Sprintf("Selbstanmeldung abgeschaltet (%s nicht gesetzt) — Konten legt "+
		"ausschließlich die Benutzerverwaltung an", selbstanmeldeDomainEnv)
}

// darfSichSelbstAnmelden prüft die Domain der Adresse gegen die Freigabe.
//
// Der Vergleich läuft über das LETZTE "@" und die vollständige Domain — nicht über
// strings.Contains oder HasSuffix. Ein Suffix-Vergleich ließe "boesephilipp-reis-schule.de"
// durch, ein Contains sogar "kein@angreifer.de/philipp-reis-schule.de".
func darfSichSelbstAnmelden(email string) bool {
	freigegeben := SelbstanmeldeDomain()
	if freigegeben == "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(email[at+1:]), freigegeben)
}

// namenAusAdresse rät Vor- und Nachnamen aus dem Adressteil vor dem "@".
//
// vorname/nachname sind NOT NULL, IMAP liefert sie aber nicht mit. Geraten wird nur aus
// "vorname.nachname" — dem Muster der Schuladressen. Passt es nicht, bleibt der
// Nachname der ganze Adressteil und der Vorname leer; die Bibliothek sieht beim
// Freischalten, was daraus geworden ist, und kann es richtigstellen. Bewusst kein
// Kunstname wie "Unbekannt": Der stünde später in Mahnungen.
func namenAusAdresse(email string) (vorname, nachname string) {
	lokal := email
	if at := strings.LastIndex(email, "@"); at > 0 {
		lokal = email[:at]
	}
	lokal = strings.TrimSpace(lokal)

	teile := strings.SplitN(lokal, ".", 2)
	if len(teile) == 2 && teile[0] != "" && teile[1] != "" {
		return grossErsterBuchstabe(teile[0]), grossErsterBuchstabe(teile[1])
	}
	return "", grossErsterBuchstabe(lokal)
}

// grossErsterBuchstabe schreibt den ersten Buchstaben groß und lässt den Rest, wie er
// ist — "flasch" wird zu "Flasch", "McDonald" bleibt "McDonald".
func grossErsterBuchstabe(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return ""
	}
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// legeZugangsanfrageAn legt einen INAKTIVEN Benutzer an und gibt ihn zurück.
//
// ON CONFLICT (email) DO NOTHING plus anschließendes Lesen: Zwei gleichzeitige
// Anmeldeversuche derselben Person (zwei Geräte, Doppelklick) dürfen nicht an der
// eindeutigen E-Mail-Spalte scheitern. Wer verliert, liest die Zeile des Gewinners.
func legeZugangsanfrageAn(ctx context.Context, dbPool db.PgxPoolIface, email string) (loginUser, error) {
	if !darfSichSelbstAnmelden(email) {
		return loginUser{}, errSelbstanmeldungNichtErlaubt
	}

	vorname, nachname := namenAusAdresse(email)

	// aktiv = false ist der Kern dieser Funktion: Der Login lehnt inaktive Konten ab.
	// Die Zeile entsteht, der Zugang nicht.
	if _, err := dbPool.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
		VALUES ($1, $2, LOWER($3), 'kollegium', false, CURRENT_TIMESTAMP)
		ON CONFLICT (email) DO NOTHING
	`, vorname, nachname, email); err != nil {
		return loginUser{}, fmt.Errorf("zugangsanfrage konnte nicht angelegt werden: %w", err)
	}

	var u loginUser
	err := dbPool.QueryRow(ctx, `
		SELECT id, coalesce(barcode_id, ''), rolle, vorname, nachname, aktiv
		FROM benutzer WHERE LOWER(email) = LOWER($1) LIMIT 1
	`, email).Scan(&u.id, &u.barcodeID, &u.roleStr, &u.vorname, &u.nachname, &u.aktiv)
	if err != nil {
		return loginUser{}, fmt.Errorf("zugangsanfrage konnte nicht gelesen werden: %w", err)
	}
	u.neuAngelegt = true
	return u, nil
}

// errSelbstanmeldungNichtErlaubt: Adresse gehört nicht zur freigegebenen Domain, oder die
// Selbstanmeldung ist ganz abgeschaltet. Führt zur normalen 401-Antwort — nach außen ist
// nicht zu unterscheiden, ob es das Konto nicht gibt oder es nur nicht darf.
var errSelbstanmeldungNichtErlaubt = errors.New("selbstanmeldung für diese Adresse nicht freigegeben")
