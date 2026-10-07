import { appState } from '../inventur/lib/store.svelte.js';
import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
import { uiStore } from './stores/uiStore.svelte.js';
import { apiFetch, extractApiError } from './apiFetch.js';
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
		return { kopf: null, fehler: res.status === 404 ? '' : await extractApiError(res) };
	} catch (err) {
		console.error('Fehler beim Laden des Buches:', err);
		return { kopf: null, fehler: 'Der Titel konnte nicht geladen werden (Netzwerkfehler).' };
	}
}

function useBookCover() {
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverFailed = $state(false);

	function reset(coverUrl, isbn) {
		const candidates = coverKandidaten(coverUrl, isbn);
		coverCandidates = candidates;
		currentCandidateIndex = 0;
		coverFailed = candidates.length === 0;
	}

	function onCoverError() {
		if (currentCandidateIndex < coverCandidates.length - 1) {
			currentCandidateIndex++;
		} else {
			coverFailed = true;
		}
	}

	function onCoverLoad(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		if (image.naturalWidth < 10 || image.naturalHeight < 10) onCoverError();
	}

	return {
		get coverSrc() {
			return coverCandidates[currentCandidateIndex] || '';
		},
		get coverFailed() {
			return coverFailed;
		},
		reset,
		onCoverError,
		onCoverLoad
	};
}

function useBookLists() {
	/** @type {any[]} */
	let borrowers = $state([]);
	/** @type {any[]} */
	let exemplare = $state([]);
	/** @type {any[]} */
	let history = $state([]);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	/** Listen, deren Abruf gescheitert ist — ihre Zahl ist keine Zahl, sondern ein Fragezeichen. */
	let fehlendeListen = $state(/** @type {string[]} */ ([]));

	function reset() {
		borrowers = [];
		exemplare = [];
		history = [];
		vormerkungen = [];
		fehlendeListen = [];
	}

	/**
	 * @param {string} id
	 * @param {() => boolean} isCurrent
	 */
	async function load(id, isCurrent) {
		const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
			apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: 'include' }),
			apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: 'include' })
		]);
		if (!isCurrent()) return;

		/** @type {[string, PromiseSettledResult<any>, (w: any[]) => void][]} */
		const listen = [
			['Ausleiher', bRes, (w) => (borrowers = w)],
			['Exemplare', eRes, (w) => (exemplare = w)],
			['Historie', hRes, (w) => (history = w)],
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
		get borrowers() {
			return borrowers;
		},
		get exemplare() {
			return exemplare;
		},
		set exemplare(v) {
			exemplare = v;
		},
		get history() {
			return history;
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
		reset,
		load
	};
}

function useBookActions(getBook, getExemplare) {
	async function deleteTitle(showToast, onBack) {
		const book = getBook();
		if (!book) return;
		const exemplare = getExemplare();
		if (!(await loeschenBestaetigen(`Titel mit allen ${exemplare.length} Exemplaren löschen?`)))
			return;
		try {
			const res = await apiFetch(`/api/buecher/titel/${book.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				if (showToast) showToast('Titel erfolgreich gelöscht', 'success');
				if (onBack) onBack();
			} else {
				const err = await res.json().catch((e) => {
					console.error('Fehler:', e);
					return {};
				});
				if (showToast) showToast(err.error || 'Fehler beim Löschen des Titels.', 'error');
			}
		} catch (e) {
			console.error('Titel löschen fehlgeschlagen:', e);
			if (showToast) showToast('Netzwerkfehler beim Löschen des Titels.', 'error');
		}
	}

	function editTitle() {
		const book = getBook();
		if (!book) return;
		appState.bookToEdit = book;
		appState.requestAdminView = true;
		uiStore.activeTab = 'media_catalog';
		appState.activeBookId = null;
	}

	return { deleteTitle, editTitle };
}

export function useBookAkte() {
	/** @type {any} */
	let book = $state(null);
	let activeTab = $state('ausleiher');
	let isLoading = $state(true);

	/** Der Kopf des Titels kam nicht an — im Unterschied zu „den Titel gibt es nicht". */
	let kopfFehler = $state('');

	// Sequenznummer wie in der Schülerakte (useStudentProfile) und im orderStore: Die
	// Buch-Akte bleibt beim Wechsel MONTIERT — die Omnibox setzt nur appState.activeBookId,
	// der Router hält `book_detail`. Zwei Titel kurz hintereinander geöffnet, und die
	// langsamere Antwort gewinnt.
	let laufNr = 0;

	const cover = useBookCover();
	const lists = useBookLists();
	const actions = useBookActions(
		() => book,
		() => lists.exemplare
	);

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
	 *   - „Gesamten Titel löschen" schickt `book.id` — also den ALTEN Titel, samt allen
	 *     Exemplaren, Ausleihen und offenen Forderungen. Die Rückfrage nannte dabei die
	 *     Exemplarzahl des NEUEN („ALLE 12 zugehörigen Exemplare").
	 *   - „Titel bearbeiten" öffnet den Editor auf dem alten Titel.
	 *
	 * @param {string} id
	 */
	async function loadAll(id) {
		const meine = ++laufNr;
		const isCurrent = () => meine === laufNr;
		isLoading = true;
		// Alles, was zum vorigen Titel gehört, geht mit ihm. Ein leerer Kopf ist die
		// ehrliche Antwort auf „konnte nicht geladen werden" — der alte Kopf ist eine
		// falsche.
		book = null;
		kopfFehler = '';
		lists.reset();

		// Der Kopf läuft durch eine LOKALE Variable, nie durch `book` zurück: Dieser Lauf
		// steht in einem $effect (BookAkte.svelte). Ein Effekt, der `book` schreibt und im
		// selben Atemzug wieder liest, abonniert es — und löst sich beim nächsten Anlass
		// mit seinem eigenen `book = null` endlos selbst aus, bis Svelte nach 1.000
		// Umläufen abbricht (effect_update_depth_exceeded) und isLoading hängen bleibt.
		/** @type {any} */
		let kopf = null;
		if (appState.selectedBook?.id === id) {
			kopf = appState.selectedBook;
		} else {
			const geholt = await holeKopf(id);
			// Erst prüfen, wenn der Kopf ganz gelesen ist: Zwischen der Antwort und ihrem Körper
			// kann ein jüngerer Titel schon stehen, und dieser Kopf läge über dessen Listen.
			if (!isCurrent()) return;
			kopf = geholt.kopf;
			kopfFehler = geholt.fehler;
		}
		book = kopf;

		cover.reset(kopf?.coverUrl, kopf?.isbn);

		await lists.load(id, isCurrent);

		if (isCurrent()) {
			isLoading = false;
		}
	}

	return {
		get book() {
			return book;
		},
		get borrowers() {
			return lists.borrowers;
		},
		get kopfFehler() {
			return kopfFehler;
		},
		get fehlendeListen() {
			return lists.fehlendeListen;
		},
		get exemplare() {
			return lists.exemplare;
		},
		set exemplare(v) {
			lists.exemplare = v;
		},
		get history() {
			return lists.history;
		},
		get vormerkungen() {
			return lists.vormerkungen;
		},
		set vormerkungen(v) {
			lists.vormerkungen = v;
		},
		get activeTab() {
			return activeTab;
		},
		set activeTab(v) {
			activeTab = v;
		},
		get isLoading() {
			return isLoading;
		},
		get coverSrc() {
			return cover.coverSrc;
		},
		get coverFailed() {
			return cover.coverFailed;
		},
		loadAll,
		deleteTitle: actions.deleteTitle,
		editTitle: actions.editTitle,
		onCoverError: cover.onCoverError,
		onCoverLoad: cover.onCoverLoad
	};
}
