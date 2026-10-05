import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen, fragen } from '../../lib/stores/bestaetigung.svelte.js';
import { DubletteFehler, speichereBuch } from './buch_speichern.js';
import { appState, showToast } from './store.svelte.js';

/** @typedef {import('./buch_speichern.js').VorhandenerTitel} VorhandenerTitel */

// Die Frage der Buchmaske an den eigenen Katalog: Gibt es zu dieser ISBN schon einen Titel?
// Gestellt wird sie bei der Eingabe der ISBN und noch einmal, wenn das Speichern ablehnt.

/**
 * Fragt vor dem Speichern, ob die ISBN schon ein Titel trägt — nach der Regel und mit der
 * Meldung der Dublettenkontrolle, die sonst erst das Speichern ablehnen ließe. Die zehn- und
 * die dreizehnstellige Form einer ISBN sind für den Katalog dieselbe Nummer.
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
 * Ohne ISBN heißt ein vorhandener Titel gleich: Die Maske fragt, ob es dasselbe Medium ist.
 * Hefte einer Zeitschrift und Bände eines Werks tragen denselben Titel und Autor, deshalb
 * lehnt sie nicht ab. „Titel öffnen" führt zum vorhandenen, dort kommt das Exemplar dazu;
 * wer den Dialog nur schließt, hat nicht geantwortet, und nichts wird angelegt.
 * @param {string} meldung
 * @param {VorhandenerTitel} vorhanden
 * @returns {Promise<boolean>} ob der Titel als anderes Medium angelegt werden soll
 */
async function frageGleichenTitel(meldung, vorhanden) {
	const oeffnen = await fragen({
		titel: 'Ist es dasselbe Medium?',
		text: `${meldung} Beim Öffnen werden die Eingaben dieser Maske verworfen.`,
		aktion: 'Titel öffnen',
		abbruch: 'Anderes Medium'
	});
	if (oeffnen) appState.bookToEdit = { id: vorhanden.id };
	return oeffnen === false;
}

/**
 * Speichert den Titel der Maske. Heißt beim Anlegen ohne ISBN ein vorhandener gleich, fragt
 * sie und schickt denselben Titel nach „Anderes Medium" noch einmal.
 * @param {any} formular
 * @returns {Promise<any | null>} der gespeicherte Titel; null, wenn die Frage zum vorhandenen
 *   Titel führte oder ohne Antwort blieb
 */
export async function speichereMitFrage(formular) {
	try {
		return await speichereBuch(formular);
	} catch (err) {
		if (!(err instanceof DubletteFehler && err.gleicherTitel) || formular.id) throw err;
		if (!(await frageGleichenTitel(err.message, err.vorhanden))) return null;
		return speichereBuch(formular, { anderesMedium: true });
	}
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
