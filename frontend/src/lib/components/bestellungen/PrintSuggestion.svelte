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
     Liste, statt sie hier ein zweites Mal zu bauen. -->
<script>
	import { AlertTriangle, Printer, ArrowRight } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	/** @type {{ printSuggestion: any[] | null, onPrint: () => void, offeneEtiketten?: number }} */
	let { printSuggestion, onPrint, offeneEtiketten = 0 } = $props();

	function zumNachdruck() {
		uiStore.requestedDruckCenterTab = 'nachdruck';
		uiStore.activeTab = 'druck-center';
	}
</script>

{#if printSuggestion}
	<div
		class="bg-amber-50 border border-amber-200 rounded-xl p-5 shadow-2xs space-y-3.5 text-left animate-fade-in no-print"
	>
		<div class="flex items-start gap-2.5">
			<AlertTriangle class="h-4 w-4" aria-hidden="true" />
			<div>
				<h3 class="text-xs font-medium text-amber-800">Etikettendruck erforderlich</h3>
				<p class="text-xs text-amber-700 font-medium leading-relaxed mt-1">
					Es gibt {printSuggestion.length} Exemplare in dieser freigegebenen Lieferung, für die noch keine
					Barcode-Etiketten gedruckt wurden (z.B. Amazon-Bestellung).
				</p>
			</div>
		</div>
		<Button size="sm" onclick={onPrint} class="w-full">
			<Printer class="h-4 w-4" aria-hidden="true" /> Etiketten für diese Lieferung drucken
		</Button>
	</div>
{:else if offeneEtiketten > 0}
	<div
		class="bg-slate-50 border border-slate-200 rounded-xl p-5 shadow-2xs space-y-3.5 text-left animate-fade-in no-print"
	>
		<div class="flex items-start gap-2.5">
			<Printer class="h-4 w-4 shrink-0 text-slate-500" aria-hidden="true" />
			<div>
				<h3 class="text-xs font-medium text-slate-700">Etiketten stehen aus</h3>
				<p class="text-xs text-slate-500 font-medium leading-relaxed mt-1">
					{offeneEtiketten}
					{offeneEtiketten === 1 ? 'Exemplar hat' : 'Exemplare haben'} noch kein Barcode-Etikett — auch
					aus früheren Lieferungen.
				</p>
			</div>
		</div>
		<Button size="sm" variant="secondary" onclick={zumNachdruck} class="w-full">
			Im Druck-Center nachdrucken
			<ArrowRight class="h-4 w-4" aria-hidden="true" />
		</Button>
	</div>
{/if}
