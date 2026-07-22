<script>
	let { incomingShipments, showGreenFade, onOpenWareneingang } = $props();

	let totalItems = $derived(
		incomingShipments.reduce((sum, s) => sum + s.items.reduce((s2, i) => s2 + i.menge, 0), 0)
	);
	let totalShipments = $derived(incomingShipments.length);
	let hatZulauf = $derived(incomingShipments.length > 0);
</script>

<!-- Slimer Status-Streifen: im Leerzustand nur eine dezente Zeile, mit Zulauf eine
     kompakte, klickbare Karte — nie mehr die halbe Spalte für „nichts da". -->
<div
	class="rounded-2xl border px-4 py-3 flex items-center gap-3 transition-colors {hatZulauf
		? 'bg-white border-slate-200/80 shadow-sm'
		: 'bg-slate-50/60 border-slate-200/60 border-dashed'} {showGreenFade ? 'animate-green-fade' : ''}"
>
	<div
		class="w-9 h-9 rounded-full flex items-center justify-center shrink-0 {hatZulauf
			? 'bg-blue-50 text-blue-600'
			: 'bg-slate-100 text-slate-400'}"
	>
		<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
			/>
		</svg>
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
