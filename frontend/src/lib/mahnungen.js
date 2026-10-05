// Wie oft und wann zuletzt ein Mahnbrief gedruckt wurde. Der Server zählt je Buch
// (ausleihen.mahnstufe, letztes_mahndatum); Mahnliste und Bescheid-Fenster sagen es mit
// denselben Worten.
import { formatDatum } from './utils/format.js';

/**
 * Wie die Ansichten der Mahnliste heißen, am Reiter wie auf dem Ausdruck. Der Schlüssel
 * ist der Filterwert des Stores.
 * @type {Record<string, string>}
 */
export const ANSICHTEN = {
	Alle: 'Alle',
	'1. Erinnerung': 'Akut fällig',
	Mahnung: 'Eskaliert',
	Schadensersatz: 'Schadensersatz'
};

/**
 * Die Mahnungen eines Kindes: die höchste Zahl und der jüngste Tag über seine Bücher. Ein
 * später fällig gewordenes Buch steht noch bei null, während zum ersten schon zwei Briefe
 * hinausgingen.
 * @param {any[]} medien
 * @returns {{ gemahnt: number, zuletztGemahnt: string }}
 */
export function mahnungenJeKind(medien) {
	let gemahnt = 0;
	let zuletztGemahnt = '';
	for (const m of medien ?? []) {
		if ((m.mahnstufe ?? 0) > gemahnt) gemahnt = m.mahnstufe;
		// JJJJ-MM-TT ordnet sich als Text wie das Datum.
		if ((m.letztes_mahndatum ?? '') > zuletztGemahnt) zuletztGemahnt = m.letztes_mahndatum;
	}
	return { gemahnt, zuletztGemahnt };
}

/**
 * Der Satz dazu: „noch nicht gemahnt", „2× gemahnt, zuletzt 26.09.2026". Ohne Datum steht
 * nur die Zahl: Eine aus Littera übernommene Ausleihe bringt die Zahl mit, den Tag nicht.
 * @param {number | null | undefined} anzahl
 * @param {string | null | undefined} [zuletzt] Kalendertag der Schule (JJJJ-MM-TT)
 * @returns {string}
 */
export function gemahntSatz(anzahl, zuletzt) {
	if (!anzahl) return 'noch nicht gemahnt';
	const datum = formatDatum(zuletzt);
	return datum ? `${anzahl}× gemahnt, zuletzt ${datum}` : `${anzahl}× gemahnt`;
}
