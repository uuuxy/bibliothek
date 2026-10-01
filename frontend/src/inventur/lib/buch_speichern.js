import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen } from '../../lib/stores/bestaetigung.svelte.js';
import { holeBuchDetail } from './admin_api.js';
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
 * Die Ablehnung einer Bestandsänderung (409): Die Maske hat einen anderen Bestand gesehen,
 * als jetzt am Server steht. Mit bestand stellt sie ihr Feld nach.
 */
export class BestandVeraltetFehler extends Error {
	/**
	 * @param {string} meldung
	 * @param {number} bestand
	 */
	constructor(meldung, bestand) {
		super(meldung);
		this.bestand = bestand;
	}
}

/**
 * Was die Maske dem Server zum Bestand sagt. Ein vorhandener Titel nennt ihn nur, wenn das
 * Feld eine andere Zahl zeigt als beim Öffnen (stockGesehen), und schickt beide Zahlen: Der
 * Server ändert nur, wenn die gesehene noch gilt. Ein leeres Feld heißt „nicht anfassen",
 * nicht „null Exemplare" — Number(null) ist 0.
 * @param {any} formular
 * @returns {{ stock?: number, stockGesehen?: number }}
 * @throws {Error} wenn im Feld keine ganze Zahl ab 0 steht
 */
export function bestandsAngabe(formular) {
	const feld = formular.stock;
	if (feld === null || feld === undefined || feld === '') return {};
	const soll = Number(feld);
	if (!Number.isInteger(soll) || soll < 0) {
		throw new Error('Der Bestand muss eine ganze Zahl ab 0 sein.');
	}
	if (!formular.id) return { stock: soll };
	if (soll === formular.stockGesehen) return {};
	return { stock: soll, stockGesehen: formular.stockGesehen };
}

/**
 * Die Rückfrage vor dem Verringern, gemessen an der Zahl vom Öffnen der Maske. Die
 * Titelliste taugt dafür nicht: Ein Titel, den die Suche ausblendet, steht nicht in ihr.
 * @param {any} formular
 * @returns {{ titel: string, text: string, aktion: string, gefaehrlich: boolean } | null}
 */
export function verringernRueckfrage(formular) {
	const { stock, stockGesehen } = bestandsAngabe(formular);
	if (stock === undefined || stockGesehen === undefined || stock >= stockGesehen) return null;
	const weg = stockGesehen - stock;
	const wer =
		weg === 1
			? 'Ein Exemplar wird ausgesondert und steht'
			: `${weg} Exemplare werden ausgesondert und stehen`;
	return {
		titel: `Bestand von ${stockGesehen} auf ${stock} verringern?`,
		text: `${wer} im Abgangsbuch unter „Bestandskorrektur“. Welche es trifft, wählt das Programm: zuerst nicht ausgeliehene.`,
		aktion: 'Verringern',
		gefaehrlich: true
	};
}

/**
 * Nach einer Änderung an den Exemplaren aus der Maske heraus (eines gelöscht) holt sie den
 * Bestand neu. Stand das Feld noch auf der gesehenen Zahl, zeigt es die neue; eine getippte
 * Zahl bleibt stehen. Scheitert das Nachladen, meldet erst das Speichern einer geänderten
 * Zahl den neuen Stand.
 * @param {any} formular
 */
export async function ladeBestandNach(formular) {
	const unberuehrt = formular.stock === formular.stockGesehen;
	try {
		const { stock } = await holeBuchDetail(formular.id);
		formular.stockGesehen = stock;
		if (unberuehrt) formular.stock = stock;
	} catch {
		// Die Zahlen bleiben, wie sie sind.
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
			lastCounted: formular.lastCounted || null,
			// Der Bestand geht nur über bestandsAngabe hinaus, nie als Feld der Maske.
			stock: undefined,
			stockGesehen: undefined,
			...bestandsAngabe(formular)
		})
	});
	if (!res.ok) {
		const fehler = await res.json().catch(() => null);
		const meldung = fehler?.error || fehler?.message || 'Speichern fehlgeschlagen';
		if (res.status === 409 && fehler?.vorhanden?.id) {
			throw new DubletteFehler(meldung, fehler.vorhanden);
		}
		if (res.status === 409 && Number.isInteger(fehler?.bestand)) {
			throw new BestandVeraltetFehler(meldung, fehler.bestand);
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
