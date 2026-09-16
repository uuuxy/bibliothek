// Die Art eines Lesers — Schüler, Lehrkraft oder LiV (Migration 123/125).
//
// Eine Datei, weil vier Ansichten dasselbe Wort brauchen: die Trefferliste an der Theke,
// die schmale Karte eines geladenen Kollegen, die Liste der Leserdatei und die Akte.
// Stünde die Zuordnung an jeder Stelle einzeln, hieße derselbe Mensch an der Theke
// „LiV" und in der Leserdatei „Referendar".
//
// „LiV" ist das gewählte Wort (15.09.2026) und nur eine Bezeichnung; entschieden wird an der
// Art nichts — ausleihen darf jeder aktive Leser.

/** @type {Record<string, string>} */
const BEZEICHNUNG = {
	schueler: 'Schüler',
	lehrkraft: 'Lehrkraft',
	liv: 'LiV'
};

/**
 * Das Wort zur Art. Eine unbekannte oder fehlende Art ergibt „Schüler": Das ist die
 * Vorgabe der Spalte, und eine Zeile ohne Art ist eine Zeile aus der Zeit davor.
 * @param {string | null | undefined} art
 * @returns {string}
 */
export function leserArtText(art) {
	return BEZEICHNUNG[art ?? ''] ?? BEZEICHNUNG.schueler;
}

/**
 * Ist dieser Leser ein Kollege (Lehrkraft oder LiV)? Die Frage steht an genug Stellen,
 * dass sie nicht überall neu als Vergleich geschrieben werden sollte.
 * @param {{ art?: string | null } | null | undefined} leser
 * @returns {boolean}
 */
export function istKollegium(leser) {
	return !!leser?.art && leser.art !== 'schueler';
}

/**
 * Die Aufschrift des Ausweises. Sie steht auf der Karte und sagt, WAS das Dokument ist —
 * nicht, wer jemand ist. Ein Kollege bekam bis zum 16.09.2026 einen Ausweis mit der
 * Aufschrift „Schülerausweis": Der Titel war ein fester Text im Design und kannte die
 * Art nicht.
 *
 * LiV und Lehrkraft bekommen dieselbe Aufschrift. Der Ausweis weist jemanden als
 * Lehrkraft der Schule aus; die Ausbildungsstufe gehört nicht auf die Karte.
 * @param {string | null | undefined} art
 * @returns {string}
 */
export function ausweisTitel(art) {
	return istKollegium({ art }) ? 'Lehrerausweis' : 'Schülerausweis';
}
