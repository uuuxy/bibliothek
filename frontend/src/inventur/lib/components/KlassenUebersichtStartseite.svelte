<script>
	import KlassenBuchKachelStartseite from '$lib/components/KlassenBuchKachelStartseite.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import { ChevronDown } from '@lucide/svelte';

	/**
	 * @type {{
	 *   filteredClasses: any[],
	 *   getStockColor: (stock: number) => string,
	 *   onBookClick: (book: any) => void
	 * }}
	 */
	let { filteredClasses, getStockColor, onBookClick } = $props();

	// Immer nur ein Jahrgang offen — wie unter Verwaltung → Schulklassen. Die Wahl
	// ueberlebt einen Filterwechsel absichtlich: Wer den Filter wieder leert, findet
	// die Stelle offen vor, an der er war.
	let offenerJahrgang = $state('');

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
				<span
					class="shrink-0 rounded-lg border border-blue-100 bg-blue-50 px-3 py-1 text-sm font-semibold text-blue-600"
					>{cls.books.length}
					{cls.books.length === 1 ? 'Buch' : 'Bücher'}</span
				>
				<span class="truncate font-sans text-2xl font-bold text-slate-800">{cls.name}</span>
			</h2>
		{:else}
			<!-- Die ganze Zeile schaltet um, nicht nur das Dreieck: Das Ziel ist so gross
			     wie die Aussage, die es betrifft. -->
			<button
				type="button"
				onclick={() => (offenerJahrgang = offen ? '' : cls.name)}
				aria-expanded={offen}
				aria-controls={rasterID}
				class="flex w-full min-w-0 items-center gap-3 rounded-lg py-3 text-left"
			>
				<ChevronDown
					class="h-5 w-5 shrink-0 text-slate-500 transition-transform {offen ? '' : '-rotate-90'}"
					aria-hidden="true"
				/>
				<span
					class="shrink-0 rounded-lg border border-blue-100 bg-blue-50 px-3 py-1 text-sm font-semibold text-blue-600"
					>{cls.books.length}
					{cls.books.length === 1 ? 'Buch' : 'Bücher'}</span
				>
				<span class="truncate font-sans text-2xl font-bold text-slate-800">{cls.name}</span>
			</button>
		{/if}

		{#if offen}
			<div id={rasterID} class="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-5 pt-1 pb-6">
				{#each sortBooks(cls.books) as book (book.id)}
					<KlassenBuchKachelStartseite {book} {getStockColor} onclick={() => onBookClick(book)} />
				{/each}
			</div>
		{/if}
	</section>
{/each}
