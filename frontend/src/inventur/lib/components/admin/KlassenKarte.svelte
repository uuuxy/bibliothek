<script>
	import KlassenBuchKachel from '$lib/components/admin/KlassenBuchKachel.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import { scrollCarousel, scrollHandler } from '$lib/carousel_utils.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { ChevronLeft, ChevronRight, Pencil, Trash2 } from '@lucide/svelte';

	/**
	 * @type {{
	 *   group: {
	 *     className: string,
	 *     books: any[]
	 *   },
	 *   darfPflegen?: boolean,
	 *   onEdit: () => void,
	 *   onDelete: () => void
	 * }}
	 */
	let { group, darfPflegen = false, onEdit, onDelete } = $props();

	let sortedBooks = $derived([...group.books].sort(sortBooksBySubjectAndTitle));
</script>

<div class="class-group">
	<div class="flex justify-between items-center mb-4 px-2">
		<h2 class="text-2xl font-bold text-slate-800 flex items-center gap-2 font-sans">
			<span
				class="bg-blue-50 border border-blue-105 text-blue-600 px-3 py-1 rounded-lg text-sm font-semibold"
				>{group.books.length} Bücher</span
			>
			{group.className}
		</h2>
		<!-- Ohne edit_books bleibt die Karte lesbar und verliert nur die Aktionen. Der
		     Server entscheidet ohnehin (POST/DELETE hängen an edit_books) — hier geht
		     es darum, niemandem einen Knopf anzubieten, der im 403 endet. -->
		{#if darfPflegen}
			<div class="flex gap-2">
				<Button
					variant="secondary"
					onclick={onEdit}
					class="border-blue-100 bg-blue-50 text-blue-600 hover:bg-blue-100"
					title="Klasse bearbeiten"
					aria-label="Klasse bearbeiten"
				>
					<Pencil class="w-4 h-4" aria-hidden="true" />
					Bücher verwalten
				</Button>
				<button
					onclick={onDelete}
					class="text-rose-500 hover:text-rose-600 hover:bg-rose-50 p-2 rounded-lg transition-colors cursor-pointer"
					title="Klasse löschen"
					aria-label="Klasse löschen"
				>
					<Trash2 class="w-5 h-5" aria-hidden="true" />
				</button>
			</div>
		{/if}
	</div>

	<!-- Horizontal Scroll Container (Netflix Style) -->
	<div
		class="relative group/admin-carousel carousel-wrapper"
		data-can-scroll-left="false"
		data-can-scroll-right="false"
	>
		<!-- Left Gradient -->
		<div
			class="absolute left-0 top-0 bottom-6 w-24 bg-linear-to-r from-slate-50 to-transparent pointer-events-none z-30 opacity-0 group-data-[can-scroll-left=true]/admin-carousel:opacity-100 transition-opacity duration-300 rounded-l-2xl"
		></div>

		<button
			class="btn-left absolute left-0 top-1/2 -translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white border border-slate-200 flex items-center justify-center text-slate-500 hover:text-blue-600 hover:border-blue-300 hover:shadow-md transition-all z-40 cursor-pointer"
			onclick={(e) => scrollCarousel(e, -1)}
			aria-label="Nach links scrollen"
		>
			<ChevronLeft class="w-5 h-5" aria-hidden="true" />
		</button>

		<div
			use:scrollHandler
			class="carousel-container flex overflow-x-auto gap-5 pb-6 px-2 snap-x hide-scrollbar scroll-smooth"
		>
			{#each sortedBooks as book (book.id)}
				<KlassenBuchKachel {book} {onEdit} />
			{/each}
		</div>

		<!-- Right Gradient -->
		<div
			class="absolute right-0 top-0 bottom-6 w-32 bg-linear-to-l from-slate-50 to-transparent pointer-events-none z-30 opacity-0 group-data-[can-scroll-right=true]/admin-carousel:opacity-100 transition-opacity duration-300 rounded-r-2xl"
		></div>

		<button
			class="btn-right absolute right-0 top-1/2 translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white border border-slate-200 flex items-center justify-center text-slate-500 hover:text-blue-600 hover:border-blue-300 hover:shadow-md transition-all z-40 cursor-pointer"
			onclick={(e) => scrollCarousel(e, 1)}
			aria-label="Nach rechts scrollen"
		>
			<ChevronRight class="w-5 h-5" aria-hidden="true" />
		</button>
	</div>
</div>
