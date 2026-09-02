package repository

// Die automatische Abgänger-Sperre hat EIN Präfix und EINE Uhr (Ratsche
// abgaenger_sperre_ratsche_test.go, Rasterdurchgang 02.09.2026). Drei Wege machen
// einen Schüler zum Abgänger — LUSD-Import (api/lusd_apply.go), Versetzung
// (api/student_promotion.go) und indirekt das Zusammenführen — und zwei Wege heben die
// Sperre wieder auf (Rückkehr per Import, Zusammenführen). Sie erkennen die Automatik
// am Präfix des Sperrgrunds; ein manueller Grund trägt es nicht und bleibt stehen.
// Bis Migration 095 schrieb die Versetzung „Automatische …" — die Rückkehr erkannte das
// nicht, der Schüler kam aktiv, aber gesperrt zurück (Ghost-Block).
const (
	AbgaengerSperrPraefix = "Automatisierte Abgänger-Sperre"

	AbgaengerSperrgrundOffen              = AbgaengerSperrPraefix + " (offene Vorgänge)"
	AbgaengerSperrgrundKarenz             = AbgaengerSperrPraefix + " (Karenzzeit vor Anonymisierung)"
	AbgaengerSperrgrundSchuljahreswechsel = AbgaengerSperrPraefix + " (Schuljahreswechsel)"

	// SQLAbgaengerSperreAutomatisch ist das Prädikat, an dem SQL die Automatik erkennt.
	SQLAbgaengerSperreAutomatisch = "block_reason LIKE '" + AbgaengerSperrPraefix + "%'"
)
