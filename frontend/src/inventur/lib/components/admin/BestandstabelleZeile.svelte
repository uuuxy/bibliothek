<script>
	import { ChevronRight } from '@lucide/svelte';
	import Kaestchen from '../../../../lib/components/ui/Kaestchen.svelte';
	import BuchCover from '../../../../lib/components/ui/BuchCover.svelte';
	import StatusChip from '../../../../lib/components/ui/StatusChip.svelte';
	import { standortZeile } from '../../../../lib/utils/standorte.js';
	import { jahrgangSpanne } from '../../../../lib/utils/format.js';

	/**
	 * @type {{
	 *   titel: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     jahrgangVon: number,
	 *     jahrgangBis: number,
	 *     istLernmittel: boolean,
	 *     stock: number,
	 *     verfuegbar: number,
	 *     gesamt: number,
	 *     coverUrl: string,
	 *     lastCounted: string,
	 *     standorte?: import('../../../../lib/utils/standorte.js').StandortZahl[]
	 *   },
	 *   gewaehlt: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onToggleSelect: (id: string) => void
	 * }}
	 */
	let { titel, gewaehlt, onOpenDetail, onToggleSelect } = $props();

	// Die Standorte der Exemplare im Bestand, gezählt vom Server (docs/OFFEN.md 5.53).
	const standort = $derived(standortZeile(titel.standorte));
	const jahrgang = $derived(jahrgangSpanne(titel.jahrgangVon, titel.jahrgangBis));
</script>

<!-- Abstand, Trennlinie, Rückmeldung beim Zeigen und die Fläche der gewählten Zeile kommen
     aus ui/Tabelle. -->
<tr class="group cursor-pointer" aria-selected={gewaehlt} onclick={() => onOpenDetail(titel)}>
	<td onclick={(event) => event.stopPropagation()}>
		<Kaestchen
			aria-label="Buch auswählen"
			checked={gewaehlt}
			onchange={() => onToggleSelect(titel.id)}
		/>
	</td>

	<!-- Der Titel steht in der Nachbarzelle; das Cover sagt ihn kein zweites Mal an. -->
	<td>
		<BuchCover
			coverUrl={titel.coverUrl}
			isbn={titel.isbn}
			titel={titel.title}
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
				onOpenDetail(titel);
			}}
			class="text-left font-semibold text-on-surface hover:text-primary hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-primary"
		>
			{titel.title}
		</button>
		<div class="text-on-surface-variant">{titel.author}</div>
	</td>

	<!-- Was man nur liest, steht als Text; wo nichts steht, ein Strich. -->
	<td>
		{#if titel.subject}{titel.subject}{:else}<span class="text-on-surface-variant">–</span>{/if}
	</td>
	<!-- Ohne Eintrag ist der Jahrgang unbekannt: ein Strich. -->
	<td>
		{#if jahrgang}{jahrgang}{:else}<span class="text-on-surface-variant">–</span>{/if}
	</td>

	<td>
		{#if titel.istLernmittel}
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
		{#if titel.lastCounted}
			{new Date(titel.lastCounted).toLocaleDateString('de-DE')}
		{:else}
			<span class="text-on-surface-variant">–</span>
		{/if}
	</td>

	<!-- Eine Zahl, keine Farbe: Die meisten Titel der Bücherei haben ein Exemplar, und eine
	     Fehlerfarbe an fast jeder Zeile sagt nichts mehr. -->
	<td class="text-right tabular-nums">{titel.gesamt}</td>

	<td class="text-right">
		<ChevronRight
			class="h-5 w-5 text-on-surface-variant opacity-0 transition-opacity group-hover:opacity-100"
			aria-hidden="true"
		/>
	</td>
</tr>
