package bereitschaft

import "strings"

// ErzwingeProdGeheimnisse entscheidet, ob der Server den Start mit einem bekannten
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
func ErzwingeProdGeheimnisse(appEnv, roh string) bool {
	wert := strings.ToLower(strings.TrimSpace(roh))
	if wert == "false" {
		return false
	}
	if wert == "" && !IstEchterBetrieb(strings.ToLower(strings.TrimSpace(appEnv))) {
		return false
	}
	return true
}
