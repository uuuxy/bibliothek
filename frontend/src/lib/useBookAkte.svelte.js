import { appState } from '../inventur/lib/store.svelte.js';
import { uiStore } from './stores/uiStore.svelte.js';
import { apiFetch } from './apiFetch.js';

/**
 * Liefert das geparste JSON eines erfüllten, erfolgreichen Promise.allSettled-Ergebnisses,
 * sonst ein leeres Array.
 * @param {PromiseSettledResult<any>} settled
 * @returns {Promise<any[]>}
 */
async function jsonOrEmpty(settled) {
	if (settled.status === 'fulfilled' && settled.value.ok) {
		return await settled.value.json();
	}
	return [];
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

	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverFailed = $state(false);

	async function loadAll(id) {
		isLoading = true;
		if (appState.selectedBook && appState.selectedBook.id === id) {
			book = appState.selectedBook;
		} else {
			try {
				const res = await apiFetch(`/api/books/${id}`, { credentials: 'include' });
				if (res.ok) book = await res.json();
			} catch (err) {
				console.error('Fehler beim Laden des Buches:', err);
			}
		}

		const candidates = [];
		if (book?.coverUrl) candidates.push(book.coverUrl);
		if (book?.isbn) {
			const clean = book.isbn.replace(/[- ]/g, '');
			candidates.push(
				`https://books.google.com/books/content?id=&vid=ISBN:${clean}&printsec=frontcover&img=1&zoom=1`
			);
			candidates.push(`https://covers.openlibrary.org/b/isbn/${clean}-L.jpg`);
		}
		coverCandidates = candidates;
		currentCandidateIndex = 0;
		coverFailed = candidates.length === 0;

		const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
			apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: 'include' }),
			apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: 'include' }),
			apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: 'include' })
		]);

		borrowers = await jsonOrEmpty(bRes);
		exemplare = await jsonOrEmpty(eRes);
		history = await jsonOrEmpty(hRes);
		vormerkungen = await jsonOrEmpty(vRes);
		isLoading = false;
	}

	async function deleteTitle(showToast, onBack) {
		if (!book) return;
		if (
			!confirm(
				`Achtung: Dies löscht diesen Titel und ALLE ${exemplare.length} zugehörigen Exemplare unwiderruflich. Fortfahren?`
			)
		)
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
				alert(err.error || 'Fehler beim Löschen des Titels.');
			}
		} catch (e) {
			alert('Netzwerkfehler beim Löschen des Titels.');
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
