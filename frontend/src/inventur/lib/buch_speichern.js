import { apiFetch } from '../../lib/apiFetch.js';

/**
 * Die Ablehnung einer vergebenen ISBN (409) samt dem Titel, der sie trägt. Die Maske führt
 * damit zu ihm: Ein Titel ohne Exemplar steht in keiner Suche.
 */
export class DubletteFehler extends Error {
	/**
	 * @param {string} meldung
	 * @param {{ id: string, title: string, ohneExemplar: boolean }} vorhanden
	 */
	constructor(meldung, vorhanden) {
		super(meldung);
		this.vorhanden = vorhanden;
	}
}

/**
 * Legt den Titel der Maske an (ohne id) oder ändert ihn.
 * @param {any} formular
 * @returns {Promise<any>} der gespeicherte Titel
 */
export async function speichereBuch(formular) {
	const res = await apiFetch(formular.id ? `/api/books/${formular.id}` : '/api/books', {
		method: formular.id ? 'PUT' : 'POST',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			...formular,
			gradeLevel: Number(formular.gradeLevel),
			istLernmittel: !!formular.istLernmittel,
			// Keine Zahl im Feld heißt „nicht anfassen", nicht „null Exemplare": Number(undefined)
			// ist NaN und würde in JSON zu null.
			stock: Number.isFinite(Number(formular.stock)) ? Number(formular.stock) : undefined,
			lastCounted: formular.lastCounted || null
		})
	});
	if (!res.ok) {
		const fehler = await res.json().catch(() => null);
		const meldung = fehler?.error || fehler?.message || 'Speichern fehlgeschlagen';
		if (res.status === 409 && fehler?.vorhanden?.id) {
			throw new DubletteFehler(meldung, fehler.vorhanden);
		}
		throw new Error(meldung);
	}
	return (await res.json()).data;
}

/**
 * Steht ein Titel mit diesem Bestand in der gewählten Sicht der Titelliste? „Mit Exemplaren"
 * zeigt nur Titel mit Bestand, „Ohne Exemplare" die übrigen.
 * @param {number|string} bestand
 * @param {'mit'|'ohne'} sicht
 */
export function stehtInSicht(bestand, sicht) {
	return Number(bestand) > 0 === (sicht !== 'ohne');
}
