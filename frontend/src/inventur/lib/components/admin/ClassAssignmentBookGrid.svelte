<script>
	import { Book, Check } from '@lucide/svelte';
	import Suchpille from '../../../../lib/components/ui/Suchpille.svelte';
	import { jahrgangSpanne } from '../../../../lib/utils/format.js';
	// buecherFehler: Der Abruf der Bücher ist gescheitert. Ein leeres Gitter sähe sonst aus
	// wie „kein Buch im Bestand" — der Dialog wäre unbenutzbar, ohne zu sagen, warum.
	let { books = [], buecherFehler = false, selectedBookIds = $bindable(new Set()) } = $props();

	let searchQuery = $state('');

	const filteredBooks = $derived(
		books.filter(
			(b) =>
				b.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.author.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	// Nicht den ganzen Bestand zeichnen: Mit 5.875 Titeln sind es 59.120 DOM-Knoten und 2,5
	// Sekunden, bis der Dialog steht. Die Zahl im Suchfeld nennt alle Treffer, nicht die
	// gezeichneten — sonst verschwiege die Oberfläche, dass es mehr gibt.
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
	<p class="text-xs text-on-surface-variant font-medium mb-1">BÜCHER FINDEN</p>

	<Suchpille
		id="book-search-field"
		bind:wert={searchQuery}
		platzhalter="Titel, Fach oder ISBN eingeben …"
		etikett="Bücher durchsuchen"
		autofokus
		{nachlaufend}
	/>

	{#if buecherFehler}
		<p class="mt-2 px-1 text-sm font-semibold text-error" role="alert">
			Die Bücherliste konnte nicht geladen werden — hier steht deshalb nichts. Das heißt NICHT, dass
			keine Bücher im Bestand sind.
		</p>
	{/if}

	{#if filteredBooks.length > ANZEIGE_GRENZE}
		<p class="mt-2 px-1 text-xs text-on-surface-variant">
			Gezeigt werden die ersten {ANZEIGE_GRENZE} von {filteredBooks.length}. Suchbegriff eingrenzen,
			um das gesuchte Buch zu sehen.
		</p>
	{/if}
</div>

{#snippet nachlaufend()}
	<span
		class="shrink-0 whitespace-nowrap rounded-full bg-on-surface/5 px-3 py-1 text-xs font-bold text-on-surface-variant"
		>{filteredBooks.length} Treffer</span
	>
{/snippet}

<div
	class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 sm:gap-6 pb-2 mt-6 sm:mt-8"
>
	{#each sichtbareBuecher as book (book.id)}
		{@const jahrgang = jahrgangSpanne(book.jahrgangVon, book.jahrgangBis)}
		<button
			onclick={() => toggleBook(book.id)}
			aria-pressed={selectedBookIds.has(book.id)}
			class="group relative flex flex-col text-left rounded-3xl overflow-hidden transition-all duration-300 transform active:scale-95
            {selectedBookIds.has(book.id)
				? 'bg-primary-container/30 ring-4 ring-primary shadow-xl scale-[1.02]'
				: 'bg-surface-container-lowest shadow-md hover:shadow-xl'}"
		>
			<!-- Selection Overlay -->
			{#if selectedBookIds.has(book.id)}
				<div
					class="absolute top-4 right-4 z-10 bg-primary text-on-primary p-1.5 rounded-full border-2 border-surface-container-lowest animate-in zoom-in-50 duration-200"
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
					<div
						class="w-full h-full hidden items-center justify-center bg-surface-container text-outline-variant"
					>
						<Book class="w-5 h-5" aria-hidden="true" />
					</div>
				{:else}
					<div
						class="w-full h-full flex items-center justify-center bg-surface-container text-outline-variant"
					>
						<Book class="w-5 h-5" aria-hidden="true" />
					</div>
				{/if}
				<div
					class="absolute inset-0 bg-linear-to-t from-scrim/20 to-transparent group-hover:from-scrim/40 transition-colors"
				></div>
			</div>

			<!-- Content -->
			<div class="p-5 flex flex-col grow justify-end space-y-3 w-full">
				<!-- Beide Marken nur, wenn sie etwas zu sagen haben: Ohne Eintrag ist der Jahrgang
				     unbekannt, und die Karte trägt keine Marke dafür. -->
				<div class="flex flex-wrap gap-1.5 items-start">
					{#if book.subject}
						<span
							class="px-2.5 py-0.5 bg-primary-container text-on-primary-container text-xs font-black rounded-lg"
							>{book.subject}</span
						>
					{/if}
					{#if jahrgang}
						<span
							class="px-2.5 py-0.5 bg-surface-container-high text-on-surface-variant text-xs font-black rounded-lg"
						>
							Jg. {jahrgang}
						</span>
					{/if}
				</div>
				<div>
					<h3 class="font-bold text-on-surface leading-tight line-clamp-2">
						{book.title}
					</h3>
				</div>
			</div>
		</button>
	{/each}
</div>
