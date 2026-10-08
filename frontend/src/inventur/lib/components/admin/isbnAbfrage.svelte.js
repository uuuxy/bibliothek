import { apiFetch } from '../../../../lib/apiFetch.js';
import { showToast } from '$lib/store.svelte.js';
import { frageWennVergeben } from '../../buch_vorhanden.js';

/** @typedef {{ text: string, fehler: boolean }} Ausgang */

const VON_HAND = 'Angaben von Hand eintragen oder später erneut abfragen.';
/** @type {Ausgang} */
const NICHTS_BEKANNT = {
	text: 'Zu dieser ISBN ist bei den Katalogdiensten nichts bekannt.',
	fehler: false
};
/** @type {Ausgang} */
const GESCHEITERT = { text: `Die ISBN-Abfrage ist fehlgeschlagen. ${VON_HAND}`, fehler: true };
/** Was eine Abfrage ohne Treffer bedeutet, je Status der Antwort.
 * @type {Record<number, Ausgang>} */
const AUSGAENGE = {
	400: { text: 'Die Nummer hat nicht die Form einer ISBN (10 oder 13 Stellen).', fehler: true },
	404: NICHTS_BEKANNT,
	502: { text: `Die Katalogdienste sind nicht erreichbar. ${VON_HAND}`, fehler: true }
};

/**
 * Welches Feld der Maske die Antwort der Katalogdienste füllt. Was sie nicht nennt, fehlt.
 * @param {any} daten
 * @returns {[string, any][]}
 */
function felderAus(daten) {
	const jahr = Number.parseInt(daten.jahr);
	// Die Stufe aus dem Titel: 0 heißt unbekannt, „von" und „bis" kommen zusammen oder gar nicht.
	const von = daten.jahrgangVon || undefined;
	const bis = daten.jahrgangBis || undefined;
	/** @type {[string, any][]} */
	const felder = [
		['title', daten.title],
		['untertitel', daten.subtitle],
		['author', daten.author],
		['verlag', daten.verlag],
		['erscheinungsjahr', jahr > 0 ? jahr : undefined],
		['coverUrl', daten.coverUrl],
		['subject', daten.subject],
		['jahrgangVon', bis && von],
		['jahrgangBis', von && bis]
	];
	return felder.filter(([, wert]) => wert !== undefined && wert !== '');
}

/** @param {any} wert */
const istLeer = (wert) => wert === null || wert === undefined || wert === '';

/**
 * Die ISBN-Abfrage einer Buchmaske, für jeden Weg der Eingabe dieselbe: erst der eigene
 * Katalog, dann die Katalogdienste. In einer neuen Maske führt eine ISBN, die schon ein Titel
 * trägt, zu ihm, statt Angaben für ein zweites Buch zu laden.
 * @param {() => any} maske liefert das Formular der Maske
 * @param {() => ({ lade: (isbn: string) => void } | undefined)} dnbVorschlag holt nach einem
 *   Treffer die Schlagworte der DNB dazu
 */
export function erzeugeIsbnAbfrage(maske, dnbVorschlag) {
	let aktiv = $state(false);
	// Der Ausgang einer Abfrage ohne Treffer bleibt stehen, bis neu gefragt oder die ISBN
	// geändert wird: Eine Meldung, die nach Sekunden verschwindet, sähe aus wie „nichts
	// bekannt", auch wenn nur der Dienst kurz fort war.
	/** @type {Ausgang | null} */
	let ausgang = $state(null);
	// Ein Klick auf den Knopf verlässt zugleich das Feld: Der zweite Auslöser schließt sich
	// dem laufenden Ablauf an, statt Katalog und Katalogdienste doppelt zu fragen.
	/** @type {Promise<boolean> | null} */
	let lauf = null;
	let aufWunsch = false;
	// Was die letzte Abfrage in welches Formular geschrieben hat, und zu welcher ISBN. Eine
	// andere ISBN nimmt es zurück, soweit niemand es geändert hat: Sonst stünden die Angaben
	// des ersten Buchs unter der Nummer des zweiten.
	/** @type {Record<string, { vorher: any, geschrieben: any }>} */
	let uebernommen = {};
	let uebernommenZu = '';
	/** @type {any} */
	let uebernommenIn = null;

	/** @param {any} formular */
	function nimmZurueck(formular) {
		if (formular === uebernommenIn) {
			for (const [feld, { vorher, geschrieben }] of Object.entries(uebernommen)) {
				if (formular[feld] === geschrieben) formular[feld] = vorher;
			}
		}
		uebernommen = {};
		uebernommenZu = '';
		uebernommenIn = formular;
	}

	/** @param {any} formular @param {string} feld @param {any} wert */
	function schreibe(formular, feld, wert) {
		const bisher = uebernommen[feld];
		const unberuehrt = bisher && formular[feld] === bisher.geschrieben;
		uebernommen[feld] = { vorher: unberuehrt ? bisher.vorher : formular[feld], geschrieben: wert };
		formular[feld] = wert;
	}

	/** @param {Ausgang} ergebnis */
	function melde(ergebnis) {
		ausgang = ergebnis;
		showToast(ergebnis.text, ergebnis.fehler ? 'error' : 'info');
	}

	/** @param {any} formular */
	async function holeAngaben(formular) {
		const isbn = formular.isbn;
		// Die ISBN hat sich während der Abfrage geändert, etwa durch einen zweiten Scan: Die
		// Antwort gehört zum vorigen Buch und stünde sonst unter der neuen Nummer.
		const veraltet = () => formular.isbn !== isbn;
		// Was jemand tippt, während die Dienste antworten, bleibt stehen.
		const beimBeginn = { ...formular };
		aktiv = true;
		ausgang = null;
		try {
			const antwort = await apiFetch(`/api/lookup/${encodeURIComponent(isbn)}`);
			if (veraltet()) return;
			if (!antwort.ok) return melde(AUSGAENGE[antwort.status] ?? GESCHEITERT);
			const daten = (await antwort.json()).data ?? {};
			if (veraltet()) return;
			const frei = felderAus(daten).filter(([feld]) => formular[feld] === beimBeginn[feld]);
			for (const [feld, wert] of frei) schreibe(formular, feld, wert);
			// Der Ladenpreis der DNB ist ein Vorschlag für den Listenpreis: Er füllt nur ein
			// leeres Feld. Was jemand eingetragen hat, bleibt.
			if (daten.preis > 0 && istLeer(formular.listenpreis)) {
				schreibe(formular, 'listenpreis', daten.preis);
			}
			uebernommenZu = isbn;
			if (!daten.title) return melde(NICHTS_BEKANNT);
			dnbVorschlag()?.lade(isbn);
			showToast(`Metadaten übernommen: ${daten.title}`, 'success');
		} catch (err) {
			console.error('Fehler beim Nachschlagen der ISBN', err);
			if (!veraltet()) melde(GESCHEITERT);
		} finally {
			aktiv = false;
		}
	}

	/**
	 * Fragt zu der ISBN, die beim Beginn im Formular steht. Ändert sie sich unterwegs, endet
	 * der Durchlauf ohne Angaben; der Ablauf fragt dann mit der neuen.
	 * @param {any} formular
	 * @returns {Promise<boolean>} ob die Maske nach einem vorhandenen Titel gefragt hat
	 */
	async function durchlauf(formular) {
		const isbn = formular.isbn;
		if (formular !== uebernommenIn || isbn !== uebernommenZu) nimmZurueck(formular);
		if (!formular.id && (await frageWennVergeben(isbn))) return true;
		if (formular.isbn === isbn && (aufWunsch || !formular.title)) await holeAngaben(formular);
		return false;
	}

	/**
	 * Ein zweiter Scan ersetzt die ISBN, während zur ersten noch gefragt wird, und seine
	 * Eingabetaste schließt sich dem laufenden Ablauf an. Der Ablauf endet deshalb erst, wenn
	 * zu der ISBN gefragt ist, die im Formular steht.
	 * @returns {Promise<boolean>} ob die Maske nach einem vorhandenen Titel gefragt hat
	 */
	async function ablauf() {
		const formular = maske();
		for (;;) {
			const isbn = formular.isbn;
			if (await durchlauf(formular)) return true;
			if (maske() !== formular || !formular.isbn || formular.isbn === isbn) return false;
		}
	}

	/** @param {boolean} wunsch true: der Knopf lädt auch, wenn schon ein Titel dasteht */
	function nachschlagen(wunsch) {
		aufWunsch ||= wunsch;
		lauf ??= ablauf().finally(() => {
			lauf = null;
			aufWunsch = false;
		});
		return lauf;
	}

	return {
		/** Läuft gerade eine Abfrage bei den Katalogdiensten? */
		get aktiv() {
			return aktiv;
		},
		get ausgang() {
			return ausgang;
		},
		/** Die ISBN wurde geändert: Der Satz zur vorigen gilt nicht mehr. */
		vergissAusgang() {
			ausgang = null;
		},
		nachschlagen,
		/**
		 * Wartet auf einen laufenden Ablauf: Wer speichert, braucht dessen Angaben.
		 * @returns {Promise<boolean>} ob dabei nach einem vorhandenen Titel gefragt wurde
		 */
		ruht: () => lauf ?? Promise.resolve(false)
	};
}
