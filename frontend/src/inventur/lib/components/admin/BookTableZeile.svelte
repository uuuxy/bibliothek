<script>
	import { ChevronRight, Menu } from '@lucide/svelte';
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
	 *   index: number,
	 *   dragOverIndex: number|null,
	 *   isSelected: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onToggleSelect: (id: string) => void,
	 *   onDragStart: (event: any, index: number) => void,
	 *   onDragOver: (event: any, index: number) => void,
	 *   onDragLeave: (event: any, index: number) => void,
	 *   onDrop: (event: any, index: number) => void,
	 *   onDragEnd: (event: any) => void
	 * }}
	 */
	let {
		book,
		index,
		dragOverIndex,
		isSelected,
		onOpenDetail,
		onToggleSelect,
		onDragStart,
		onDragOver,
		onDragLeave,
		onDrop,
		onDragEnd
	} = $props();

	// Die Standorte der Exemplare im Bestand, gezählt vom Server (docs/OFFEN.md 5.53).
	const standort = $derived(standortZeile(book.standorte));
</script>

<!-- Abstand, Trennlinie, Rückmeldung beim Zeigen und die Fläche der gewählten Zeile kommen
     aus ui/Tabelle. Hier steht nur, was diese Zeile eigen hat: die Marke beim Verschieben. -->
<tr
	class="group cursor-pointer {dragOverIndex === index ? 'border-t-2 border-primary' : ''}"
	aria-selected={isSelected}
	draggable="true"
	ondragstart={(event) => onDragStart(event, index)}
	ondragover={(event) => onDragOver(event, index)}
	ondragleave={(event) => onDragLeave(event, index)}
	ondrop={(event) => onDrop(event, index)}
	ondragend={onDragEnd}
	onclick={() => onOpenDetail(book)}
>
	<td onclick={(event) => event.stopPropagation()}>
		<div class="flex items-center gap-2">
			<Menu
				class="h-4 w-4 cursor-grab text-on-surface-variant active:cursor-grabbing"
				aria-hidden="true"
			/>
			<Kaestchen
				aria-label="Buch auswählen"
				checked={isSelected}
				onchange={() => onToggleSelect(book.id)}
			/>
		</div>
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

	<td>
		<span class="font-semibold">{book.title}</span>
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
