<!--
	+page.svelte (Medienkatalog → Suche & Filter)

	Suchfeld und Cover-Raster der Buch-Suche. Der Reiter „Jahrgänge" (Gruppierung nach der
	gepflegten Jahrgangsspanne) ist am 02.09.2026 weg: 13.055 von 13.060 Titeln trugen den
	unangetasteten Import-Default 5–10 und lagen in EINER Gruppe „Ohne genaue Zuordnung" —
	der Reiter hätte erst nach dem Nachpflegen von 13.000 Titeln etwas gezeigt. Welche
	Klasse welche Bücher hat, steht unter Bibliothek → Klassensätze.
-->
<script>
	import { onMount } from 'svelte';
	import { appState } from '$lib/store.svelte.js';
	import BuchRasterStartseite from '$lib/components/BuchRasterStartseite.svelte';
	import StartseitenFilter from '$lib/components/StartseitenFilter.svelte';
	import Button from '../../lib/components/ui/Button.svelte';
	import { hatRecht } from '../../lib/menu.js';
	import { authStore } from '../../lib/stores/authStore.svelte.js';
	import { buecherLaden, buecherSuchen } from '$lib/startseiten_api.js';

	let searchQuery = $state('');
	let selectedBook = $state(/** @type {any} */ (null)); // For Quick-Edit Drawer
	// Der Stift auf der Kachel gehört nur denen, die bearbeiten dürfen. Vorher sah ihn
	// jeder — und ohne Recht tat er dasselbe wie der Klick auf die Kachel.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	/** Navigate to the full-page book detail view */
	/** @param {any} book */
	function navigateToDetail(book) {
		appState.activeBookId = book.id;
		appState.selectedBook = book;
		window.history.pushState(null, '', `/medienkatalog/buch/${book.id}`);
		// Signal App.svelte to switch to book_detail tab via popstate trick
		window.dispatchEvent(new PopStateEvent('popstate'));
	}

	/** @type {any[]} */
	let books = $state.raw([]);
	let filteredBooks = $derived(buecherSuchen(books, searchQuery));

	let displayLimit = $state(50);
	let paginatedBooks = $derived(filteredBooks.slice(0, displayLimit));

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		filteredBooks;
		displayLimit = 50;
	});

	onMount(() => {
		ladeDaten();
	});

	async function ladeDaten() {
		try {
			books = await buecherLaden();
			appState.guestAuthenticated = true;
		} catch {
			appState.guestAuthenticated = false;
		}
	}

	// Synchronize appState.selectedBook with local selectedBook
	$effect(() => {
		if (appState.selectedBook) {
			selectedBook = appState.selectedBook;
		}
	});
	$effect(() => {
		if (selectedBook === null) {
			appState.selectedBook = null;
		}
	});
</script>

<div class="w-full text-slate-800 font-sans">
	<StartseitenFilter bind:searchQuery />

	<main class="relative">
		<BuchRasterStartseite
			filteredBooks={paginatedBooks}
			onBookClick={(book) => navigateToDetail(book)}
			onEditClick={darfBearbeiten
				? (book) => {
						// Die Titel-Verwaltung öffnet die Akte, sobald sie geladen hat
						// (admin/+page: appState.bookToEdit) — unabhängig davon, ob sie in
						// dieser Sitzung schon einmal offen war.
						appState.bookToEdit = book;
						appState.requestAdminView = true;
					}
				: undefined}
		/>
		{#if displayLimit < filteredBooks.length}
			<div class="mt-8 flex justify-center">
				<Button variant="secondary" onclick={() => (displayLimit += 50)} class="px-6">
					Mehr laden ({filteredBooks.length - displayLimit} weitere)
				</Button>
			</div>
		{/if}
	</main>
</div>
