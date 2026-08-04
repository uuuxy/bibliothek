<script>
	import { ChevronRight } from '@lucide/svelte';
	import OrderSearch from './OrderSearch.svelte';
	import OrderCart from './OrderCart.svelte';

	/**
	 * onCollapse — wenn gesetzt, trägt die Kopfzeile einen Knopf zum Einklappen der Spalte.
	 * Als Prop statt fest verdrahtet, weil das Panel nichts über das Seitenlayout wissen soll:
	 * Wer es ohne Spalten einbindet, übergibt nichts und bekommt keinen Knopf.
	 * @type {{ onCollapse?: (() => void) | null }}
	 */
	let { onCollapse = null } = $props();
</script>

<div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm">
	<div class="px-5 pt-5 pb-4 border-b border-slate-100 flex items-center justify-between gap-2">
		<h2 class="text-lg font-bold text-slate-900 tracking-tight">Deine Bestellung</h2>
		<div class="flex items-center gap-2 shrink-0">
			<span class="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-medium">Entwurf</span>
			{#if onCollapse}
				<!-- Der Knopf sitzt IN der Kopfzeile, nicht darüber im Leerraum: Ein Bedienelement
				     muss sichtbar zu dem gehören, was es bedient. -->
				<button
					onclick={onCollapse}
					aria-label="Bestellspalte einklappen"
					data-tip="Bestellspalte einklappen"
					aria-controls="bestellpanel"
					aria-expanded="true"
					class="icon-btn hidden lg:inline-flex text-slate-400 hover:text-slate-700"
				>
					<ChevronRight class="h-4 w-4" aria-hidden="true" />
				</button>
			{/if}
		</div>
	</div>
	<div class="p-5 space-y-5">
		<OrderSearch />
		<OrderCart />
	</div>
</div>
