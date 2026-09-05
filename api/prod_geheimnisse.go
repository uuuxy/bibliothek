package api

import "strings"

// ErzwingeProdGeheimnisse entscheidet, ob der Server den Start mit einem bekannten
// Beispiel-Geheimnis verweigert (main.go) — und ob die Selbstprüfung
// (betriebsbereitschaft.go) die Absicherung als scharf meldet. EINE Funktion für beide,
// sonst sagt die Seite „scharf", während der Server aus demselben Grund durchstartet.
//
// Bis zum 05.09.2026 galt: aus, solange niemand ENFORCE_PROD_SECRETS=true schrieb. Das
// war die unsichere Richtung als Vorgabe — eine vergessene Zeile in der .env, und der
// Schulserver lief mit dem JWT-Schlüssel aus dem Repository; Admin-Sitzungen wären
// fälschbar gewesen, ohne dass irgendetwas rot wurde. Jetzt gilt: an, außer
//
//   - die Umgebung ist eine Spielwiese (APP_ENV=local/development/test) und die Variable
//     ist nicht gesetzt — dort SIND die Beispielwerte die richtigen Werte; oder
//   - jemand schreibt ausdrücklich ENFORCE_PROD_SECRETS=false. Das bleibt möglich
//     (Testphase auf einem Server mit APP_ENV=production), steht dann aber sichtbar in
//     der .env und wird von Selbstprüfung und pruefe_secrets.sh gemeldet.
//
// Ein unlesbarer Wert („ja", „0") zählt als an: Unsicher muss man hinschreiben können,
// aber nicht vertippen.
func ErzwingeProdGeheimnisse(appEnv, roh string) bool {
	wert := strings.ToLower(strings.TrimSpace(roh))
	if wert == "false" {
		return false
	}
	if wert == "" && !istEchterBetrieb(strings.ToLower(strings.TrimSpace(appEnv))) {
		return false
	}
	return true
}
