<!-- @component PrintSuggestion — weist darauf hin, dass Exemplare ohne Barcode-Etikett
     im Bestand stehen.

     Zwei Stufen, weil zwei verschiedene Fragen dahinterstecken:

     1. FRISCH EINGEBUCHT. Direkt nach einem Wareneingang sind die betroffenen Exemplare
        namentlich bekannt — der Knopf druckt genau diese, ohne Umweg über eine Liste.
     2. STEHENDER HINWEIS. War dieser Moment vorbei, blieb hier bisher nichts. Genau so
        geht eine Lieferung verloren: Der Hinweis wird weggeklickt, und niemand erfährt
        je wieder, dass 30 Bücher ohne Etikett im Regal stehen. Deshalb zählt das
        Bestellwesen die offenen Etiketten und verweist ins Druck-Center, wo sie gesucht
        und gedruckt werden können.

     Bewusst KEIN zweiter Druckweg für Stufe 2: Der Hinweis verlinkt die vorhandene
     Liste, statt sie hier ein zweites Mal zu bauen.

     FORM (04.08.2026): Derselbe Statusstreifen wie IncomingShipments — gleiche Höhe,
     gleicher Symbolkreis, gleicher Knopf rechts. Beide sagen dasselbe: „hier wartet eine
     Aufgabe, die nichts mit der Bestellung zu tun hat, an der du gerade schreibst."
     Vorher war dieser Hinweis eine hohe Karte IN der Bestellspalte und stand damit über
     dem Warenkorb, zu dem er nicht gehört. Deshalb gibt es jetzt auch einen Leerzustand:
     Ohne ihn risse der Statusstreifen oben eine Lücke, sobald nichts aussteht. -->
<script>
	import { AlertTriangle, Printer, ArrowRight, CircleCheck } from '@lucide/svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	/** @type {{ printSuggestion: any[] | null, onPrint: () => void, offeneEtiketten?: number }} */
	let { printSuggestion, onPrint, offeneEtiketten = 0 } = $props();

	let stufe = $derived(printSuggestion ? 'lieferung' : offeneEtiketten > 0 ? 'offen' : 'leer');

	function zumNachdruck() {
		uiStore.requestedDruckCenterTab = 'nachdruck';
		uiStore.activeTab = 'druck-center';
	}
</script>

<div
	class="rounded-2xl border px-4 py-3 flex items-center gap-3 transition-colors no-print {stufe ===
	'lieferung'
		? 'bg-amber-50 border-amber-200'
		: stufe === 'offen'
			? 'bg-white border-slate-200/80 shadow-sm'
			: 'bg-slate-50/60 border-slate-200/60 border-dashed'}"
>
	<div
		class="w-9 h-9 rounded-full flex items-center justify-center shrink-0 {stufe === 'lieferung'
			? 'bg-amber-100 text-amber-700'
			: stufe === 'offen'
				? 'bg-blue-50 text-blue-600'
				: 'bg-slate-100 text-slate-400'}"
	>
		{#if stufe === 'lieferung'}
			<AlertTriangle class="h-5 w-5" aria-hidden="true" />
		{:else if stufe === 'offen'}
			<Printer class="h-5 w-5" aria-hidden="true" />
		{:else}
			<CircleCheck class="h-5 w-5" aria-hidden="true" />
		{/if}
	</div>

	<!-- Direkt auf den Wert geprüft statt auf die abgeleitete Stufe: Über `stufe` kann der
	     Typprüfer nicht verengen, printSuggestion bliebe „möglicherweise null". -->
	{#if printSuggestion}
		<div class="min-w-0 flex-1">
			<div class="text-sm font-bold text-amber-900">Etiketten für diese Lieferung</div>
			<div class="text-xs text-amber-700">
				{printSuggestion.length}
				{printSuggestion.length === 1 ? 'Exemplar' : 'Exemplare'} ohne Barcode-Etikett
			</div>
		</div>
		<!-- Sichtbar kurz, zugänglich vollständig: Im Streifen ist „Drucken" neben der
		     Überschrift eindeutig, vorgelesen wäre es das nicht. Das aria-label trägt
		     deshalb den ganzen Satz — und ist zugleich der Vertrag, auf dem
		     e2e/etiketten-druck.spec.js steht. -->
		<button
			onclick={onPrint}
			aria-label="Etiketten für diese Lieferung drucken"
			class="shrink-0 flex items-center gap-1.5 py-2 px-3.5 bg-amber-600 text-white font-bold text-xs rounded-xl transition-colors cursor-pointer"
		>
			<Printer class="h-4 w-4" aria-hidden="true" />
			Drucken
		</button>
	{:else if offeneEtiketten > 0}
		<div class="min-w-0 flex-1">
			<div class="text-sm font-bold text-slate-900">
				{offeneEtiketten} Etiketten stehen aus
			</div>
			<div class="text-xs text-slate-500">auch aus früheren Lieferungen</div>
		</div>
		<button
			onclick={zumNachdruck}
			aria-label="Im Druck-Center nachdrucken"
			class="shrink-0 flex items-center gap-1.5 py-2 px-3.5 bg-slate-900 text-white font-bold text-xs rounded-xl transition-colors cursor-pointer"
		>
			Nachdrucken
			<ArrowRight class="h-4 w-4" aria-hidden="true" />
		</button>
	{:else}
		<div class="text-sm text-slate-400 font-medium">Alle Etiketten gedruckt</div>
	{/if}
</div>
