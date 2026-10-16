<script>
	import LeserTreffer from './omnibox/LeserTreffer.svelte';
	import TitelTreffer from './omnibox/TitelTreffer.svelte';

	let { suchTreffer, markierterTreffer, onSelect } = $props();

	/**
	 * Beschriftet eine Ergebnisgruppe. Die Liste ist serverseitig auf 10 begrenzt —
	 * ohne den Zusatz "von N" behauptet die Überschrift, es gäbe nur zehn Treffer,
	 * und wer weiter tippen müsste, sieht keinen Anlass dazu.
	 * @param {string} label
	 * @param {number} gezeigt
	 * @param {number} gesamt
	 */
	function gruppenTitel(label, gezeigt, gesamt) {
		if (gesamt > gezeigt) return `${label} (${gezeigt} von ${gesamt})`;
		return `${label} (${gezeigt})`;
	}
</script>

<!-- Solide Fläche statt Milchglas: Die Liste schwebt über Inhalt und muss von ihm
     getrennt sein, sonst steht Text auf Text. Ein flacher Schatten leistet das
     (Material-3-Elevation 2). Gruppen brauchen keine eigenen Wrapper: Die Überschrift ist
     ein Geschwister der Zeilen, kein Elternteil. -->
<div
	id="omnibox-dropdown"
	role="listbox"
	aria-label="Suchergebnisse"
	class="absolute top-full left-0 right-0 mt-2 z-50 max-h-[60vh] overflow-y-auto overscroll-contain bg-surface-container rounded-sm shadow-xl py-2"
>
	{#if suchTreffer.students.length > 0}
		<div class="px-4 pt-3 pb-2 text-xs font-medium text-on-surface-variant">
			{gruppenTitel(
				'Leser',
				suchTreffer.students.length,
				suchTreffer.studentsTotal ?? suchTreffer.students.length
			)}
		</div>
		{#each suchTreffer.students as student, i (i)}
			<LeserTreffer leser={student} index={i} gewaehlt={markierterTreffer === i} {onSelect} />
		{/each}
	{/if}
	{#if suchTreffer.books.length > 0}
		<div class="px-4 pt-3 pb-2 text-xs font-medium text-on-surface-variant">
			{gruppenTitel(
				'Bücher',
				suchTreffer.books.length,
				suchTreffer.booksTotal ?? suchTreffer.books.length
			)}
		</div>
		{#each suchTreffer.books as book, j (j)}
			{@const index = j + suchTreffer.students.length}
			<TitelTreffer treffer={book} {index} gewaehlt={markierterTreffer === index} {onSelect} />
		{/each}
	{/if}
</div>
