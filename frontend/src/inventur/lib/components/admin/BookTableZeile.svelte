<script>
	import { ChevronRight } from '@lucide/svelte';
	import Kaestchen from '../../../../lib/components/ui/Kaestchen.svelte';
	import BuchCover from '../../../../lib/components/ui/BuchCover.svelte';
	import StatusChip from '../../../../lib/components/ui/StatusChip.svelte';
	import { standortZeile } from '../../../../lib/utils/standorte.js';

	/**
	 * @type {{
	 *   book: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     gradeLevel: number,
	 *     istLernmittel: boolean,
	 *     stock: number,
	 *     verfuegbar: number,
	 *     gesamt: number,
	 *     coverUrl: string,
	 *     lastCounted: string,
	 *     standorte?: import('../../../../lib/utils/standorte.js').StandortZahl[]
	 *   },
	 *   isSelected: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onToggleSelect: (id: string) => void
	 * }}
	 */
	let { book, isSelected, onOpenDetail, onToggleSelect } = $props();

	// Die Standorte der Exemplare im Bestand, gezählt vom Server (docs/OFFEN.md 5.53).
	const standort = $derived(standortZeile(book.standorte));
</script>

<!-- Abstand, Trennlinie, Rückmeldung beim Zeigen und die Fläche der gewählten Zeile kommen
     aus ui/Tabelle. -->
<tr class="group cursor-pointer" aria-selected={isSelected} onclick={() => onOpenDetail(book)}>
	<td onclick={(event) => event.stopPropagation()}>
		<Kaestchen
			aria-label="Buch auswählen"
			checked={isSelected}
			onchange={() => onToggleSelect(book.id)}
		/>
	</td>

	<!-- Der Titel steht in der Nachbarzelle; das Cover sagt ihn kein zweites Mal an. -->
	<td>
		<BuchCover
			coverUrl={book.coverUrl}
			isbn={book.isbn}
			titel={book.title}
			groesse="liste"
			dekorativ
		/>
	</td>

	<!-- Der Titel ist ein Knopf wie der Name in der Leserdatei: So öffnet auch die Tastatur die
	     Akte. Der Klick bleibt beim Knopf, sonst öffnete die Zeile ein zweites Mal. -->
	<td>
		<button
			type="button"
			onclick={(event) => {
				event.stopPropagation();
				onOpenDetail(book);
			}}
			class="text-left font-semibold text-on-surface hover:text-primary hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-primary"
		>
			{book.title}
		</button>
		<div class="text-on-surface-variant">{book.author}</div>
	</td>

	<!-- Was man nur liest, steht als Text; wo nichts steht, ein Strich. -->
	<td>
		{#if book.subject}{book.subject}{:else}<span class="text-on-surface-variant">–</span>{/if}
	</td>
	<!-- Klasse 0 = nicht zugeordnet: „–" statt einer sinnlosen „Kl. 0". -->
	<td>
		{#if book.gradeLevel}Kl. {book.gradeLevel}{:else}<span class="text-on-surface-variant">–</span
			>{/if}
	</td>

	<td>
		{#if book.istLernmittel}
			<StatusChip text="Lernmittel" />
		{:else}
			<span class="text-on-surface-variant">Bibliothek</span>
		{/if}
	</td>

	<td>
		{#if standort}
			{standort}
		{:else}
			<span class="text-on-surface-variant">–</span>
		{/if}
	</td>

	<td class="text-right tabular-nums">
		{#if book.lastCounted}
			{new Date(book.lastCounted).toLocaleDateString('de-DE')}
		{:else}
			<span class="text-on-surface-variant">–</span>
		{/if}
	</td>

	<!-- Eine Zahl, keine Farbe: Die meisten Titel der Bücherei haben ein Exemplar, und eine
	     Fehlerfarbe an fast jeder Zeile sagt nichts mehr. -->
	<td class="text-right tabular-nums">{book.gesamt}</td>

	<td class="text-right">
		<ChevronRight
			class="h-5 w-5 text-on-surface-variant opacity-0 transition-opacity group-hover:opacity-100"
			aria-hidden="true"
		/>
	</td>
</tr>
