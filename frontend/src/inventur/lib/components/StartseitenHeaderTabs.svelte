<script>
	let { viewMode = $bindable(), onWechsel } = $props();
</script>

<div
	class="relative flex p-1 bg-slate-200/70 rounded-full shadow-inner w-full max-w-md"
	role="tablist"
>
	<button
		role="tab"
		aria-selected={viewMode === "suche"}
		class="relative flex-1 py-2.5 text-sm font-medium rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2 {viewMode ===
		'suche'
			? 'text-emerald-900 shadow-sm bg-emerald-100'
			: 'text-slate-600 hover:text-slate-900 hover:bg-slate-200/50'}"
		onclick={() => onWechsel("suche")}
	>
		Buch-Suche
	</button>
	<button
		role="tab"
		aria-selected={viewMode === "jahrgaenge"}
		class="relative flex-1 py-2.5 text-sm font-medium rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2 {viewMode ===
		'jahrgaenge'
			? 'text-emerald-900 shadow-sm bg-emerald-100'
			: 'text-slate-600 hover:text-slate-900 hover:bg-slate-200/50'}"
		onclick={() => onWechsel("jahrgaenge")}
	>
		Jahrgänge
	</button>
	<button
		role="tab"
		aria-selected={viewMode === "schulklassen"}
		class="relative flex-1 py-2.5 text-sm font-medium rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2 {viewMode ===
		'schulklassen'
			? 'text-emerald-900 shadow-sm bg-emerald-100'
			: 'text-slate-600 hover:text-slate-900 hover:bg-slate-200/50'}"
		onclick={() => onWechsel("schulklassen")}
	>
		Schulklassen
	</button>
</div>

