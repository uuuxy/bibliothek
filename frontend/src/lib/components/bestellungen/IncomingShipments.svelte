<script>
	import { Package } from '@lucide/svelte';
	let { incomingShipments, showGreenFade, onOpenWareneingang } = $props();

	let totalItems = $derived(
		incomingShipments.reduce((sum, s) => sum + s.items.reduce((s2, i) => s2 + i.menge, 0), 0)
	);
	let totalShipments = $derived(incomingShipments.length);
	let hatZulauf = $derived(incomingShipments.length > 0);
</script>

<!-- Slimer Status-Streifen: im Leerzustand nur eine dezente Zeile, mit Zulauf eine
     kompakte, klickbare Flaeche — nie mehr die halbe Spalte für „nichts da".

     Die beiden Zustaende unterscheiden sich ueber den FLAECHENTON, nicht mehr ueber
     Rahmen und Schatten. In M3 traegt die Tonstufe die Elevation; ein umrandeter,
     schattierter Kasten war auf der ansonsten flachen Seite das einzige schwebende
     Objekt — und genau das, was hier abgeschafft wurde. -->
<div
	class="rounded-xl px-4 py-3 flex items-center gap-3 transition-colors {hatZulauf
		? 'bg-blue-50/70'
		: 'bg-slate-50'} {showGreenFade ? 'animate-green-fade' : ''}"
>
	<div
		class="w-9 h-9 rounded-full flex items-center justify-center shrink-0 {hatZulauf
			? 'bg-blue-50 text-blue-600'
			: 'bg-slate-100 text-slate-400'}"
	>
		<Package class="h-5 w-5" aria-hidden="true" />
	</div>

	{#if hatZulauf}
		<div class="min-w-0 flex-1">
			<div class="text-sm font-bold text-slate-900">
				{totalItems} Exemplare im Zulauf
			</div>
			<div class="text-xs text-slate-500">
				aus {totalShipments} offenen {totalShipments === 1 ? 'Lieferung' : 'Lieferungen'}
			</div>
		</div>
		<button
			onclick={onOpenWareneingang}
			class="shrink-0 flex items-center gap-1.5 py-2 px-3.5 bg-slate-900 hover:bg-slate-700 text-white font-bold text-xs rounded-xl transition-colors cursor-pointer"
		>
			Einbuchen
			<span>→</span>
		</button>
	{:else}
		<div class="text-sm text-slate-400 font-medium">Kein Wareneingang im Zulauf</div>
	{/if}
</div>

<style>
	@keyframes greenGlow {
		0% {
			background-color: rgba(16, 185, 129, 0.15);
			border-color: rgba(16, 185, 129, 0.45);
		}
		50% {
			background-color: rgba(16, 185, 129, 0.3);
			border-color: rgba(16, 185, 129, 0.9);
		}
		100% {
			background-color: rgba(255, 255, 255, 1);
			border-color: rgba(226, 232, 240, 1);
		}
	}
	.animate-green-fade {
		animation: greenGlow 1.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
	}
</style>
