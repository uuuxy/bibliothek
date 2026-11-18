package repository

// Die automatische Sperre der Ehemaligen hat ein Präfix und eine Uhr
// (pruefung/abgaenger_sperre_ratsche_test.go). Drei Wege machen einen Schüler zum Ehemaligen:
// der LUSD-Import (internal/lusd/anwenden.go), die Versetzung (schuljahreswechsel.go) und
// mittelbar das Zusammenführen. Zwei Wege heben die Sperre wieder auf: die Rückkehr per Import
// und das Zusammenführen. Sie erkennen die Automatik am Präfix des Sperrgrunds; ein Grund von
// Hand trägt es nicht und bleibt stehen. Schriebe ein Weg einen anderen Text, erkennte die
// Rückkehr ihn nicht, und der Schüler käme aktiv, aber gesperrt zurück.
const (
	AbgaengerSperrPraefix = "Automatisierte Ehemaligen-Sperre"

	// AbgaengerSperrgrund ist der Grund jedes Ehemaligen, gleich welcher Weg ihn dazu gemacht
	// hat. Er nennt keinen Zustand, der sich ändert: „wegen offener Vorgänge" stünde nach der
	// letzten Rückgabe falsch in der Akte.
	AbgaengerSperrgrund = AbgaengerSperrPraefix
	// AbgaengerSperrgrundOffen trägt, wer wieder aktiv ist und aus der Zeit als Ehemaliger noch
	// etwas offen hat; der nächste Abgleich hebt die Sperre auf, sobald nichts mehr offen ist.
	AbgaengerSperrgrundOffen = AbgaengerSperrPraefix + " wegen offener Vorgänge"

	// sqlAbgaengerSperreAutomatisch ist das Prädikat, an dem SQL die Automatik erkennt.
	sqlAbgaengerSperreAutomatisch = "block_reason LIKE '" + AbgaengerSperrPraefix + "%'"
)
