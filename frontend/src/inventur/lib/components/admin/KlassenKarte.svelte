<script>
	import KlassenBuchKachel from '$lib/components/admin/KlassenBuchKachel.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { ChevronDown, Pencil, Trash } from '@lucide/svelte';

	/**
	 * @type {{
	 *   group: {
	 *     className: string,
	 *     books: any[]
	 *   },
	 *   offen?: boolean,
	 *   darfPflegen?: boolean,
	 *   onToggle: () => void,
	 *   onEdit?: () => void,
	 *   onDelete?: () => void
	 * }}
	 * Ohne darfPflegen (Kollegiums-Portal) braucht es keine Aktionen — die Karte ist dann
	 * dieselbe Ansicht wie unter Bibliothek → Klassensätze, nur lesend.
	 *
	 * Der Klassenname ist in beiden Ansichten eine Listenzeile von 16 px: Dichte gehört ins
	 * Padding, nicht in die Schriftgröße.
	 */
	let {
		group,
		offen = false,
		darfPflegen = false,
		onToggle,
		onEdit = undefined,
		onDelete = undefined
	} = $props();
	const zeilenTitel = 'truncate text-base font-medium text-on-surface';
	// Feste Mindestbreite und Tabellenziffern: Sonst ist der Chip bei „28 Bücher" breiter als
	// bei „5 Bücher", und der Klassenname rutscht je Zeile nach rechts.
	const zaehlerChip =
		'bg-secondary-container text-on-secondary-container inline-flex min-w-26 shrink-0 justify-center rounded-full px-3 py-0.5 text-xs font-semibold tabular-nums';

	let sortedBooks = $derived([...group.books].sort(sortBooksBySubjectAndTitle));
	const rasterID = $derived(`klassensatz-${group.className.replace(/\s+/g, '-')}`);
</script>

<!-- Eine Zeile je Klasse, ausklappbar: Die Zeile sagt, welche Klassen es gibt und wie viele
     Bücher ihr Satz hat; das Raster zeigt den Satz der einen Klasse, die man gerade ansieht.
     Mit zwanzig ausgebreiteten Sätzen wäre die Klassenliste nicht mehr zu überblicken. -->
<div class="class-group border-b border-outline-variant last:border-b-0">
	<div class="flex items-center justify-between gap-4 py-3">
		<!-- Die ganze Zeile schaltet um, nicht nur ein kleines Dreieck: Das Ziel ist so
		     gross wie die Aussage, die es betrifft. -->
		<button
			type="button"
			onclick={onToggle}
			aria-expanded={offen}
			aria-controls={rasterID}
			class="group flex min-w-0 flex-1 items-center gap-3 rounded-lg py-1 text-left"
		>
			<ChevronDown
				class="h-5 w-5 shrink-0 text-on-surface-variant transition-transform {offen
					? ''
					: '-rotate-90'}"
				aria-hidden="true"
			/>
			<span class={zaehlerChip}
				>{group.books.length}
				{group.books.length === 1 ? 'Buch' : 'Bücher'}</span
			>
			<span class={zeilenTitel}>{group.className}</span>
		</button>

		<!-- Ohne edit_books bleibt die Karte lesbar und verliert nur die Aktionen. Der
		     Server entscheidet ohnehin (POST/DELETE hängen an edit_books) — hier geht
		     es darum, niemandem einen Knopf anzubieten, der im 403 endet. -->
		{#if darfPflegen}
			<div class="flex shrink-0 gap-2">
				<Button
					variant="secondary"
					onclick={onEdit}
					title="Klasse bearbeiten"
					aria-label="Klasse bearbeiten"
				>
					<Pencil class="w-4 h-4" aria-hidden="true" />
					Bücher verwalten
				</Button>
				<button
					onclick={onDelete}
					class="icon-btn h-9 w-9 text-error"
					data-tip="Buchliste löschen"
					aria-label="Buchliste löschen"
				>
					<Trash class="w-5 h-5" aria-hidden="true" />
				</button>
			</div>
		{/if}
	</div>

	{#if offen}
		<!-- Umbrechendes Raster statt waagerechtem Karussell: Ein Karussell legt den größeren
		     Teil eines Satzes außerhalb des Bildes ab. M3 sieht es für das Stöbern in
		     Bildmaterial vor, nicht für das Prüfen eines Bestands. -->
		<div
			id={rasterID}
			class="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-x-3 gap-y-4 pt-1 pb-6"
		>
			{#each sortedBooks as book (book.id)}
				<KlassenBuchKachel {book} {onEdit} bearbeitbar={darfPflegen} />
			{/each}
		</div>
	{/if}
</div>
