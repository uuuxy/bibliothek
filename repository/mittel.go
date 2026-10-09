package repository

// Der Topf eines Exemplars als SQL-Ausdruck. Welche Töpfe es gibt und wie sie heißen, steht in
// pkg/mitteltopf; hier steht, wie eine Abfrage den Topf eines Buchs bestimmt.

// ExemplarTopfSQL ist der Topf eines Exemplars als SQL-Ausdruck: Das Eigentum folgt dem
// Geld. Zuerst gilt das Eigentum, das am Exemplar ausdrücklich steht (Migration 150: aus dem
// Littera-Vermerk oder von Hand), dann die Zuordnung seiner Bestellung, und wo es beides
// nicht gibt (Altbestand ohne Vermerk, Alt-Bestellungen ohne Zuordnung), das Feld
// ist_lernmittel am Titel — dieselbe Faustregel wie beim Nachtragen in Migration 109.
//
// Das Ausdrückliche geht vor, weil die Faustregel beim Altbestand nachweislich danebenliegt:
// Aus LMF-Mitteln dürfen auch Lektüren und Ganzschriften gekauft werden (Leitfaden
// „Lernmittelfreiheit in Hessen", Ziffer 2.1; Wörterbücher und Lexika 7.5), und was aus
// Landesmitteln beschafft ist, wird als Eigentum des Landes gekennzeichnet (11.1, 11.4). Littera
// führt solche Bücher mit „Land Hessen", ihre Signatur ist aber keine LMF-Signatur.
//
// Alle Wege, die Etikettendaten bauen, nehmen diesen Ausdruck: Mit je eigenem CASE trüge
// dasselbe Buch je nach Druckweg einen anderen Eigentumsvermerk. Der Ausdruck erwartet die
// Aliasse e (buecher_exemplare) und t (buecher_titel) und braucht ExemplarTopfJoin.
//
// Anders als beim Bestellen wird hier aus dem Titel abgeleitet: Dort wäre ein Fallback die
// stille Zuordnung zum falschen Topf, hier ist er die einzige Auskunft über ein Buch, das
// nie über dieses System bestellt wurde.
const ExemplarTopfSQL = `COALESCE(` + ExemplarTopfBelegtSQL + `, CASE WHEN t.ist_lernmittel THEN 'land' ELSE 'schultraeger' END)`

// ExemplarTopfHerkunftSQL sagt, woher ExemplarTopfSQL seinen Wert nimmt: 'littera' oder 'hand'
// (Eigentum am Exemplar, Quelle aus Migration 151), 'bestellung' oder 'vorgabe' (die
// Faustregel aus dem Titel). Die Buchakte nennt es neben dem Eigentum. Dieselbe Reihenfolge wie
// ExemplarTopfSQL, dieselben Aliasse.
const ExemplarTopfHerkunftSQL = `CASE WHEN e.eigentum IS NOT NULL THEN e.eigentum_quelle ` +
	`WHEN bv_topf.mittel IS NOT NULL THEN 'bestellung' ELSE 'vorgabe' END`

// ExemplarTopfBelegtSQL ist der Teil von ExemplarTopfSQL, der auf einem Beleg steht: das
// Eigentum am Exemplar oder der Topf seiner Bestellung. NULL, wo es beides nicht gibt. Das
// Zugangsbuch liest nur diesen Teil — es weist nach, aus welchem Geld ein Buch kam, und die
// Faustregel aus dem Titel ist dafür kein Nachweis. Dieselben Aliasse wie ExemplarTopfSQL.
const ExemplarTopfBelegtSQL = `COALESCE(e.eigentum, bv_topf.mittel)`

// ExemplarTopfJoin hängt die Bestellung des Exemplars an, aus der ExemplarTopfSQL liest.
const ExemplarTopfJoin = `LEFT JOIN bestellungen_verlauf bv_topf ON bv_topf.id = e.bestellung_id`
