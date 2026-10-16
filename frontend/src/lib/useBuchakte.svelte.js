import { appState } from '../inventur/lib/store.svelte.js';
import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
import { uiStore } from './stores/uiStore.svelte.js';
import { apiFetch, leseApiFehler } from './apiFetch.js';
import { coverKandidaten } from './utils/coverSrc.js';

/**
 * Das geparste JSON eines erfüllten, erfolgreichen Ergebnisses — sonst `null`.
 *
 * Bewusst NICHT „sonst ein leeres Array" (Sweep „verschluckte Fehlantwort", 06.09.2026):
 * Ein leeres Array ist eine Aussage über den Bestand, und der Reiter schrieb sie hin —
 * „Ausleiher (0)" für einen Titel, der Ausleiher hat. `null` heißt „nicht geladen", und
 * das ist etwas anderes als „keine da".
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

function useBuchCover() {
	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverFehlt = $state(false);

	/** @param {string[]} kandidaten */
	function setzeKandidaten(kandidaten) {
		coverCandidates = kandidaten;
		currentCandidateIndex = 0;
		coverFehlt = kandidaten.length === 0;
	}

	function naechsteQuelle() {
		if (currentCandidateIndex < coverCandidates.length - 1) {
			currentCandidateIndex++;
		} else {
			coverFehlt = true;
		}
	}

	/** @param {Event} event */
	function pruefeGeladenes(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		if (image.naturalWidth < 10 || image.naturalHeight < 10) naechsteQuelle();
	}

	return {
		get coverQuelle() {
			return coverCandidates[currentCandidateIndex] || '';
		},
		get coverFehlt() {
			return coverFehlt;
		},
		setzeKandidaten,
		naechsteQuelle,
		pruefeGeladenes
	};
}

function useBuchAktionen() {
	/**
	 * @param {any} buch
	 * @param {any[]} exemplare
	 * @param {((msg: string, type?: import('./stores/snackbars.svelte.js').ToastTyp | undefined) => void)=} melde
	 * @param {(() => void)=} onBack
	 */
	async function loescheTitel(buch, exemplare, melde, onBack) {
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
			} else {
				const err = await res.json().catch((e) => {
					console.error('Fehler:', e);
					return {};
				});
				if (melde) melde(err.error || 'Fehler beim Löschen des Titels.', 'error');
			}
		} catch (e) {
			console.error('Titel löschen fehlgeschlagen:', e);
			if (melde) melde('Netzwerkfehler beim Löschen des Titels.', 'error');
		}
	}

	/** @param {any} buch */
	function bearbeiteTitel(buch) {
		if (!buch) return;
		appState.bookToEdit = buch;
		appState.requestAdminView = true;
		uiStore.activeTab = 'media_catalog';
		appState.activeBookId = null;
	}

	return { loescheTitel, bearbeiteTitel };
}

function useBuchListen() {
	/** @type {any[]} */
	let ausleiher = $state([]);
	/** @type {any[]} */
	let exemplare = $state([]);
	/** @type {any[]} */
	let verlauf = $state([]);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	/** @type {string[]} */
	let fehlendeListen = $state([]);

	function zuruecksetzen() {
		ausleiher = [];
		exemplare = [];
		verlauf = [];
		vormerkungen = [];
		fehlendeListen = [];
	}

	/**
	 * @param {string} id
	 * @param {() => boolean} istGueltig
	 */
	async function ladeListen(id, istGueltig) {
		const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
			apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: 'include' }),
			apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: 'include' })
		]);

		if (!istGueltig()) return;

		/** @type {[string, PromiseSettledResult<any>, (w: any[]) => void][]} */
		const listen = [
			['Ausleiher', bRes, (w) => (ausleiher = w)],
			['Exemplare', eRes, (w) => (exemplare = w)],
			['Historie', hRes, (w) => (verlauf = w)],
			['Vormerkungen', vRes, (w) => (vormerkungen = w)]
		];
		/** @type {string[]} */
		const fehlend = [];
		for (const [name, antwort, setze] of listen) {
			const daten = await jsonOderNull(antwort);
			setze(daten ?? []);
			if (daten === null) fehlend.push(name);
		}
		fehlendeListen = fehlend;
	}

	return {
		get ausleiher() {
			return ausleiher;
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
		get fehlendeListen() {
			return fehlendeListen;
		},
		zuruecksetzen,
		ladeListen
	};
}

export function useBuchakte() {
	/** @type {any} */
	let buch = $state(null);
	let aktiverReiter = $state('ausleiher');
	let laedt = $state(true);
	/** Der Kopf des Titels kam nicht an — im Unterschied zu „den Titel gibt es nicht". */
	let kopfFehler = $state('');

	const listen = useBuchListen();
	const cover = useBuchCover();
	const aktionen = useBuchAktionen();

	// Sequenznummer wie in der Schülerakte (useLeserakte) und im bestellStore: Die
	// Buch-Akte bleibt beim Wechsel MONTIERT — die Omnibox setzt nur appState.activeBookId,
	// der Router hält `book_detail`. Zwei Titel kurz hintereinander geöffnet, und die
	// langsamere Antwort gewinnt.
	let laufNr = 0;

	/**
	 * Lädt Kopf und alle vier Listen eines Titels.
	 *
	 * Rasterdurchgang 06.09.2026 (Fragen 5 und 11), Zwilling des Akten-Fundes von heute
	 * (529def4d): Bis hierher stand `if (res.ok) book = await res.json();` ohne `else`,
	 * und nichts wurde beim Wechsel zurückgesetzt. Scheiterte genau diese eine Anfrage
	 * (500, oder 429 vom Rate-Limiter — es sind fünf parallele Anfragen je Titel), blieb
	 * der Kopf des VORHER geöffneten Titels stehen, während die Reiter darunter schon zum
	 * neuen gehörten. Das ist hier nicht nur Anzeige:
	 *
	 *   - „Gesamten Titel löschen" schickt `buch.id` — also den ALTEN Titel, samt allen
	 *     Exemplaren, Ausleihen und offenen Forderungen. Die Rückfrage nannte dabei die
	 *     Exemplarzahl des NEUEN („ALLE 12 zugehörigen Exemplare").
	 *   - „Titel bearbeiten" öffnet den Editor auf dem alten Titel.
	 *
	 * @param {string} id
	 */
	async function ladeAlles(id) {
		const meine = ++laufNr;
		const istGueltig = () => meine === laufNr;

		laedt = true;
		buch = null;
		kopfFehler = '';
		listen.zuruecksetzen();

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
			if (!istGueltig()) return;
			kopf = geholt.kopf;
			kopfFehler = geholt.fehler;
		}
		buch = kopf;

		cover.setzeKandidaten(coverKandidaten(kopf?.coverUrl, kopf?.isbn));

		await listen.ladeListen(id, istGueltig);

		if (!istGueltig()) return;
		laedt = false;
	}

	return {
		get buch() {
			return buch;
		},
		get ausleiher() {
			return listen.ausleiher;
		},
		get kopfFehler() {
			return kopfFehler;
		},
		get fehlendeListen() {
			return listen.fehlendeListen;
		},
		get exemplare() {
			return listen.exemplare;
		},
		set exemplare(v) {
			listen.exemplare = v;
		},
		get verlauf() {
			return listen.verlauf;
		},
		get vormerkungen() {
			return listen.vormerkungen;
		},
		set vormerkungen(v) {
			listen.vormerkungen = v;
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
		get coverQuelle() {
			return cover.coverQuelle;
		},
		get coverFehlt() {
			return cover.coverFehlt;
		},
		ladeAlles,
		/**
		 * @param {((msg: string, type?: import('./stores/snackbars.svelte.js').ToastTyp | undefined) => void)=} melde
		 * @param {(() => void)=} onBack
		 */
		loescheTitel: (melde, onBack) => aktionen.loescheTitel(buch, listen.exemplare, melde, onBack),
		bearbeiteTitel: () => aktionen.bearbeiteTitel(buch),
		naechsteQuelle: cover.naechsteQuelle,
		pruefeGeladenes: cover.pruefeGeladenes
	};
}
