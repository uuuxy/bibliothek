package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"bibliothek/db"
	"bibliothek/repository"
)

// Selbstanmeldung: Wer sich mit einem Postfach der Schule anmeldet und noch kein Konto hat,
// bekommt einen inaktiven Eintrag. Anmelden kann er sich damit nicht, denn die Anmeldung lehnt
// inaktive Konten ab. Die Bibliothek sieht die Anfrage und schaltet frei.
//
// Der Grund: Das Kollegium umfasst rund 160 Personen. Alle vorab von Hand anzulegen ist Arbeit,
// die niemand macht, und ohne Konto kommt keine Lehrkraft an die Reservierung von
// Klassensätzen. Die Selbstanmeldung dreht das um: Ein Eintrag entsteht nur für die, die das
// Programm auch benutzen wollen.
//
// Freigeschaltet wird trotzdem: IMAP beantwortet „wer bist du", nicht „darfst du hinein". Der
// Mailserver ist keine Quelle für Rechte. Die Entscheidung bleibt bei der Schule; sie kostet nur
// einen Klick statt eines Formulars.

// selbstanmeldeDomainVariable nennt die Umgebungsvariable mit der Domain, deren Postfächer sich
// selbst anmelden dürfen (etwa "schule.example"). Ist sie nicht gesetzt, ist die
// Selbstanmeldung abgeschaltet.
//
// Eine ausdrückliche Domain und kein Schalter: Ohne Einschränkung entschiede der Mailserver
// darüber, wer ins Programm darf. Solange dort nur das Kollegium liegt, wäre das vertretbar,
// aber das ist eine Annahme über eine fremde Anlage, und solche Annahmen veralten unbemerkt. Die
// Domain macht die Bedingung sichtbar und prüfbar.
const selbstanmeldeDomainVariable = "SELBSTANMELDUNG_DOMAIN"

// SelbstanmeldeDomain liefert die freigegebene Domain in Kleinschreibung, oder "", wenn die
// Selbstanmeldung abgeschaltet ist.
func SelbstanmeldeDomain() string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(
		os.Getenv(selbstanmeldeDomainVariable), "@")))
}

// SelbstanmeldungStatus beschreibt den Zustand für das Log beim Start.
//
// Es gibt die Funktion, damit die Abschaltung nicht unbemerkt bleibt: Eine vergessene
// Einstellung sieht sonst genauso aus wie eine, die wirkt, und die Lehrkraft bekommt in beiden
// Fällen nur „Anmeldung fehlgeschlagen".
func SelbstanmeldungStatus() string {
	if d := SelbstanmeldeDomain(); d != "" {
		return fmt.Sprintf("Selbstanmeldung aktiv für @%s — neue Konten entstehen inaktiv "+
			"und müssen freigeschaltet werden", d)
	}
	return fmt.Sprintf("Selbstanmeldung abgeschaltet (%s nicht gesetzt) — Konten legt "+
		"ausschließlich die Benutzerverwaltung an", selbstanmeldeDomainVariable)
}

// darfSichSelbstAnmelden prüft die Domain der Adresse gegen die Freigabe.
//
// Der Vergleich läuft über das letzte "@" und die vollständige Domain, nicht über
// strings.Contains oder HasSuffix. Ein Vergleich des Endes ließe "boeseschule.example" durch,
// ein Contains sogar "kein@angreifer.example/schule.example".
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
// vorname und nachname sind NOT NULL, IMAP liefert sie aber nicht mit. Geraten wird nur aus
// "vorname.nachname", dem Muster der Adressen der Schule. Passt es nicht, ist der Nachname der
// ganze Adressteil und der Vorname leer; die Bibliothek sieht beim Freischalten, was daraus
// geworden ist, und kann es richtigstellen. Kein Kunstname wie "Unbekannt": Der stünde später
// in Mahnungen.
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

// grossErsterBuchstabe schreibt den ersten Buchstaben groß und lässt den Rest, wie er ist:
// "berger" wird zu "Berger", "McDonald" bleibt "McDonald".
func grossErsterBuchstabe(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return ""
	}
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// legeZugangsanfrageAn legt ein inaktives Konto an und gibt es zurück.
//
// ON CONFLICT (email) DO NOTHING und danach Lesen: Zwei gleichzeitige Anmeldeversuche derselben
// Person (zwei Geräte, Doppelklick) dürfen nicht an der eindeutigen Spalte email scheitern. Wer
// verliert, liest die Zeile des Gewinners.
func legeZugangsanfrageAn(ctx context.Context, pool db.Pool, email string) (anmeldeKonto, error) {
	if !darfSichSelbstAnmelden(email) {
		return anmeldeKonto{}, errSelbstanmeldungNichtErlaubt
	}

	vorname, nachname := namenAusAdresse(email)

	// aktiv = false ist der Kern der Funktion: Die Anmeldung lehnt inaktive Konten ab. Die Zeile
	// entsteht, der Zugang nicht. neuAngelegt gilt nur, wenn dabei wirklich eine Zeile entstand:
	// Trifft die Anlage ein vorhandenes Konto, bekäme sonst eine längst freigeschaltete Lehrkraft
	// „Zugang beantragt", und das Protokoll nennte eine Selbstanmeldung, die nie stattfand.
	neuAngelegt, err := repository.LegeZugangsanfrageAn(ctx, pool, vorname, nachname, email)
	if err != nil {
		return anmeldeKonto{}, fmt.Errorf("%w: zugangsanfrage konnte nicht angelegt werden: %v", errAnmeldedienstGestoert, err)
	}

	// Dasselbe Lesen wie in der Anmeldung: Trifft die Anlage ein vorhandenes Konto, geht die
	// Anmeldung mit ihm weiter.
	gelesen, err := repository.LadeAnmeldeKonto(ctx, pool, email)
	if err != nil {
		return anmeldeKonto{}, fmt.Errorf("%w: zugangsanfrage konnte nicht gelesen werden: %v", errAnmeldedienstGestoert, err)
	}
	konto := anmeldeKontoAus(gelesen)
	konto.neuAngelegt = neuAngelegt
	if !neuAngelegt {
		// Ein vorhandenes Konto, aktiv oder wartend: kein neuer Vorgang, kein Eintrag im
		// Protokoll.
		return konto, nil
	}

	// Ein Konto, das ohne Administrator entstanden ist, braucht erst recht einen Eintrag im
	// Protokoll, sonst steht eine Zeile in benutzer, von der niemand sagen kann, wer sie wann
	// angelegt hat. admin_id ist das neue Konto selbst: Es hat sich angemeldet. Ein Fehler beim
	// Schreiben hält die Anmeldung nicht auf, wie in tuer.Basis.ProtokolliereVerwaltung.
	// ziel_id trägt die Kennung des Kontos wie bei USER_CREATE: admin_id leert sich, wenn das
	// Konto gelöscht wird, und die Auskunft fände den Eintrag dann nicht mehr.
	details, err := json.Marshal(map[string]any{"ziel_id": konto.id, "email": strings.ToLower(email), "rolle": "kollegium", "aktiv": false})
	if err != nil {
		details = []byte("{}")
	}
	if err := repository.SchreibeAdminProtokoll(ctx, pool, konto.id, "SELBSTANMELDUNG", "", string(details)); err != nil {
		slog.Warn("Selbstanmeldung: Audit-Zeile konnte nicht geschrieben werden", "fehler", err)
	}
	return konto, nil
}

// errSelbstanmeldungNichtErlaubt heißt: Die Adresse gehört nicht zur freigegebenen Domain, oder
// die Selbstanmeldung ist ganz abgeschaltet. Die Antwort ist die gewöhnliche 401; von außen ist
// nicht zu unterscheiden, ob es das Konto nicht gibt oder ob es sich nur nicht anmelden darf.
var errSelbstanmeldungNichtErlaubt = errors.New("selbstanmeldung für diese Adresse nicht freigegeben")
