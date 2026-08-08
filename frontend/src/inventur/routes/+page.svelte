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
	import KlassenUebersichtStartseite from '$lib/components/KlassenUebersichtStartseite.svelte';
	import StartseitenFilter from '$lib/components/StartseitenFilter.svelte';
	import Button from '../../lib/components/ui/Button.svelte';
	import {
		buecherLaden,
		buecherNachKlassenGruppieren,
		bestandsFarbe
	} from '$lib/startseiten_api.js';

	// --- Zustandsvariablen ---
	let viewMode = $state('suche');
	let searchQuery = $state('');
	let selectedZweig = $state('');
	let selectedJahrgang = $state('');
	let selectedBook = $state(/** @type {any} */ (null)); // For Quick-Edit Drawer

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
	// „Jahrgänge" gruppiert aus den Buch-Metadaten (gradeLevel/track). Die ECHTEN
	// Klassensätze aus /api/class-books standen hier bis zum 08.08.2026 als dritter
	// Reiter daneben — dieselbe Liste wie unter Verwaltung → Schulklassen. Aufgelöst.
	let classes = $derived(buecherNachKlassenGruppieren(books));

	// --- WZ-Synonyme für Suchbegriffe auf der Startseite ---
	const suchSynonyme = new Map([
		['powi', 'politik'],
		['mathe', 'mathematik'],
		['eng', 'englisch'],
		['deu', 'deutsch'],
		['franz', 'französisch'],
		['bio', 'biologie'],
		['che', 'chemie'],
		['phy', 'physik'],
		['geo', 'geographie'],
		['info', 'informatik'],
		['lat', 'latein'],
		['span', 'spanisch'],
		['rel', 'religion'],
		['reli', 'religion']
	]);

	let filteredBooks = $derived(
		(Array.isArray(books) ? books : []).filter((/** @type {any} */ b) => {
			let q = searchQuery.toLowerCase().trim();
			if (q === '') return true;

			// Split search into terms and resolve synonyms
			let terms = q.split(/\s+/).map((term) => suchSynonyme.get(term) || term);

			// If query has a number, ignore words like "klasse", "kl" to prevent filtering out books
			// that don't have the word "klasse" in their title/metadata but do match the grade.
			const hasNumber = terms.some((t) => !isNaN(parseInt(t, 10)));
			if (hasNumber) {
				terms = terms.filter((t) => !['klasse', 'kl', 'kl.', 'jahrgang', 'jg', 'jg.'].includes(t));
			}

			// EVERY term must match AT LEAST ONE field in the book
			return terms.every((term) => {
				if (b.title && b.title.toLowerCase().includes(term)) return true;
				if (b.isbn && b.isbn.toLowerCase().includes(term)) return true;
				if (b.author && b.author.toLowerCase().includes(term)) return true;
				if (b.subject && b.subject.toLowerCase().includes(term)) return true;
				if (b.track && b.track.toLowerCase().includes(term)) return true;

				// Grade Level Matching (e.g. term "5" matches gradeLevel 5)
				if (b.gradeLevel && b.gradeLevel.toString() === term) return true;

				// Grade Range Matching (e.g. term "6" matches range 5-10)
				const num = parseInt(term, 10);
				if (
					!isNaN(num) &&
					b.jahrgangVon &&
					b.jahrgangBis &&
					num >= b.jahrgangVon &&
					num <= b.jahrgangBis
				) {
					return true;
				}

				return false;
			});
		})
	);

	let filteredClasses = $derived(
		(Array.isArray(classes) ? classes : []).filter((cls) => {
			const zw =
				selectedZweig === '' || cls.books.some((/** @type {any} */ b) => b.track === selectedZweig);
			const jg = selectedJahrgang === '' || cls.name.includes(`Klasse ${selectedJahrgang}`);
			return zw && jg;
		})
	);

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
		<StartseitenFilter bind:viewMode bind:searchQuery bind:selectedZweig bind:selectedJahrgang />

		<main class="relative">
			{#if viewMode === 'suche'}
				<div
					in:fade={{ duration: 300, delay: 150 }}
					out:fade={{ duration: 150 }}
					role="tabpanel"
					id="content-suche"
					aria-labelledby="tab-suche"
				>
					<BuchRasterStartseite
						filteredBooks={paginatedBooks}
						onBookClick={(book) => navigateToDetail(book)}
						onEditClick={(book) => {
							if (appState.adminAuthenticated) {
								appState.bookToEdit = book;
								appState.requestAdminView = true;
							} else {
								navigateToDetail(book);
							}
						}}
					/>
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
					class="space-y-8"
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
