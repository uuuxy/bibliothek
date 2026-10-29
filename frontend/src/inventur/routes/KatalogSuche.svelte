<!--
	KatalogSuche.svelte (Medienkatalog → Suche & Filter)

	Suchfeld und Cover-Raster der Buch-Suche. Nach der Jahrgangsspanne gruppiert sie nicht:
	Fast jeder Titel trägt die Vorgabe des Imports (5–10), alle lägen in einer Gruppe. Welche
	Klasse welche Bücher hat, steht unter Bibliothek → Klassensätze.
-->
<script>
	import { onMount } from 'svelte';
	import { appState } from '$lib/store.svelte.js';
	import BuchRasterStartseite from '$lib/components/BuchRasterStartseite.svelte';
	import StartseitenFilter from '$lib/components/StartseitenFilter.svelte';
	import SuchZustand from '../../lib/components/ui/SuchZustand.svelte';
	import LadeFehler from '../../lib/components/ui/LadeFehler.svelte';
	import { fehlertext } from '../../lib/utils/fehlertext.js';
	import { Search } from '@lucide/svelte';
	import Button from '../../lib/components/ui/Button.svelte';
	import { hatRecht } from '../../lib/menue.js';
	import { authStore } from '../../lib/stores/authStore.svelte.js';
	import { buecherLaden, buecherSuchen, buecherJeBuch } from '$lib/startseiten_api.js';
	import {
		erzeugeExemplarFrage,
		hinweisOhneTreffer,
		mitTitelDesExemplars,
		satzZumExemplar
	} from '$lib/exemplarImKatalog.svelte.js';

	let suche = $state('');
	let gewaehlterTitel = $state(/** @type {any} */ (null)); // For Quick-Edit Drawer
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
	let buecher = $state.raw([]);
	let ladefehler = $state('');

	// Eine Exemplarnummer, gescannt oder getippt, steht in keiner Angabe eines Titels. Der Server
	// erkennt sie wie die Theke, auch über das Etikett von Littera; ihr Titel steht dann vorn.
	const exemplarFrage = erzeugeExemplarFrage();
	$effect(() => {
		exemplarFrage.frage(suche);
		return exemplarFrage.stoppe;
	});
	const exemplar = $derived(exemplarFrage.fuer(suche));
	const titelDesExemplars = $derived(
		exemplar.titelId ? (buecher.find((b) => b.id === exemplar.titelId) ?? null) : null
	);

	// Ein Buch in mehreren Auflagen ist eine Kachel (docs/OFFEN.md 4.18, Stufe 6).
	let gefilterteTitel = $derived(
		buecherJeBuch(buecher, mitTitelDesExemplars(titelDesExemplars, buecherSuchen(buecher, suche)))
	);

	let displayLimit = $state(50);
	let sichtbareTitel = $derived(gefilterteTitel.slice(0, displayLimit));

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		gefilterteTitel;
		displayLimit = 50;
	});

	onMount(() => {
		ladeDaten();
	});

	async function ladeDaten() {
		ladefehler = '';
		try {
			buecher = await buecherLaden();
		} catch (e) {
			// Ohne Liste sagte jede Suche „Keine Bücher gefunden"; ein gescheitertes Laden ist auch
			// keine Abmeldung (die meldet apiFetch bei 401).
			ladefehler = fehlertext(e);
		}
	}

	// Synchronize appState.selectedBook with local gewaehlterTitel
	$effect(() => {
		if (appState.selectedBook) {
			gewaehlterTitel = appState.selectedBook;
		}
	});
	$effect(() => {
		if (gewaehlterTitel === null) {
			appState.selectedBook = null;
		}
	});
</script>

<div class="w-full text-on-surface font-sans">
	<StartseitenFilter bind:suche />

	<div class="relative">
		<!-- „Nichts gefunden" muss zu sehen sein: Eine leere Fläche sieht aus wie eine Suche,
		     die gar nicht stattgefunden hat, beim Scannen also „die Kamera hat nichts gelesen"
		     statt „dieses Buch haben wir nicht". Der Hinweis wiederholt den Suchtext, bei einem
		     Scan den gelesenen Code. Form und Wortlaut wie im OPAC und im Kollegiums-Portal
		     (ui/SuchZustand); die M3-Seite zur Suche trifft dazu keine Aussage. -->
		<!-- Solange die Frage nach dem Exemplar läuft, steht noch kein „nichts gefunden". -->
		{#if ladefehler}
			<LadeFehler titel="Katalog nicht geladen" text={ladefehler} onerneut={ladeDaten} />
		{:else if suche.trim() !== '' && gefilterteTitel.length === 0 && exemplar.stand !== 'laeuft'}
			<SuchZustand
				symbol={Search}
				titel="Keine Bücher gefunden"
				hinweis={hinweisOhneTreffer(suche.trim(), exemplar)}
			/>
		{/if}
		{#if titelDesExemplars}
			<p class="pb-4 text-sm text-on-surface-variant">
				{satzZumExemplar(exemplar, titelDesExemplars)}
			</p>
		{/if}
		<BuchRasterStartseite
			titelListe={sichtbareTitel}
			onBookClick={(book) => navigateToDetail(book)}
			onEditClick={darfBearbeiten
				? (book) => {
						// Die Titel-Verwaltung holt den Titel und öffnet die Maske
						// (TitelVerwaltung: appState.bookToEdit) — unabhängig davon, ob sie in
						// dieser Sitzung schon einmal offen war.
						appState.bookToEdit = book;
						appState.requestAdminView = true;
					}
				: undefined}
		/>
		{#if displayLimit < gefilterteTitel.length}
			<div class="mt-8 flex justify-center">
				<Button variant="secondary" onclick={() => (displayLimit += 50)} class="px-6">
					Mehr laden ({gefilterteTitel.length - displayLimit} weitere)
				</Button>
			</div>
		{/if}
	</div>
</div>
