import { apiFetch, apiPut } from '../apiFetch.js';

/**
 * Die Schlagworte des Bestands als Vorschläge für ein ChipFeld (Migration 138) — die
 * häufigsten zuerst, höchstens 500 (GET /api/schlagworte).
 *
 * EIN Abruf für alle Felder, die Schlagworte entgegennehmen (Buchformular, Bestellkorb),
 * wie ladeSignaturen für die Signatur. Antwortet der Server nicht, bleibt die Liste leer:
 * Das Feld nimmt weiter freien Text an, es fehlen nur die Vorschläge.
 *
 * @returns {Promise<{ wert: string, beschreibung: string }[]>}
 */
export async function ladeSchlagwortVorschlaege() {
	try {
		const res = await apiFetch('/api/schlagworte');
		if (!res.ok) {
			return [];
		}
		const liste = await res.json();
		if (!Array.isArray(liste)) {
			return [];
		}
		return liste
			.filter((/** @type {any} */ v) => typeof v?.wort === 'string')
			.map((/** @type {{ wort: string, titel: number }} */ v) => ({
				wert: v.wort,
				beschreibung: `${v.titel} Titel`
			}));
	} catch {
		return [];
	}
}

/**
 * Die Schlagworte EINES Titels. Der Bestellkorb braucht sie, bevor er ändert: Er ersetzt
 * die Menge als Ganzes, und wer die vorhandenen nicht kennt, überschriebe sie mit seinem
 * ersten Wort. Anders als bei den Vorschlägen ist ein Fehler hier deshalb ein Fehler —
 * der Aufrufer sperrt dann das Feld, statt mit einer leeren Liste weiterzumachen.
 *
 * @param {string} titelId
 * @returns {Promise<string[]>}
 */
export async function ladeTitelSchlagworte(titelId) {
	const res = await apiFetch(`/api/buecher/titel/${encodeURIComponent(titelId)}/schlagworte`);
	if (!res.ok) {
		throw new Error(`Schlagworte konnten nicht geladen werden (${res.status})`);
	}
	const antwort = await res.json();
	if (!Array.isArray(antwort?.schlagworte)) {
		throw new Error('Schlagworte: Antwort ohne Liste');
	}
	return antwort.schlagworte;
}

/**
 * Ersetzt die Schlagworte eines Titels (PUT /api/buecher/titel/{id}/schlagworte). Eine
 * leere Liste entfernt alle. Liefert sie in der gespeicherten Schreibweise zurück —
 * „fantasy" kommt als das vorhandene „Fantasy" wieder.
 *
 * @param {string} titelId
 * @param {string[]} schlagworte
 * @returns {Promise<string[]>}
 */
export async function setzeTitelSchlagworte(titelId, schlagworte) {
	const antwort = await apiPut(`/api/buecher/titel/${encodeURIComponent(titelId)}/schlagworte`, {
		schlagworte
	});
	return antwort?.schlagworte ?? schlagworte;
}
