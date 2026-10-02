import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen } from '../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from './store.svelte.js';

/** @typedef {import('./buch_speichern.js').VorhandenerTitel} VorhandenerTitel */

// Die Frage der Buchmaske an den eigenen Katalog: Gibt es zu dieser ISBN schon einen Titel?
// Gestellt wird sie bei der Eingabe der ISBN und noch einmal, wenn das Speichern ablehnt.

/**
 * Fragt vor dem Speichern, ob die ISBN schon ein Titel trägt — nach der Regel und mit der
 * Meldung der Dublettenkontrolle, die sonst erst das Speichern ablehnen ließe. Trägt sie
 * keiner, nennt der Katalog den Titel unter der anderen Länge der ISBN (andereForm): Den
 * lehnt das Speichern nicht ab, er ist ein Vorschlag.
 * @param {string} isbn
 * @returns {Promise<{ meldung: string, vorhanden: VorhandenerTitel, andereForm: boolean } | null>}
 * @throws {Error} wenn sich der Katalog nicht fragen ließ
 */
export async function vorhandenerTitel(isbn) {
	const res = await apiFetch(`/api/books/vorhanden?isbn=${encodeURIComponent(isbn)}`, {
		credentials: 'include'
	});
	if (!res.ok) throw new Error('Der Katalog ließ sich nicht nach dieser ISBN fragen.');
	const { data } = await res.json();
	if (data?.vorhanden) {
		return { meldung: data.meldung, vorhanden: data.vorhanden, andereForm: false };
	}
	if (data?.andereForm) {
		return { meldung: data.meldung, vorhanden: data.andereForm, andereForm: true };
	}
	return null;
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
 *
 * Steht die ISBN nur in der anderen Länge im Katalog (zehn- statt dreizehnstellig oder
 * umgekehrt), fragt die Maske, ob es dasselbe Buch ist — einmal je ISBN, das hält merker
 * fest. Bei „Anderes Buch" geht es in der Maske weiter.
 * @param {string} isbn
 * @param {{ anderesBuch?: string }} [merker] die ISBN, zu der „Anderes Buch" geantwortet wurde
 * @returns {Promise<boolean>} ob die ISBN vergeben ist oder die Maske zum vorhandenen Titel führt
 */
export async function frageWennVergeben(isbn, merker = {}) {
	let treffer;
	try {
		treffer = await vorhandenerTitel(isbn);
	} catch {
		showToast('Ob es diese ISBN schon gibt, ließ sich nicht prüfen.', 'warning');
		return false;
	}
	if (!treffer) return false;
	if (!treffer.andereForm) {
		await frageVorhandenenOeffnen(treffer.meldung, treffer.vorhanden);
		return true;
	}
	if (merker.anderesBuch === isbn) return false;
	const oeffnen = await bestaetigen({
		titel: 'Ist es dasselbe Buch?',
		text: `${treffer.meldung} Beim Öffnen werden die Eingaben dieser Maske verworfen.`,
		aktion: 'Titel öffnen',
		abbruch: 'Anderes Buch'
	});
	if (oeffnen) appState.bookToEdit = { id: treffer.vorhanden.id };
	else merker.anderesBuch = isbn;
	return oeffnen;
}
