package repository

// Die automatische Abgänger-Sperre hat ein Präfix und eine Uhr
// (abgaenger_sperre_ratsche_test.go). Drei Wege machen einen Schüler zum Abgänger: der
// LUSD-Import (internal/lusd/anwenden.go), die Versetzung (schuljahreswechsel.go) und mittelbar
// das Zusammenführen. Zwei Wege heben die Sperre wieder auf: die Rückkehr per Import und das
// Zusammenführen. Sie erkennen die Automatik am Präfix des Sperrgrunds; ein Grund von Hand
// trägt es nicht und bleibt stehen. Schriebe ein Weg einen anderen Text, erkennte die Rückkehr
// ihn nicht, und der Schüler käme aktiv, aber gesperrt zurück.
const (
	AbgaengerSperrPraefix = "Automatisierte Abgänger-Sperre"

	AbgaengerSperrgrundOffen              = AbgaengerSperrPraefix + " wegen offener Vorgänge"
	AbgaengerSperrgrundKarenz             = AbgaengerSperrPraefix + " in der Karenzzeit vor der Anonymisierung"
	abgaengerSperrgrundSchuljahreswechsel = AbgaengerSperrPraefix + " zum Schuljahreswechsel"

	// sqlAbgaengerSperreAutomatisch ist das Prädikat, an dem SQL die Automatik erkennt.
	sqlAbgaengerSperreAutomatisch = "block_reason LIKE '" + AbgaengerSperrPraefix + "%'"
)
