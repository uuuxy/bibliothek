/** Die Regeln des Felds mit Vorschlägen (VorschlagFeld) — als reine Funktionen.
 *
 *  Eigene Datei aus demselben Grund wie selectTastatur.js: Welche Einträge passen und was eine
 *  Taste tut, geht leicht still kaputt und ist ohne gerendertes Bauteil prüfbar. */

/**
 * Die Vorschläge zum getippten Text: erst die, die mit ihm beginnen, dann die, die ihn
 * enthalten, jeweils in der Reihenfolge des Aufrufers. Groß- und Kleinschreibung zählt nicht,
 * ein leerer Text lässt alle stehen. Jeder Wert kommt nur einmal vor, weil die Liste ihre
 * Zeilen nach ihm schlüsselt.
 * @template {{ wert: string }} V
 * @param {V[]} vorschlaege
 * @param {string} text
 * @returns {V[]}
 */
export function passendeVorschlaege(vorschlaege, text) {
	const gesucht = text.trim().toLocaleLowerCase('de');
	const gesehen = new Set();
	/** @type {V[]} */
	const vorn = [];
	/** @type {V[]} */
	const dahinter = [];
	for (const v of vorschlaege) {
		if (gesehen.has(v.wert)) continue;
		gesehen.add(v.wert);
		const wert = v.wert.toLocaleLowerCase('de');
		if (wert.startsWith(gesucht)) vorn.push(v);
		else if (wert.includes(gesucht)) dahinter.push(v);
	}
	return [...vorn, ...dahinter];
}

/**
 * @typedef {{
 *   tat: 'nichts' | 'oeffnen' | 'wandern' | 'waehlen' | 'schliessen',
 *   richtung?: number,
 *   verhindern: boolean
 * }} VorschlagBefehl
 */

/**
 * Was eine Taste im Feld tut. Verhindert wird nur, was die Liste selbst braucht: Enter ohne
 * markierten Vorschlag geht an das Formular, Escape bei geschlossener Liste an den Dialog.
 * @param {string} taste KeyboardEvent.key
 * @param {boolean} offen die Liste ist zu sehen
 * @param {number} aktiv der markierte Vorschlag, −1 = keiner
 * @param {number} anzahl die passenden Vorschläge
 * @returns {VorschlagBefehl}
 */
export function vorschlagTaste(taste, offen, aktiv, anzahl) {
	const nichts = { tat: /** @type {const} */ ('nichts'), verhindern: false };
	switch (taste) {
		case 'ArrowDown':
			if (offen) return { tat: 'wandern', richtung: 1, verhindern: true };
			return anzahl > 0 ? { tat: 'oeffnen', verhindern: true } : nichts;
		case 'ArrowUp':
			return offen ? { tat: 'wandern', richtung: -1, verhindern: true } : nichts;
		case 'Enter':
			return offen && aktiv >= 0 ? { tat: 'waehlen', verhindern: true } : nichts;
		case 'Escape':
			return offen ? { tat: 'schliessen', verhindern: true } : nichts;
		case 'Tab':
			return offen ? { tat: 'schliessen', verhindern: false } : nichts;
		default:
			return nichts;
	}
}
