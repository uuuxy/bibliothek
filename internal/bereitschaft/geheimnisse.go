package bereitschaft

import "strings"

// IstBeispielGeheimnis meldet, ob ein Wert eines der mitgelieferten
// Beispiel-Geheimnisse ist.
//
// Die Liste stand bis zum 11.08.2026 fest verdrahtet in main.go. Sie hier ein zweites Mal
// aufzuschreiben hätte genau die Fehlerart erzeugt, gegen die diese Datei antritt: zwei
// Listen, die auseinanderlaufen, und eine Selbstprüfung, die „alles gut" meldet, während
// der Server aus demselben Grund den Start verweigert.
func IstBeispielGeheimnis(wert string) bool {
	switch wert {
	case "super-secret-default-key-at-least-32-bytes", // Default aus docker-compose.yml (JWT)
		"super-secure-aes-key-32-chars-ok", // Default aus docker-compose.yml (AES)
		"supergeheim_lokal",
		// Platzhalter aus .env.example (07.09.2026, Sicherheits-Audit): Wer die Datei
		// nach .env kopiert und nur die per docker-compose `:?` erzwungenen Werte
		// (POSTGRES_PASSWORD, IMAP_HOST) setzt, ließ diese beiden unverändert — und der
		// Server startete mit APP_ENV=production trotz scharfem ENFORCE_PROD_SECRETS,
		// weil der Wächter genau die im Repository lesbaren Beispielwerte nicht kannte.
		// Beide erfüllen zugleich die Längenprüfung, fielen also durch jede andere
		// Schranke. Die Selbstprüfung nutzt dieselbe Liste (betriebsbereitschaft_handler),
		// meldete die Absicherung also fälschlich als „scharf".
		"super-secret-key-that-is-at-least-32-bytes-long", // .env.example JWT_SECRET
		"your-32-character-encryption-key":                // .env.example APP_ENCRYPTION_KEY
		return true
	}
	return false
}

// IstGeheimnisSchutzScharf entscheidet, ob der Server den Start mit einem bekannten
// Beispiel-Geheimnis verweigert (main.go) und ob die Selbstprüfung die Absicherung als scharf
// meldet (pruefeGeheimnisse). Eine Funktion für beide, sonst sagt die Seite „scharf", während
// der Server aus demselben Grund durchstartet.
//
// Die Vorgabe ist an: Mit „aus, solange niemand ENFORCE_PROD_SECRETS=true schreibt" liefe der
// Schulserver nach einer vergessenen Zeile in der .env mit dem JWT-Schlüssel aus dem
// Repository, und Admin-Sitzungen wären fälschbar, ohne dass etwas rot würde. Aus ist sie nur,
// wenn
//
//   - die Umgebung eine Spielwiese ist (APP_ENV=local/development/test) und die Variable nicht
//     gesetzt ist: Dort sind die Beispielwerte die richtigen Werte; oder
//   - jemand ausdrücklich ENFORCE_PROD_SECRETS=false schreibt. Das bleibt möglich (Testphase
//     auf einem Server mit APP_ENV=production), steht dann aber sichtbar in der .env und wird
//     von Selbstprüfung und pruefe_secrets.sh gemeldet.
//
// Ein unlesbarer Wert („ja", „0") zählt als an: Unsicher muss man hinschreiben können, aber
// nicht vertippen.
func IstGeheimnisSchutzScharf(umgebung, roh string) bool {
	wert := strings.ToLower(strings.TrimSpace(roh))
	if wert == "false" {
		return false
	}
	if wert == "" && !IstEchterBetrieb(strings.ToLower(strings.TrimSpace(umgebung))) {
		return false
	}
	return true
}

func pruefeGeheimnisse(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Geheimnisse"}
	beispielAktiv := IstBeispielGeheimnis(l.JWTGeheimnis) ||
		IstBeispielGeheimnis(l.Schluessel)

	switch {
	case !beispielAktiv && l.GeheimnisSchutzScharf:
		b.Stufe = StufeOK
		b.Befund = "Eigene Geheimnisse gesetzt, die harte Absicherung ist scharf."
	case !beispielAktiv:
		b.Stufe = StufeWarnung
		b.Befund = "Eigene Geheimnisse gesetzt, ENFORCE_PROD_SECRETS steht aber ausdrücklich auf false."
		b.Folge = "Der Server würde auch mit einem Beispiel-Geheimnis starten — " +
			"ein versehentliches Zurückfallen fiele nicht auf."
		b.Abhilfe = "Die Zeile ENFORCE_PROD_SECRETS=false aus der .env entfernen (Vorgabe ist scharf)."
	case echt:
		b.Stufe = StufeKritisch
		b.Befund = "Es laufen mitgelieferte Beispiel-Geheimnisse."
		b.Folge = "JWT_SECRET und APP_ENCRYPTION_KEY stehen öffentlich im Repository. " +
			"Wer sie kennt, kann Sitzungen fälschen und verschlüsselte Ablagen lesen."
		b.Abhilfe = "Eigene Werte setzen (JWT_SECRET ≥32 Zeichen, APP_ENCRYPTION_KEY genau 32 Byte) " +
			"und die Zeile ENFORCE_PROD_SECRETS=false entfernen."
	default:
		b.Stufe = StufeOK
		b.Befund = befundNichtImEchtbetrieb("Beispiel-Geheimnisse", l.Umgebung)
	}
	return b
}

// pruefeSchluesselPasstZumBestand: Passt APP_ENCRYPTION_KEY zu dem, was in der Datenbank
// verschlüsselt liegt? Der Fall dahinter ist die Wiederherstellung auf einem neuen Server:
// Die Sicherung bringt die Daten zurück, die .env nicht — sie steht in keiner Sicherung.
// Mit einem neuen Schlüssel startet das System ohne Meldung, und gemerkt wird es erst am
// leeren Schülerfoto oder an der Mahnmail, die nicht rausgeht.
//
// Kritisch ohne Blick auf APP_ENV: Ein unlesbarer Bestand ist auf jedem System ein Fehler.
func pruefeSchluesselPasstZumBestand(l Lage) Befund {
	b := Befund{Bereich: "Schlüssel und Bestand"}
	switch {
	case l.SchluesselProbe == nil:
		b.Stufe = StufeWarnung
		b.Befund = "Ob APP_ENCRYPTION_KEY zum Bestand passt, ließ sich nicht prüfen."
		b.Folge = "Ein falscher Schlüssel bliebe unbemerkt, bis ein Foto leer bleibt oder der Mailversand scheitert."
		b.Abhilfe = abhilfeDBNeuLaden
	case len(l.SchluesselProbe.NichtLesbar) > 0:
		b.Stufe = StufeKritisch
		b.Befund = "APP_ENCRYPTION_KEY passt nicht zum Bestand — nicht entschlüsselbar: " +
			strings.Join(l.SchluesselProbe.NichtLesbar, ", ") + "."
		b.Folge = "Schülerfotos bleiben leer, und ohne lesbares SMTP-Passwort geht keine Mail hinaus. " +
			"Die Daten sind nicht verloren, solange der ursprüngliche Schlüssel noch existiert."
		b.Abhilfe = "Den APP_ENCRYPTION_KEY der Anlage, von der die Daten stammen, in die .env eintragen " +
			"und den Stack neu starten. Ist er verloren: SMTP-Passwort in den Einstellungen neu " +
			"eingeben, Fotos neu einspielen."
	case l.SchluesselProbe.Geprueft == 0:
		b.Stufe = StufeOK
		b.Befund = "Es liegt noch nichts verschlüsselt in der Datenbank — der Schlüssel kann nichts verfehlen."
	default:
		b.Stufe = StufeOK
		b.Befund = "APP_ENCRYPTION_KEY passt zum Bestand (Stichprobe je verschlüsselter Spalte)."
	}
	return b
}
