<!--
	+page.svelte (Startseite)

	Orchestriert die Gast-Ansicht der Inventur-App:
	Login, Navigation, Filterung und Anzeige der Bücher/Klassen.
	Die eigentliche Logik und UI-Blöcke sind in Unterkomponenten ausgelagert.
-->
<script>
	import { fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { appState } from '$lib/store.svelte.js';
	import BuchRasterStartseite from '$lib/components/BuchRasterStartseite.svelte';
	import BuchListeStartseite from '$lib/components/BuchListeStartseite.svelte';
	import KatalogFilterleiste from '$lib/components/KatalogFilterleiste.svelte';
	import KlassenUebersichtStartseite from '$lib/components/KlassenUebersichtStartseite.svelte';
	import StartseitenFilter from '$lib/components/StartseitenFilter.svelte';
	import Button from '../../lib/components/ui/Button.svelte';
	import { hatRecht } from '../../lib/menu.js';
	import { authStore } from '../../lib/stores/authStore.svelte.js';
	import {
		buecherLaden,
		buecherSuchen,
		buecherFiltern,
		buecherSortieren,
		leererFilter,
		fachOptionenAus,
		medientypOptionenAus,
		buecherNachKlassenGruppieren,
		klassenFiltern,
		zweigOptionenAus,
		jahrgangOptionenAus,
		bestandsFarbe
	} from '$lib/startseiten_api.js';

	// --- Zustandsvariablen ---
	let viewMode = $state('suche');
	let searchQuery = $state('');
	let selectedZweig = $state('');
	let selectedJahrgang = $state('');
	let selectedBook = $state(/** @type {any} */ (null)); // For Quick-Edit Drawer
	// Filter, Sortierung und Ansicht der Buch-Suche (KatalogFilterleiste).
	const suchFilter = $state(leererFilter());
	let sortierung = $state('');
	let ansicht = $state(/** @type {'karten'|'liste'} */ ('karten'));

	// Der Stift auf der Karte gehört nur denen, die bearbeiten dürfen. Vorher sah ihn
	// jeder — und ohne Recht tat er dasselbe wie der Klick auf die Karte.
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
	// --- Abgeleitete Werte ---
	// „Jahrgänge" gruppiert aus der Jahrgangsspanne der Bücher (Begründung in
	// startseiten_api.js). Die ECHTEN Klassensätze aus /api/class-books standen hier
	// bis zum 08.08.2026 als dritter Reiter daneben — dieselbe Liste wie unter
	// Verwaltung → Schulklassen. Aufgelöst.
	let classes = $derived(buecherNachKlassenGruppieren(books));

	let filteredBooks = $derived(
		buecherSortieren(buecherFiltern(buecherSuchen(books, searchQuery), suchFilter), sortierung)
	);

	let filteredClasses = $derived(klassenFiltern(classes, selectedZweig, selectedJahrgang));

	let displayLimit = $state(50);
	let paginatedBooks = $derived(filteredBooks.slice(0, displayLimit));

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		filteredBooks;
		displayLimit = 50;
	});

	// --- Initialisierung ---
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
	<div class="w-full transition-all duration-300">
		<StartseitenFilter
			bind:viewMode
			bind:searchQuery
			bind:selectedZweig
			bind:selectedJahrgang
			zweigOptionen={zweigOptionenAus(books)}
			jahrgangOptionen={jahrgangOptionenAus(classes)}
		/>

		<main class="relative">
			{#if viewMode === 'suche'}
				<div
					in:fade={{ duration: 300, delay: 150 }}
					out:fade={{ duration: 150 }}
					role="tabpanel"
					id="content-suche"
					aria-labelledby="tab-suche"
					class="space-y-4"
				>
					<KatalogFilterleiste
						filter={suchFilter}
						bind:sortierung
						bind:ansicht
						fachOptionen={fachOptionenAus(books)}
						jahrgangOptionen={jahrgangOptionenAus(classes)}
						zweigOptionen={zweigOptionenAus(books)}
						medientypOptionen={medientypOptionenAus(books)}
						treffer={filteredBooks.length}
						gesamt={books.length}
					/>
					{#if ansicht === 'liste'}
						<BuchListeStartseite
							filteredBooks={paginatedBooks}
							onBookClick={(book) => navigateToDetail(book)}
						/>
					{:else}
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
					{/if}
					{#if displayLimit < filteredBooks.length}
						<div class="mt-8 flex justify-center">
							<Button variant="secondary" onclick={() => (displayLimit += 50)} class="px-6">
								Mehr laden ({filteredBooks.length - displayLimit} weitere)
							</Button>
						</div>
					{/if}
				</div>
			{:else}
				<div
					in:fade={{ duration: 300, delay: 150 }}
					out:fade={{ duration: 150 }}
					role="tabpanel"
					id="content-jahrgaenge"
					aria-labelledby="tab-jahrgaenge"
				>
					<KlassenUebersichtStartseite
						{filteredClasses}
						getStockColor={bestandsFarbe}
						onBookClick={(book) => navigateToDetail(book)}
					/>
				</div>
			{/if}
		</main>
	</div>
</div>
