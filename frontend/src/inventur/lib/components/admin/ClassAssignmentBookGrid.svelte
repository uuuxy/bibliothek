<script>
	import { Book, Check } from '@lucide/svelte';
	import Suchpille from '../../../../lib/components/ui/Suchpille.svelte';
	let { books = [], selectedBookIds = $bindable(new Set()) } = $props();

	let searchQuery = $state('');

	const filteredBooks = $derived(
		books.filter(
			(b) =>
				b.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.author.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	// Nicht den ganzen Bestand zeichnen.
	//
	// Gemessen am 11.08.2026 mit 5.875 Titeln: 5.875 Kacheln, 59.120 DOM-Knoten,
	// 2,5 Sekunden bis der Dialog stand. Die Produktion fuehrt rund 13.700 Titel — dort
	// waere das ein Vielfaches davon, auf einem Schulrechner ohne Weiteres eine halbe
	// Minute. Und es bringt nichts: Durch dreizehntausend Kacheln scrollt niemand, um ein
	// bestimmtes Buch zu finden. Dafuer ist die Suche da, und die Ueberschrift sagt es auch.
	//
	// Die Zahl daneben nennt weiterhin ALLE Treffer, nicht die gezeichneten — sonst
	// verschwiege die Oberflaeche, dass es mehr gibt.
	const ANZEIGE_GRENZE = 60;
	const sichtbareBuecher = $derived(filteredBooks.slice(0, ANZEIGE_GRENZE));

	/**
	 * @param {string} id
	 */
	function toggleBook(id) {
		if (selectedBookIds.has(id)) {
			selectedBookIds = new Set([...selectedBookIds].filter((bId) => bId !== id));
		} else {
			selectedBookIds = new Set([...selectedBookIds, id]);
		}
	}

	/**
	 * @param {Event & { target: any }} event
	 */
	function handleImageError(event) {
		event.target.style.display = 'none';
		event.target.nextElementSibling.style.display = 'flex';
	}
</script>

<div class="mb-4 px-1">
	<p class="text-xs text-slate-500 font-medium mb-1">BÜCHER FINDEN</p>

	<Suchpille
		id="book-search-field"
		bind:wert={searchQuery}
		platzhalter="Titel, Fach oder ISBN eingeben …"
		etikett="Bücher durchsuchen"
		autofokus
		{nachlaufend}
	/>

	{#if filteredBooks.length > ANZEIGE_GRENZE}
		<p class="mt-2 px-1 text-xs text-slate-500">
			Gezeigt werden die ersten {ANZEIGE_GRENZE} von {filteredBooks.length}. Suchbegriff eingrenzen,
			um das gesuchte Buch zu sehen.
		</p>
	{/if}
</div>

{#snippet nachlaufend()}
	<span
		class="shrink-0 whitespace-nowrap rounded-full bg-black/5 px-3 py-1 text-xs font-bold text-slate-500"
		>{filteredBooks.length} Treffer</span
	>
{/snippet}

<div
	class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 sm:gap-6 pb-2 mt-6 sm:mt-8"
>
	{#each sichtbareBuecher as book (book.id)}
		<button
			onclick={() => toggleBook(book.id)}
			aria-pressed={selectedBookIds.has(book.id)}
			class="group relative flex flex-col text-left rounded-3xl overflow-hidden transition-all duration-300 transform active:scale-95
            {selectedBookIds.has(book.id)
				? 'bg-primary-50 ring-4 ring-primary-500 shadow-xl scale-[1.02]'
				: 'bg-white hover:bg-surface-container-low shadow-md hover:shadow-xl border border-surface-variant/10 hover:border-primary-200'}"
		>
			<!-- Selection Overlay -->
			{#if selectedBookIds.has(book.id)}
				<div
					class="absolute top-4 right-4 z-10 bg-primary-600 text-white p-1.5 rounded-full shadow-lg border-2 border-white animate-in zoom-in-50 duration-200"
				>
					<Check class="w-5 h-5" aria-hidden="true" />
				</div>
			{/if}

			<!-- Cover -->
			<div class="aspect-2/3 w-full overflow-hidden bg-surface-container relative shrink-0">
				{#if book.coverUrl}
					<img
						src={book.coverUrl}
						alt={book.title}
						class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
						onerror={handleImageError}
					/>
					<div class="w-full h-full hidden items-center justify-center bg-slate-100 text-slate-300">
						<Book class="w-5 h-5" aria-hidden="true" />
					</div>
				{:else}
					<div class="w-full h-full flex items-center justify-center bg-slate-100 text-slate-300">
						<Book class="w-5 h-5" aria-hidden="true" />
					</div>
				{/if}
				<div
					class="absolute inset-0 bg-linear-to-t from-black/20 to-transparent group-hover:from-black/40 transition-colors"
				></div>
			</div>

			<!-- Content -->
			<div class="p-5 flex flex-col grow justify-end space-y-3 w-full">
				<!-- Beide Marken nur, wenn sie etwas zu sagen haben. Vorher stand auf jeder
				     Karte ohne Jahrgang „Kl. 0" und daneben eine leere Fachmarke — eine
				     Klasse 0 gibt es nicht, die Angabe behauptete also etwas Falsches,
				     statt zu schweigen. -->
				<div class="flex flex-wrap gap-1.5 items-start">
					{#if book.subject}
						<span
							class="px-2.5 py-0.5 bg-primary-100 text-primary-900 text-xs font-black rounded-lg"
							>{book.subject}</span
						>
					{/if}
					{#if book.gradeLevel}
						<span
							class="px-2.5 py-0.5 bg-surface-container-high text-surface-variant text-xs font-black rounded-lg"
						>
							Kl. {book.gradeLevel}
						</span>
					{/if}
				</div>
				<div>
					<h3
						class="font-bold text-primary-950 leading-tight line-clamp-2 group-hover:text-primary-700 transition-colors"
					>
						{book.title}
					</h3>
				</div>
			</div>
		</button>
	{/each}
</div>
