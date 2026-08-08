<script>
	import KlassenBuchKachel from '$lib/components/admin/KlassenBuchKachel.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { Pencil, Trash2 } from '@lucide/svelte';

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

	<!-- Umbrechendes Raster statt waagerechtem Karussell.
	     Gemessen am 08.08.2026: 2.876 px Inhalt auf 1.280 px Fläche — NEUN von sechzehn
	     Büchern lagen ausserhalb des Bildes. Dazu drei Fehler, die sich gegenseitig
	     verdeckten: ein 95 % SCHWARZER Verlauf (rgba(9,9,11,.95)) aus der CSS-Datei legte
	     sich ueber das erste und letzte Buch, im Markup lag ein zweiter Verlauf nach
	     slate-50 darueber (die Arbeitsflaeche ist seit a4133a2 weiss), und die Pfeile
	     standen auf opacity:0 + pointer-events:none, sichtbar erst bei :hover — am Tablet
	     am Pult also nie.

	     Der eigentliche Punkt ist aber nicht die Reparatur: M3 kennt ein Carousel, meint
	     damit aber das STOEBERN in Bildmaterial (hero / multi-browse / uncontained). Diese
	     Seite beantwortet eine Verwaltungsfrage — „hat 05F1 alle sechzehn Buecher?". Dafuer
	     muss man sie sehen, nicht durch sie scrollen. Ein Raster zeigt alle auf einmal und
	     braucht weder Pfeil noch Verlauf noch Hover. -->
	<div class="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-5 pb-2">
		{#each sortedBooks as book (book.id)}
			<KlassenBuchKachel {book} {onEdit} />
		{/each}
	</div>
</div>
