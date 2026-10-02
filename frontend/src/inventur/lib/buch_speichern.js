import { apiFetch } from '../../lib/apiFetch.js';
import { holeBuchDetail } from './admin_api.js';

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
