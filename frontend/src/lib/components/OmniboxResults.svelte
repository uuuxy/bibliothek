<script>
	import StudentResultItem from './omnibox/StudentResultItem.svelte';
	import BookResultItem from './omnibox/BookResultItem.svelte';

	let { unifiedSearchResults, selectedDropdownIndex, onSelect } = $props();

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
     (Material-3-Elevation 2) — bg-white/80 + backdrop-blur-2xl + weißer Rahmen war
     Dekoration, die dasselbe schlechter konnte. Gruppen brauchen keine eigenen
     Wrapper: Die Überschrift ist ein Geschwister der Zeilen, kein Elternteil. -->
<div
	id="omnibox-dropdown"
	role="listbox"
	aria-label="Suchergebnisse"
	class="absolute top-full left-0 right-0 mt-2 z-50 max-h-[60vh] overflow-y-auto overscroll-contain bg-white border border-slate-200 rounded-sm shadow-[0_2px_6px_rgb(0,0,0,0.10)] py-1"
>
	{#if unifiedSearchResults.students.length > 0}
		<div class="px-3 pt-2 pb-1 text-xs text-slate-500">
			{gruppenTitel(
				'Schüler',
				unifiedSearchResults.students.length,
				unifiedSearchResults.studentsTotal ?? unifiedSearchResults.students.length
			)}
		</div>
		{#each unifiedSearchResults.students as student, i (i)}
			<StudentResultItem {student} index={i} selected={selectedDropdownIndex === i} {onSelect} />
		{/each}
	{/if}
	{#if unifiedSearchResults.books.length > 0}
		<div class="px-3 pt-2 pb-1 text-xs text-slate-500">
			{gruppenTitel(
				'Bücher',
				unifiedSearchResults.books.length,
				unifiedSearchResults.booksTotal ?? unifiedSearchResults.books.length
			)}
		</div>
		{#each unifiedSearchResults.books as book, j (j)}
			{@const index = j + unifiedSearchResults.students.length}
			<BookResultItem {book} {index} selected={selectedDropdownIndex === index} {onSelect} />
		{/each}
	{/if}
</div>
