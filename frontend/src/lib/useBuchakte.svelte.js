import { appState } from '../inventur/lib/store.svelte.js';
import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
import { uiStore } from './stores/uiStore.svelte.js';
import { apiFetch, leseApiFehler } from './apiFetch.js';

/**
 * Das geparste JSON eines erfüllten, erfolgreichen Ergebnisses — sonst `null`.
 *
 * Nicht „sonst ein leeres Array": Ein leeres Array ist eine Aussage über den Bestand, und
 * der Reiter schriebe sie hin („Ausleiher (0)" für einen Titel, der Ausleiher hat). `null`
 * heißt „nicht geladen", und das ist etwas anderes als „keine da".
 *
 * @param {PromiseSettledResult<any>} settled
 * @returns {Promise<any[] | null>}
 */
async function jsonOderNull(settled) {
	if (settled.status === 'fulfilled' && settled.value.ok) {
		return await settled.value.json();
	}
	return null;
}

/**
 * Holt den Kopf eines Titels vom Server. Ein 404 heißt „den Titel gibt es nicht" und trägt
 * keinen Fehlertext. Jede andere Fehlantwort und ein Netzfehler tragen einen: Den Titel gibt
 * es dann womöglich, es kam nur nichts an.
 *
 * @param {string} id
 * @returns {Promise<{ kopf: any, fehler: string }>}
 */
async function holeKopf(id) {
	try {
		const res = await apiFetch(`/api/books/${id}`, { credentials: 'include' });
		if (res.ok) return { kopf: await res.json(), fehler: '' };
		return { kopf: null, fehler: res.status === 404 ? '' : await leseApiFehler(res) };
	} catch (err) {
		console.error('Fehler beim Laden des Buches:', err);
		return { kopf: null, fehler: 'Der Titel konnte nicht geladen werden (Netzwerkfehler).' };
	}
}

export function useBuchakte() {
	/** @type {any} */
	let buch = $state(null);
	/** @type {any[]} */
	let ausleiher = $state([]);
	/** @type {any[]} */
	let exemplare = $state([]);
	/** @type {any[]} */
	let verlauf = $state([]);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	let aktiverReiter = $state('ausleiher');
	let laedt = $state(true);

	/** Listen, deren Abruf gescheitert ist — ihre Zahl ist keine Zahl, sondern ein Fragezeichen. */
	let fehlendeListen = $state(/** @type {string[]} */ ([]));
	/** Der Kopf des Titels kam nicht an — im Unterschied zu „den Titel gibt es nicht". */
	let kopfFehler = $state('');

	// Sequenznummer wie in der Schülerakte (useLeserakte) und im bestellStore: Die
	// Buch-Akte bleibt beim Wechsel MONTIERT — die Omnibox setzt nur appState.activeBookId,
	// der Router hält `book_detail`. Zwei Titel kurz hintereinander geöffnet, und die
	// langsamere Antwort gewinnt.
	let laufNr = 0;

	/**
	 * Lädt Kopf und alle vier Listen eines Titels.
	 *
	 * Beim Wechsel wird alles zurückgesetzt, und eine gescheiterte Anfrage (500, oder 429 vom
	 * Rate-Limiter bei fünf parallelen Anfragen je Titel) lässt keinen alten Kopf stehen. Das
	 * ist nicht nur Anzeige: Unter dem Kopf des vorigen Titels schickte „Gesamten Titel
	 * löschen" dessen `buch.id`, samt allen Exemplaren, Ausleihen und offenen Forderungen,
	 * und „Titel bearbeiten" öffnete den Editor auf dem falschen Titel.
	 *
	 * @param {string} id
	 */
	async function ladeAlles(id) {
		const meine = ++laufNr;
		laedt = true;
		// Alles, was zum vorigen Titel gehört, geht mit ihm. Ein leerer Kopf ist die
		// ehrliche Antwort auf „konnte nicht geladen werden" — der alte Kopf ist eine
		// falsche.
		buch = null;
		ausleiher = [];
		exemplare = [];
		verlauf = [];
		vormerkungen = [];
		fehlendeListen = [];
		kopfFehler = '';

		// Der Kopf läuft durch eine LOKALE Variable, nie durch `buch` zurück: Dieser Lauf
		// steht in einem $effect (Buchakte.svelte). Ein Effekt, der `buch` schreibt und im
		// selben Atemzug wieder liest, abonniert es — und löst sich beim nächsten Anlass
		// mit seinem eigenen `buch = null` endlos selbst aus, bis Svelte nach 1.000
		// Umläufen abbricht (effect_update_depth_exceeded) und laedt hängen bleibt.
		/** @type {any} */
		let kopf = null;
		if (appState.selectedBook?.id === id) {
			kopf = appState.selectedBook;
		} else {
			const geholt = await holeKopf(id);
			// Erst prüfen, wenn der Kopf ganz gelesen ist: Zwischen der Antwort und ihrem Körper
			// kann ein jüngerer Titel schon stehen, und dieser Kopf läge über dessen Listen.
			if (meine !== laufNr) return;
			kopf = geholt.kopf;
			kopfFehler = geholt.fehler;
		}
		buch = kopf;

		const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
			apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: 'include' }),
			apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: 'include' })
		]);
		if (meine !== laufNr) return;

		/** @type {[string, PromiseSettledResult<any>, (w: any[]) => void][]} */
		const listen = [
			['Ausleiher', bRes, (w) => (ausleiher = w)],
			['Exemplare', eRes, (w) => (exemplare = w)],
			['Historie', hRes, (w) => (verlauf = w)],
			['Vormerkungen', vRes, (w) => (vormerkungen = w)]
		];
		// Wie beim Kopf erst prüfen, wenn die Körper gelesen sind: Ist ein jüngerer Titel
		// inzwischen ganz geladen, lägen diese Listen sonst unter seinem Kopf.
		const daten = await Promise.all(listen.map(([, antwort]) => jsonOderNull(antwort)));
		if (meine !== laufNr) return;
		/** @type {string[]} */
		const fehlend = [];
		listen.forEach(([name, , setze], i) => {
			setze(daten[i] ?? []);
			if (daten[i] === null) fehlend.push(name);
		});
		fehlendeListen = fehlend;
		laedt = false;
	}

	async function loescheTitel(melde, onBack) {
		if (!buch) return;
		if (!(await loeschenBestaetigen(`Titel mit allen ${exemplare.length} Exemplaren löschen?`)))
			return;
		try {
			const res = await apiFetch(`/api/buecher/titel/${buch.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				if (melde) melde('Titel erfolgreich gelöscht', 'success');
				if (onBack) onBack();
			} else if (melde) {
				melde(await leseApiFehler(res, 'Fehler beim Löschen des Titels.'), 'error');
			}
		} catch (e) {
			console.error('Titel löschen fehlgeschlagen:', e);
			if (melde) melde('Netzwerkfehler beim Löschen des Titels.', 'error');
		}
	}

	function bearbeiteTitel() {
		if (!buch) return;
		appState.bookToEdit = buch;
		appState.requestAdminView = true;
		uiStore.activeTab = 'media_catalog';
		appState.activeBookId = null;
	}

	return /** @satisfies {object} */ ({
		get buch() {
			return buch;
		},
		get ausleiher() {
			return ausleiher;
		},
		get kopfFehler() {
			return kopfFehler;
		},
		get fehlendeListen() {
			return fehlendeListen;
		},
		get exemplare() {
			return exemplare;
		},
		set exemplare(v) {
			exemplare = v;
		},
		get verlauf() {
			return verlauf;
		},
		get vormerkungen() {
			return vormerkungen;
		},
		set vormerkungen(v) {
			vormerkungen = v;
		},
		get aktiverReiter() {
			return aktiverReiter;
		},
		set aktiverReiter(v) {
			aktiverReiter = v;
		},
		get laedt() {
			return laedt;
		},
		ladeAlles,
		loescheTitel,
		bearbeiteTitel
	});
}
