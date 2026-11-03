<script>
	import BuchKarte from './BuchKarte.svelte';

	/**
	 * @type {{
	 *   titelListe: any[],
	 *   onBookClick: (book: any) => void,
	 *   onEditClick?: (book: any) => void
	 * }}
	 */
	let { titelListe, onBookClick, onEditClick } = $props();
</script>

<!-- Feste Kachelbreite, variable Anzahl: mindestens 192 px, damit eine 13-stellige ISBN in
     Festbreite neben dem Kopieren-Symbol im Balken Platz hat. Bei 960 px Inhalt sind das vier
     je Reihe, auf einem breiten Monitor acht. Gleicher Abstand in beide Richtungen, weil die
     Karten einen Rahmen tragen und ungleiche Fugen sichtbar wären; 8 px nach M3 (Cards, Specs:
     „Padding between cards: 8dp max"). -->
<div class="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-2">
	{#each titelListe as book (book.id)}
		<BuchKarte
			titel={book}
			onclick={() => onBookClick(book)}
			onEditClick={onEditClick ? () => onEditClick(book) : undefined}
		/>
	{/each}
</div>
