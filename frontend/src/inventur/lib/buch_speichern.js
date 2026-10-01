import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen } from '../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from './store.svelte.js';

/** @typedef {{ id: string, title: string, ohneExemplar: boolean }} VorhandenerTitel */

/**
 * Die Ablehnung einer vergebenen ISBN (409) samt dem Titel, der sie trägt. Die Maske führt
 * damit zu ihm: Ein Titel ohne Exemplar steht in keiner Suche.
 */
export class DubletteFehler extends Error {
	/**
	 * @param {string} meldung
	 * @param {VorhandenerTitel} vorhanden
	 */
	constructor(meldung, vorhanden) {
		super(meldung);
		this.vorhanden = vorhanden;
	}
}

/**
 * Fragt vor dem Speichern, ob die ISBN schon ein Titel trägt — nach der Regel und mit der
 * Meldung der Dublettenkontrolle, die sonst erst das Speichern ablehnen ließe.
 * @param {string} isbn
 * @returns {Promise<{ meldung: string, vorhanden: VorhandenerTitel } | null>}
 * @throws {Error} wenn sich der Katalog nicht fragen ließ
 */
export async function vorhandenerTitel(isbn) {
	const res = await apiFetch(`/api/books/vorhanden?isbn=${encodeURIComponent(isbn)}`, {
		credentials: 'include'
	});
	if (!res.ok) throw new Error('Der Katalog ließ sich nicht nach dieser ISBN fragen.');
	const { data } = await res.json();
	return data?.vorhanden ? { meldung: data.meldung, vorhanden: data.vorhanden } : null;
}

/**
 * Die ISBN trägt schon ein Titel: Die Maske fragt und führt zu ihm, dort kommt das Exemplar
 * dazu. Die Seite öffnet ihn über appState.bookToEdit.
 * @param {string} meldung
 * @param {VorhandenerTitel} vorhanden
 */
export async function frageVorhandenenOeffnen(meldung, vorhanden) {
	const oeffnen = await bestaetigen({
		titel: 'Vorhandenen Titel öffnen?',
		text: `${meldung} Die Eingaben dieser Maske werden dabei verworfen.`,
		aktion: 'Titel öffnen'
	});
	if (oeffnen) appState.bookToEdit = { id: vorhanden.id };
}

/**
 * Neue Maske, die ISBN steht im Feld: Trägt sie schon ein Titel, fragt die Maske sofort und
 * nicht erst beim Speichern, wenn alles eingetragen ist. Lässt sich der Katalog nicht fragen,
 * geht es mit einer Meldung weiter; das Speichern prüft dieselbe Regel noch einmal.
 * @param {string} isbn
 * @returns {Promise<boolean>} ob die ISBN vergeben ist
 */
export async function frageWennVergeben(isbn) {
	let treffer;
	try {
		treffer = await vorhandenerTitel(isbn);
	} catch {
		showToast('Ob es diese ISBN schon gibt, ließ sich nicht prüfen.', 'warning');
		return false;
	}
	if (!treffer) return false;
	await frageVorhandenenOeffnen(treffer.meldung, treffer.vorhanden);
	return true;
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
