package repository

// Mittel: aus welchem Topf eine Bestellung bezahlt wird (bestellungen_verlauf.mittel,
// Migration 109).
//
// Die Schule beschafft aus zwei getrennten Haushalten: Lernmittel (Schulbücher) aus
// Landesmitteln im Rahmen der Lernmittelfreiheit, den Bestand der Schülerbücherei aus
// Mitteln des Schulträgers. Der Händler gewährt darauf verschiedene Nachlässe, und die
// Rechnungen gehen getrennte Wege. Deshalb ist der Topf eine Eigenschaft der BESTELLUNG,
// nicht des Titels: Der Titel (ist_lernmittel) schlägt ihn nur vor.
const (
	// MittelLand: Lernmittelfreiheit — Sammelbestellung, Eigentum des Landes.
	MittelLand = "land"
	// MittelSchultraeger: Schülerbücherei — Anschaffung aus Mitteln des Schulträgers.
	MittelSchultraeger = "schultraeger"
)

// MittelGueltig meldet, ob der Wert zum Vokabular gehört. Dieselbe Menge wie der CHECK
// bestellungen_verlauf_mittel_check — hier, damit die Tür 400 sagt statt 500.
func MittelGueltig(mittel string) bool {
	return mittel == MittelLand || mittel == MittelSchultraeger
}

// Den VORSCHLAG aus dem Titel (Lernmittel → Land, sonst Schulträger) rechnet allein der
// Warenkorb (frontend/src/lib/components/bestellungen/mittel.js): Er ist eine Entscheidung
// der Bestellung, die der Server nur noch prüft — ein Server-Fallback wäre die stille
// Zuordnung zum falschen Topf, die Migration 109 abschafft.
