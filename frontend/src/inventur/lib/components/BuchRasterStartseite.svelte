<script>
	import BuchKarte from './BuchKarte.svelte';

	/**
	 * @type {{
	 *   filteredBooks: any[],
	 *   onBookClick: (book: any) => void,
	 *   onEditClick?: (book: any) => void
	 * }}
	 */
	let { filteredBooks, onBookClick, onEditClick } = $props();
</script>

<!-- Feste Kachelbreite, variable Anzahl (wie Google Play Books): mindestens 192 px, damit
     „ISBN: 9783127335712" in Festbreite neben dem Kopieren-Symbol Platz hat. Bei 960 px
     Inhalt sind das vier je Reihe, auf einem breiten Monitor acht. -->
<div class="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-x-3 gap-y-4">
	{#each filteredBooks as book (book.id)}
		<BuchKarte
			{book}
			onclick={() => onBookClick(book)}
			onEditClick={onEditClick ? () => onEditClick(book) : undefined}
		/>
	{/each}
</div>
