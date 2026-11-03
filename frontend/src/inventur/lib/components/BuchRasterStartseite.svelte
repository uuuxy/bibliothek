<script>
	import BuchKarte, { RASTER_ABSTAND } from './BuchKarte.svelte';

	/**
	 * @type {{
	 *   titelListe: any[],
	 *   onBookClick: (book: any) => void,
	 *   onEditClick?: (book: any) => void
	 * }}
	 */
	let { titelListe, onBookClick, onEditClick } = $props();
</script>

<!-- Feste Kachelbreite, variable Anzahl: mindestens 192 px, damit die ISBN in Festbreite in der
     Einblendung in eine Zeile passt; bei 960 px Inhalt sind das vier je Reihe. Den Abstand
     gibt RASTER_ABSTAND, gleich in Katalog, Klassensätzen und Portal. -->
<div class="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] {RASTER_ABSTAND}">
	{#each titelListe as book (book.id)}
		<BuchKarte
			titel={book}
			onclick={() => onBookClick(book)}
			onEditClick={onEditClick ? () => onEditClick(book) : undefined}
		/>
	{/each}
</div>
