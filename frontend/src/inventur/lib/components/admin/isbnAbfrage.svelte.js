import { apiFetch } from '../../../../lib/apiFetch.js';
import { showToast } from '$lib/store.svelte.js';
import { frageWennVergeben } from '../../buch_speichern.js';

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
	const jahr = parseInt(daten.jahr);
	const klasse = parseInt(daten.grade);
	/** @type {[string, any][]} */
	const felder = [
		['title', daten.title],
		['author', daten.author],
		['verlag', daten.verlag],
		['erscheinungsjahr', jahr > 0 ? jahr : undefined],
		['coverUrl', daten.coverUrl],
		['subject', daten.subject],
		['gradeLevel', Number.isNaN(klasse) ? undefined : klasse]
	];
	return felder.filter(([, wert]) => wert !== undefined && wert !== '');
}

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
	let laeuft = false;
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
		aktiv = true;
		ausgang = null;
		try {
			const antwort = await apiFetch(`/api/lookup/${encodeURIComponent(isbn)}`);
			if (!antwort.ok) return melde(AUSGAENGE[antwort.status] ?? GESCHEITERT);
			const daten = (await antwort.json()).data ?? {};
			for (const [feld, wert] of felderAus(daten)) schreibe(formular, feld, wert);
			uebernommenZu = isbn;
			if (!daten.title) return melde(NICHTS_BEKANNT);
			dnbVorschlag()?.lade(isbn);
			showToast(`Metadaten übernommen: ${daten.title}`, 'success');
		} catch (fehler) {
			console.error('Fehler beim Nachschlagen der ISBN', fehler);
			melde(GESCHEITERT);
		} finally {
			aktiv = false;
		}
	}

	/** @param {boolean} wunsch true: der Knopf lädt auch, wenn schon ein Titel dasteht */
	async function nachschlagen(wunsch) {
		aufWunsch ||= wunsch;
		if (laeuft) return;
		laeuft = true;
		try {
			const formular = maske();
			if (formular !== uebernommenIn || formular.isbn !== uebernommenZu) nimmZurueck(formular);
			if (!formular.id && (await frageWennVergeben(formular.isbn))) return;
			if (aufWunsch || !formular.title) await holeAngaben(formular);
		} finally {
			laeuft = false;
			aufWunsch = false;
		}
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
		nachschlagen
	};
}
