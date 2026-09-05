<script>
	import KlassenBuchKachel from '$lib/components/admin/KlassenBuchKachel.svelte';
	import { sortBooksBySubjectAndTitle } from '$lib/book_sorting.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { ChevronDown, Pencil, Trash2 } from '@lucide/svelte';

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
	 * EINE Fassung seit 05.09.2026: Bis dahin trug die Karte unter Bibliothek → Klassensätze
	 * den Klassennamen als Seitenüberschrift (24 px fett, slate) und nur das Portal die
	 * Listenzeile (`kompakt`, 25.08.2026). Die Karte ist in beiden Fällen dasselbe
	 * ausklappbare Listenelement (siehe Kommentar am Markup), und eine Listenzeile hat in M3
	 * 16 px — die 24 px stammten aus dem ersten Feature-Commit, nicht aus einer Entscheidung.
	 * Dichte gehört ins Padding, nicht in die Schriftgröße; die Farben in die Rollen.
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
	// Feste Mindestbreite + Tabellenziffern: Bei „28 Bücher" war der Chip breiter als bei
	// „5 Bücher", und der Klassenname rutschte je Zeile nach rechts (Peter, 26.08.2026).
	const zaehlerChip =
		'bg-secondary-container text-on-secondary-container inline-flex min-w-[6.5rem] shrink-0 justify-center rounded-full px-3 py-0.5 text-xs font-semibold tabular-nums';

	let sortedBooks = $derived([...group.books].sort(sortBooksBySubjectAndTitle));
	const rasterID = $derived(`klassensatz-${group.className.replace(/\s+/g, '-')}`);
</script>

<!-- Eine Zeile je Klasse, ausklappbar — nicht alle Saetze gleichzeitig ausgebreitet.
     Peter am 09.08.2026: "jetzt sind die fotos untereinander! bei 20 klassen wird das
     unuebersichtlich". Er hat recht, und es war meine Ueberkorrektur: Das Raster hat den
     Seitwaertsscroll je Klasse beseitigt, dafuer standen bei zwanzig Klassen 320 Kacheln
     untereinander — die Klassenliste selbst war dann nicht mehr zu ueberblicken.

     Die Seite beantwortet zwei verschiedene Fragen. "Welche Klassen gibt es, und haben
     sie ueberhaupt einen Satz?" beantwortet die Zeile mit Namen und Anzahl. "Stimmt der
     Satz von 05F1?" beantwortet das Raster — aber nur fuer die eine Klasse, die man
     gerade ansieht. In M3 ist das ein ausklappbares Listenelement; die Anzahl steht
     schon in der eingeklappten Zeile, damit man zum Nachsehen gar nicht erst oeffnen
     muss. -->
<div class="class-group border-b border-slate-200 last:border-b-0">
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

	{#if offen}
		<!-- Umbrechendes Raster statt waagerechtem Karussell (08.08.2026): Gemessen lagen
		     2.876 px Inhalt auf 1.280 px Flaeche, neun von sechzehn Buechern also
		     ausserhalb des Bildes. Die Pfeile dazu standen auf opacity:0 und erschienen
		     erst bei :hover — am Tablet am Pult nie. M3 kennt zwar ein Carousel, meint
		     damit aber das STOEBERN in Bildmaterial, nicht das Pruefen eines Bestands. -->
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
