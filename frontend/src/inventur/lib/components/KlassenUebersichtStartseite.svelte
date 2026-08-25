<script>
	import KlassenBuchKachelStartseite from '$lib/components/KlassenBuchKachelStartseite.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import { ChevronDown } from '@lucide/svelte';
	import Button from '../../../lib/components/ui/Button.svelte';

	/**
	 * onBookClick ist optional: Das Lehrerportal zeigt dieselben Gruppen als reine
	 * Lese-Sicht — ohne Klickziel bleiben die Kacheln dort auch keins.
	 * @type {{
	 *   filteredClasses: any[],
	 *   getStockColor: (stock: number) => string,
	 *   onBookClick?: (book: any) => void,
	 *   kompakt?: boolean
	 * }}
	 * kompakt: Listenzeile statt Seitenüberschrift (Kollegiums-Portal, wo der Jahrgang
	 * unter einem Reiter steht und keine 24 px trägt).
	 */
	let { filteredClasses, getStockColor, onBookClick = undefined, kompakt = false } = $props();
	const zeilenTitel = $derived(
		kompakt
			? 'truncate text-base font-medium text-on-surface'
			: 'truncate font-sans text-2xl font-bold text-slate-800'
	);
	const zaehlerChip = $derived(
		kompakt
			? 'bg-secondary-container text-on-secondary-container shrink-0 rounded-full px-3 py-0.5 text-xs font-semibold tabular-nums'
			: 'shrink-0 rounded-lg border border-blue-100 bg-blue-50 px-3 py-1 text-sm font-semibold text-blue-600'
	);

	// Immer nur ein Jahrgang offen — wie unter Verwaltung → Schulklassen. Die Wahl
	// ueberlebt einen Filterwechsel absichtlich: Wer den Filter wieder leert, findet
	// die Stelle offen vor, an der er war.
	let offenerJahrgang = $state('');

	// Seit die Gruppen aus der Jahrgangsspanne entstehen, hält die Sammelgruppe „Ohne
	// genaue Zuordnung" leicht tausende Titel — alle auf einmal zu rendern legte die
	// Seite lahm. Aufgeklappt erscheint deshalb portionsweise, wie in der Buch-Suche
	// nebenan (displayLimit in +page.svelte).
	const ANZEIGE_SCHRITT = 96;
	let anzeigeLimit = $state(ANZEIGE_SCHRITT);

	/**
	 * @param {any[]} books
	 */
	function sortBooks(books) {
		return [...books].sort(sortBooksBySubjectAndTitle);
	}
</script>

<!-- Ausklappbare Liste statt waagerechtem Karussell (10.08.2026) — derselbe Umbau wie am
     08.08. unter Verwaltung → Schulklassen, hier war die zweite Fundstelle uebersehen
     worden. Das Karussell hatte drei Fehler auf einmal:

     1. Die Kacheln lagen 176 px breit nebeneinander (w-40 + gap-4). Ein Jahrgang mit
        sechzehn Buechern brauchte 2.816 px; sichtbar war rund die Haelfte.
     2. Die Blaetterpfeile standen per CSS auf opacity:0 und erschienen erst bei :hover —
        am Tablet also nie.
     3. Der Verlauf ueber den Raendern war rgba(9,9,11,.95): ein fast schwarzer Schleier
        auf weissem Grund, uebrig aus einem dunklen Entwurf.

     M3 kennt ein Carousel, meint damit aber das STOEBERN in Bildmaterial. Hier sucht
     jemand ein bestimmtes Buch eines Jahrgangs — dafuer ist die Liste mit umbrechendem
     Raster die richtige Form, und die Anzahl steht schon in der zugeklappten Zeile. -->
{#each filteredClasses as cls (cls.name)}
	{@const alleine = filteredClasses.length === 1}
	{@const offen = alleine || cls.name === offenerJahrgang}
	{@const rasterID = `jahrgang-${cls.name.replace(/\s+/g, '-')}`}

	<section class="border-b border-slate-200 last:border-b-0">
		{#if alleine}
			<!-- Genau ein Jahrgang uebrig: Die Liste ist keine Auswahl mehr, sondern eine
			     Ueberschrift. Ein Schalter, der nur zwischen "Inhalt" und "leere Seite"
			     umlegt, waere ein Bedienelement ohne Aussage — also gibt es hier keinen. -->
			<h2 class="flex items-center gap-3 py-3">
				<span class={zaehlerChip}
					>{cls.books.length}
					{cls.books.length === 1 ? 'Buch' : 'Bücher'}</span
				>
				<span class={zeilenTitel}>{cls.name}</span>
			</h2>
		{:else}
			<!-- Die ganze Zeile schaltet um, nicht nur das Dreieck: Das Ziel ist so gross
			     wie die Aussage, die es betrifft. -->
			<button
				type="button"
				onclick={() => {
					offenerJahrgang = offen ? '' : cls.name;
					anzeigeLimit = ANZEIGE_SCHRITT;
				}}
				aria-expanded={offen}
				aria-controls={rasterID}
				class="flex w-full min-w-0 items-center gap-3 rounded-lg py-3 text-left"
			>
				<ChevronDown
					class="h-5 w-5 shrink-0 text-slate-500 transition-transform {offen ? '' : '-rotate-90'}"
					aria-hidden="true"
				/>
				<span class={zaehlerChip}
					>{cls.books.length}
					{cls.books.length === 1 ? 'Buch' : 'Bücher'}</span
				>
				<span class={zeilenTitel}>{cls.name}</span>
			</button>
		{/if}

		{#if offen}
			<div
				id={rasterID}
				class="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-5 pt-1 pb-6"
			>
				{#each sortBooks(cls.books).slice(0, anzeigeLimit) as book (book.id)}
					<KlassenBuchKachelStartseite
						{book}
						{getStockColor}
						onclick={onBookClick ? () => onBookClick(book) : undefined}
					/>
				{/each}
			</div>
			{#if cls.books.length > anzeigeLimit}
				<div class="flex justify-center pb-6">
					<Button
						variant="secondary"
						onclick={() => (anzeigeLimit += ANZEIGE_SCHRITT)}
						class="px-6"
					>
						Mehr laden ({cls.books.length - anzeigeLimit} weitere)
					</Button>
				</div>
			{/if}
		{/if}
	</section>
{/each}
