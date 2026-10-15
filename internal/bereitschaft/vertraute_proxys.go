package bereitschaft

import (
	"strings"

	"bibliothek/pkg/clientip"
)

// pruefeVertrauteProxys nennt die Einträge in TRUSTED_PROXIES, die der Server übergeht, weil
// sie weder Netz noch Adresse sind; der Start schreibt sie sonst nur ins Log. Eine Warnung,
// weil ein anderer Eintrag das Netz des Proxys trotzdem decken kann.
func pruefeVertrauteProxys(l Lage) Befund {
	b := Befund{Bereich: "Vertraute Proxys"}
	if unlesbar := clientip.UnlesbareEintraege(l.VertrauteProxys); len(unlesbar) > 0 {
		b.Stufe = StufeWarnung
		b.Befund = "TRUSTED_PROXIES enthält Einträge, die weder Netz noch Adresse sind: " + strings.Join(unlesbar, ", ") + "."
		b.Folge = "Der Server übergeht sie. Fehlt dadurch das Netz des Proxys, sieht er jeden Browser unter der " +
			"Adresse des Proxys: Die ganze Schule teilt sich dann die Bremse der Anfragen und die 50 Fehlversuche " +
			"bei der Anmeldung in 15 Minuten, und das Protokoll der Verwaltung nennt den Proxy statt des Arbeitsplatzes."
		b.Abhilfe = "TRUSTED_PROXIES in der .env berichtigen: Netze wie 172.16.0.0/12 oder einzelne Adressen, " +
			"durch Komma getrennt. Der Server liest den Wert beim Start."
		return b
	}
	b.Stufe = StufeOK
	if strings.TrimSpace(strings.Join(l.VertrauteProxys, "")) == "" {
		b.Befund = "TRUSTED_PROXIES ist leer: Der Server glaubt nur einem Proxy auf demselben Rechner."
		return b
	}
	b.Befund = "Jeder Eintrag in TRUSTED_PROXIES ist ein Netz oder eine Adresse."
	return b
}
