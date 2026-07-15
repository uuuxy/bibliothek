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
	import {
		buecherLaden,
		echteKlassenLaden,
		buecherNachKlassenGruppieren,
		bestandsFarbe
	} from '$lib/startseiten_api.js';

	// --- Zustandsvariablen ---
	let viewMode = $state('suche');
	let searchQuery = $state('');
	let selectedZweig = $state('');
	let selectedJahrgang = $state('');
	let klasseSearchQuery = $state('');
	let selectedKlasse = $state('');
	let isKlasseDropdownOpen = $state(false);
	let selectedBook = $state(/** @type {any} */ (null)); // For Quick-Edit Drawer

	/** Navigate to the full-page book detail view */
	/** @param {any} book */
	function navigateToDetail(book) {
		appState.activeBookId = book.id;
		appState.selectedBook = book;
		window.history.pushState(null, '', `/katalog/buch/${book.id}`);
		// Signal App.svelte to switch to book_detail tab via popstate trick
		window.dispatchEvent(new PopStateEvent('popstate'));
	}

	/** @type {any[]} */
	let books = $state.raw([]);
	/** @type {any[]} */
	let realClasses = $state.raw([]);
	// --- Abgeleitete Werte ---
	let classes = $derived(buecherNachKlassenGruppieren(books));
	// Set: Das replace() bildet zwei Namen auf denselben Wert ab, sobald „Klasse 5a" und
	// „5a" nebeneinander existieren. Die Liste wird per Wert als each-Key genutzt —
	// doppelte Keys reissen die Ansicht ab (each_key_duplicate).
	let klassenList = $derived([...new Set(realClasses.map((c) => c.name.replace('Klasse ', '')))]);

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

	let filteredRealClasses = $derived(
		(Array.isArray(realClasses) ? realClasses : []).filter(
			(cls) => selectedKlasse === '' || cls.name === selectedKlasse
		)
	);

	let filteredKlassenList = $derived(
		klassenList.filter((k) => k.toLowerCase().includes(klasseSearchQuery.toLowerCase()))
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
			realClasses = await echteKlassenLaden();
			appState.guestAuthenticated = true;
		} catch {
			appState.guestAuthenticated = false;
		}
	}

	// Automatischer Reset der Auswahl, wenn das Suchfeld geleert wird
	$effect(() => {
		if (klasseSearchQuery === '' && selectedKlasse !== '') {
			selectedKlasse = '';
		}
	});

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

	// Wenn exakt eine Klasse getippt wurde, die existiert, diese auch selektieren
	$effect(() => {
		if (klasseSearchQuery !== '' && selectedKlasse === '') {
			const exactMatch = realClasses.find(
				(c) => c.name === klasseSearchQuery || c.name === `Klasse ${klasseSearchQuery}`
			);
			if (exactMatch) {
				selectedKlasse = exactMatch.name;
			}
		}
	});

	/**
	 * @param {string} klasse
	 */
	function selectKlasse(klasse) {
		selectedKlasse = klasse;
		klasseSearchQuery = klasse;
		isKlasseDropdownOpen = false;
	}
</script>

<div class="w-full text-slate-800 font-sans">
	<div class="w-full transition-all duration-300">
		<StartseitenFilter
			bind:viewMode
			bind:searchQuery
			bind:selectedZweig
			bind:selectedJahrgang
			bind:klasseSearchQuery
			bind:isKlasseDropdownOpen
			{filteredKlassenList}
			onSelectKlasse={selectKlasse}
		/>

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
							<button
								class="px-6 py-2 bg-white hover:bg-slate-50 text-slate-700 font-semibold rounded-full border border-slate-300 shadow-sm transition-all cursor-pointer"
								onclick={() => (displayLimit += 50)}
							>
								Mehr laden ({filteredBooks.length - displayLimit} weitere)
							</button>
						</div>
					{/if}
				</div>
			{:else if viewMode === 'jahrgaenge'}
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
			{:else}
				<div
					in:fade={{ duration: 300, delay: 150 }}
					out:fade={{ duration: 150 }}
					class="space-y-8"
					role="tabpanel"
					id="content-schulklassen"
					aria-labelledby="tab-schulklassen"
				>
					{#key selectedKlasse}
						<KlassenUebersichtStartseite
							filteredClasses={filteredRealClasses}
							getStockColor={bestandsFarbe}
							onBookClick={(book) => navigateToDetail(book)}
						/>
					{/key}
				</div>
			{/if}
		</main>
	</div>
</div>
