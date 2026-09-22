package api

import (
	"testing"

	"bibliothek/repository"
)

// Gate: Die Waisenübernahme der Idempotenz kann keinen lebenden Besitzer treffen.
//
// repository.ReserviereIdempotenzSchluessel übernimmt eine Reservierung, die älter als
// IdempotenzReservierungsfrist ist, mit der Begründung „der Server ist gestorben". Das ist nur
// dann die einzige Erklärung, wenn kein Besitzer so lange leben KANN: Die TimeoutMiddleware
// bricht seine Arbeit nach RequestFrist ab, und saveToCache darf danach noch
// idempotenzSpeicherfrist lang schreiben — mit WithoutCancel, also über die Bearbeitungsfrist
// hinaus. Erst wenn beides zusammen unter der Übernahmefrist liegt, ist die Reservierung beim
// Übernehmen wirklich verwaist.
//
// Bis zum 15.09.2026 standen die drei Werte als 60, 15 und 5 Sekunden in drei Dateien
// (repository/idempotenz.go, api/router.go, api/action.go), und nichts hielt sie zusammen
// (Rasterdurchgang 15.09.2026, OFFEN.md 5.14). Wer die Theke in die lang laufenden Pfade
// aufnimmt (5 Minuten) oder die Übernahme kürzt, bekäme zwei Besitzer desselben Schlüssels:
// Beide buchen, und der Verlierer meldet das nur im Log.
//
// Geprüft werden die Pfade, die den Schlüssel reservieren. Beim Nachbuchen reserviert jeder
// Eintrag erst vor seiner eigenen Arbeit, also frühestens am Anfang und spätestens kurz vor
// Ablauf der Anfrage — die Schranke gilt je Eintrag ab seiner Reservierung und ist damit
// dieselbe.
func TestIdempotenz_UebernahmeFristLaengerAlsBearbeitungUndSpeichern(t *testing.T) {
	for _, pfad := range []string{"/api/action", "/api/action/nachbuchen"} {
		arbeit := RequestFrist(pfad, StandardBearbeitungsfrist)
		spaetestensFertig := arbeit + idempotenzSpeicherfrist
		if spaetestensFertig >= repository.IdempotenzReservierungsfrist {
			t.Errorf("%s: Arbeit (%s) + Speichern (%s) = %s erreicht die Übernahmefrist (%s) — eine "+
				"laufende Anfrage könnte ihren Schlüssel an die Wiederholung verlieren, und beide "+
				"buchen. Entweder die Frist der Middleware für diesen Pfad senken oder "+
				"repository.IdempotenzReservierungsfrist anheben.",
				pfad, arbeit, idempotenzSpeicherfrist, spaetestensFertig, repository.IdempotenzReservierungsfrist)
		}
	}
}
