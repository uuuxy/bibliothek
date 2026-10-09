// Die Art eines Lesers — Schüler, Lehrkraft, LiV und die Sonderkonten Praktikum, Sekretariat,
// U-plus und Fachbereich (Migration 123/125/153).
//
// Eine Datei, weil vier Ansichten dasselbe Wort brauchen: die Trefferliste an der Theke,
// die schmale Karte eines geladenen Kollegen, die Liste der Leserdatei und die Akte.
// Stünde die Zuordnung an jeder Stelle einzeln, hieße derselbe Mensch an der Theke
// „LiV" und in der Leserdatei „Referendar".
//
// Entschieden wird an der Art nur zweierlei: Schüler oder Kollegium (istKollegium) und, im
// Kollegium, ob ein Zugang zu „Mein Portal" dazugehört (artMitKonto). Alles andere ist
// Bezeichnung — ausleihen darf jeder aktive Leser. Dieselben Werte stehen im Server
// (pkg/leserart); beide Seiten prüft leserArt.faelle.json.

/** Die Arten in der Reihenfolge der Auswahl „Art des Lesers". */
export const LESER_ARTEN = [
	'schueler',
	'lehrkraft',
	'liv',
	'praktikum',
	'sekretariat',
	'uplus',
	'fachbereich'
];

/** @type {Record<string, string>} */
const BEZEICHNUNG = {
	schueler: 'Schüler',
	lehrkraft: 'Lehrkraft',
	liv: 'LiV',
	praktikum: 'Praktikum',
	sekretariat: 'Sekretariat',
	uplus: 'U-plus',
	fachbereich: 'Fachbereich'
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
 * Ist dieser Leser ein Kollege — jede Art außer Schüler? Die Frage steht an genug Stellen,
 * dass sie nicht überall neu als Vergleich geschrieben werden sollte.
 * @param {{ art?: string | null } | null | undefined} leser
 * @returns {boolean}
 */
export function istKollegium(leser) {
	return !!leser?.art && leser.art !== 'schueler';
}

/**
 * Gehört zu dieser Art ein Zugang zu „Mein Portal" — und damit die Schul-E-Mail, aus der er
 * entsteht? Lehrkraft, LiV, Sekretariat und U-plus ja; Praktikum und Fachbereich nicht
 * (Entscheidung vom 30.09.2026: Ein Fachbereich ist ein Sammelkonto der Kollegen des Fachs,
 * ein Praktikant leiht aus, braucht aber keinen Zugang). Ein Schüler hat nie ein Konto.
 * @param {string | null | undefined} art
 * @returns {boolean}
 */
export function artMitKonto(art) {
	return art === 'lehrkraft' || art === 'liv' || art === 'sekretariat' || art === 'uplus';
}

/**
 * Die Aufschrift des Ausweises. Sie steht auf der Karte und sagt, WAS das Dokument ist —
 * nicht, wer jemand ist. Ein Kollege bekam bis zum 16.09.2026 einen Ausweis mit der
 * Aufschrift „Schülerausweis": Der Titel war ein fester Text im Design und kannte die
 * Art nicht.
 *
 * LiV und Lehrkraft bekommen dieselbe Aufschrift. Der Ausweis weist jemanden als
 * Lehrkraft der Schule aus; die Ausbildungsstufe gehört nicht auf die Karte. Die
 * Sonderkonten sind keine Lehrkräfte: Ihre Karte heißt „Leserausweis", das Wort, das
 * Littera für jeden Ausweis führt (30.09.2026).
 * @param {string | null | undefined} art
 * @returns {string}
 */
export function ausweisTitel(art) {
	if (!istKollegium({ art })) return 'Schülerausweis';
	return art === 'lehrkraft' || art === 'liv' ? 'Lehrerausweis' : 'Leserausweis';
}
