<script>
	let {
		selectedZweig = $bindable(),
		selectedJahrgang = $bindable(),
		schulzweige,
		jahrgaenge,
	} = $props();
</script>

<div class="flex flex-col sm:flex-row gap-4 justify-center">
	<div class="relative w-full sm:w-64">
		<select
			bind:value={selectedZweig}
			aria-label="Nach Schulzweig filtern"
			class="appearance-none block w-full bg-white border border-slate-200 text-slate-700 py-3.5 pl-5 pr-10 rounded-2xl shadow-sm hover:border-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-300 focus:border-emerald-400 transition-all duration-200 cursor-pointer text-base"
		>
			<option value="">Alle Zweige</option>
			{#each schulzweige as zweig (zweig)}
				<option value={zweig}>{zweig}</option>
			{/each}
		</select>
		<div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-slate-400">
			<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</div>
	</div>

	<div class="relative w-full sm:w-64">
		<select
			bind:value={selectedJahrgang}
			aria-label="Nach Jahrgangsstufe filtern"
			class="appearance-none block w-full bg-white border border-slate-200 text-slate-700 py-3.5 pl-5 pr-10 rounded-2xl shadow-sm hover:border-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-300 focus:border-emerald-400 transition-all duration-200 cursor-pointer text-base"
		>
			<option value="">Alle Jahrgänge</option>
			{#each jahrgaenge as jahrgang (jahrgang)}
				<option value={jahrgang}>Klasse {jahrgang}</option>
			{/each}
		</select>
		<div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-slate-400">
			<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</div>
	</div>
</div>

