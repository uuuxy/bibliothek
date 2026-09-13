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

export function useBookAkte() {
	/** @type {any} */
	let book = $state(null);
	/** @type {any[]} */
	let borrowers = $state([]);
	/** @type {any[]} */
	let exemplare = $state([]);
	/** @type {any[]} */
	let history = $state([]);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	let activeTab = $state('ausleiher');
	let isLoading = $state(true);

	/** Listen, deren Abruf gescheitert ist — ihre Zahl ist keine Zahl, sondern ein Fragezeichen. */
	let fehlendeListen = $state(/** @type {string[]} */ ([]));
	/** Der Kopf des Titels kam nicht an — im Unterschied zu „den Titel gibt es nicht". */
	let kopfFehler = $state('');

	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverFailed = $state(false);

	// Sequenznummer wie in der Schülerakte (useStudentProfile) und im orderStore: Die
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
	 *   - „Gesamten Titel löschen" schickt `book.id` — also den ALTEN Titel, samt allen
	 *     Exemplaren, Ausleihen und offenen Forderungen. Die Rückfrage nannte dabei die
	 *     Exemplarzahl des NEUEN („ALLE 12 zugehörigen Exemplare").
	 *   - „Titel bearbeiten" öffnet den Editor auf dem alten Titel.
	 *
	 * @param {string} id
	 */
	async function loadAll(id) {
		const meine = ++laufNr;
		isLoading = true;
		// Alles, was zum vorigen Titel gehört, geht mit ihm. Ein leerer Kopf ist die
		// ehrliche Antwort auf „konnte nicht geladen werden" — der alte Kopf ist eine
		// falsche.
		book = null;
		borrowers = [];
		exemplare = [];
		history = [];
		vormerkungen = [];
		fehlendeListen = [];
		kopfFehler = '';

		// Der Kopf läuft durch eine LOKALE Variable, nie durch `book` zurück: Dieser Lauf
		// steht in einem $effect (BookAkte.svelte). Ein Effekt, der `book` schreibt und im
		// selben Atemzug wieder liest, abonniert es — und löst sich beim nächsten Anlass
		// mit seinem eigenen `book = null` endlos selbst aus, bis Svelte nach 1.000
		// Umläufen abbricht (effect_update_depth_exceeded) und isLoading hängen bleibt.
		/** @type {any} */
		let kopf = null;
		if (appState.selectedBook && appState.selectedBook.id === id) {
			kopf = appState.selectedBook;
		} else {
			try {
				const res = await apiFetch(`/api/books/${id}`, { credentials: 'include' });
				if (meine !== laufNr) return; // ein jüngerer Titel ist schon unterwegs oder da
				kopf = res.ok ? await res.json() : null;
				// „Buch nicht gefunden" ist eine Aussage über den Bestand — die Ansicht
				// zeigt sie, wenn kein Kopf da ist. Ein 500 oder ein Netzfehler ist etwas
				// anderes: Das Buch GIBT es womöglich, es kam nur nichts an. Seit dem
				// 12.09.2026 steht der Unterschied in kopfFehler (Register,
				// Bestands-Durchgang 10.09.); nur der 404 bleibt „nicht gefunden".
				kopfFehler = res.ok || res.status === 404 ? '' : await extractApiError(res);
			} catch (err) {
				if (meine !== laufNr) return;
				kopfFehler = 'Der Titel konnte nicht geladen werden (Netzwerkfehler).';
				console.error('Fehler beim Laden des Buches:', err);
			}
		}
		book = kopf;

		const candidates = coverKandidaten(kopf?.coverUrl, kopf?.isbn);
		coverCandidates = candidates;
		currentCandidateIndex = 0;
		coverFailed = candidates.length === 0;

		const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
			apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: 'include' }),
			apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: 'include' })
		]);
		if (meine !== laufNr) return;

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
		isLoading = false;
	}

	async function deleteTitle(showToast, onBack) {
		if (!book) return;
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
		if (!book) return;
		appState.bookToEdit = book;
		appState.requestAdminView = true;
		uiStore.activeTab = 'media_catalog';
		appState.activeBookId = null;
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
		get book() {
			return book;
		},
		get borrowers() {
			return borrowers;
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
		get history() {
			return history;
		},
		get vormerkungen() {
			return vormerkungen;
		},
		set vormerkungen(v) {
			vormerkungen = v;
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
			return coverCandidates[currentCandidateIndex] || '';
		},
		get coverFailed() {
			return coverFailed;
		},
		loadAll,
		deleteTitle,
		editTitle,
		onCoverError,
		onCoverLoad
	};
}
