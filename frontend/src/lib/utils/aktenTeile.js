// utils/aktenTeile.js
// Eine Akte (Leser, Titel) lädt ihre Teile nebeneinander. Fehlt ein Teil, steht der Rest trotzdem
// da, und der fehlende ist mit dem Grund des Servers vermerkt: Eine leere Liste ohne Grund sähe
// aus wie „nichts da".
import { leseApiFehler } from '../apiFetch.js';
import { fehlertext } from './fehlertext.js';

/**
 * Ein Teil einer Akte aus seiner Antwort: der Wert oder der Grund, warum er fehlt.
 * @param {PromiseSettledResult<Response>} ergebnis
 * @returns {Promise<{ wert: any, grund: string }>} grund leer, wenn der Teil kam
 */
export async function leseTeil(ergebnis) {
	if (ergebnis.status === 'rejected') return { wert: null, grund: fehlertext(ergebnis.reason) };
	if (!ergebnis.value.ok) return { wert: null, grund: await leseApiFehler(ergebnis.value) };
	return { wert: await ergebnis.value.json(), grund: '' };
}

/**
 * Die Listen, deren Abruf gescheitert ist, mit ihrem Grund; ein Grund für mehrere steht einmal:
 * „Gebühren, Bescheide (Datenbank nicht erreichbar)".
 * @param {{ name: string, grund: string }[]} fehlende
 */
export function nenneFehlende(fehlende) {
	/** @type {Map<string, string[]>} */
	const je = new Map();
	for (const { name, grund } of fehlende) je.set(grund, [...(je.get(grund) ?? []), name]);
	return [...je].map(([grund, namen]) => `${namen.join(', ')} (${grund})`).join(', ');
}
