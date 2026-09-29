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

// ExemplarTopfSQL ist der Topf eines EXEMPLARS als SQL-Ausdruck — das Eigentum folgt dem
// Geld. Zuerst gilt das Eigentum, das am Exemplar ausdrücklich steht (Migration 150: aus dem
// Littera-Vermerk oder von Hand), dann die Zuordnung seiner Bestellung, und wo es beides
// nicht gibt (Altbestand ohne Vermerk, Alt-Bestellungen ohne Zuordnung), das Feld
// ist_lernmittel am Titel — dieselbe Faustregel wie beim Nachtragen in Migration 109.
//
// Das Ausdrückliche geht vor, weil die Faustregel beim Altbestand nachweislich danebenliegt:
// Aus LMF-Mitteln dürfen auch Lektüren, Ganzschriften und Nachschlagewerke gekauft werden,
// und sie bleiben Eigentum des Landes (Leitfaden „Lernmittelfreiheit in Hessen", Ziffern 2.1
// und 9.4.2). Littera führt sie mit „Land Hessen", ihre Signatur ist aber keine LMF-Signatur.
//
// EINE Formulierung für alle Wege, die Etikettendaten bauen: Vier Abfragen mit je eigenem
// CASE liefen auseinander, und dasselbe Buch trüge je nach Druckweg einen anderen
// Eigentumsvermerk. Der Ausdruck erwartet die Aliasse e (buecher_exemplare) und
// t (buecher_titel) und braucht ExemplarTopfJoin.
//
// Anders als beim Bestellen wird hier aus dem Titel abgeleitet: Dort wäre ein Fallback die
// stille Zuordnung zum falschen Topf, hier ist er die einzige Auskunft über ein Buch, das
// nie über dieses System bestellt wurde.
const ExemplarTopfSQL = `COALESCE(` + ExemplarTopfBelegtSQL + `, CASE WHEN t.ist_lernmittel THEN 'land' ELSE 'schultraeger' END)`

// ExemplarTopfBelegtSQL ist der Teil von ExemplarTopfSQL, der auf einem Beleg steht: das
// Eigentum am Exemplar oder der Topf seiner Bestellung. NULL, wo es beides nicht gibt. Das
// Zugangsbuch liest nur diesen Teil — es weist nach, aus welchem Geld ein Buch kam, und die
// Faustregel aus dem Titel ist dafür kein Nachweis. Dieselben Aliasse wie ExemplarTopfSQL.
const ExemplarTopfBelegtSQL = `COALESCE(e.eigentum, bv_topf.mittel)`

// ExemplarTopfJoin hängt die Bestellung des Exemplars an, aus der ExemplarTopfSQL liest.
const ExemplarTopfJoin = `LEFT JOIN bestellungen_verlauf bv_topf ON bv_topf.id = e.bestellung_id`

// Den VORSCHLAG aus dem Titel (Lernmittel → Land, sonst Schulträger) rechnet allein der
// Warenkorb (frontend/src/lib/components/bestellungen/mittel.js): Er ist eine Entscheidung
// der Bestellung, die der Server nur noch prüft — ein Server-Fallback wäre die stille
// Zuordnung zum falschen Topf, die Migration 109 abschafft.
