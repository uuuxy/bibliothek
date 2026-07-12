<script>
	let { searchQuery = $bindable() } = $props();
</script>

<div class="relative group">
	<div class="absolute inset-y-0 left-0 pl-5 flex items-center pointer-events-none">
		<svg
			class="h-6 w-6 text-slate-400 group-focus-within:text-emerald-500 transition-colors duration-200"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
			/>
		</svg>
	</div>
	<input
		type="text"
		bind:value={searchQuery}
		placeholder="Titel, ISBN oder Autor suchen..."
		aria-label="Suchbegriff für Bücher eingeben"
		class="block w-full pl-14 pr-12 py-4 bg-white border border-slate-200 rounded-full text-slate-900 shadow-sm hover:border-emerald-300 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-300 focus:outline-none transition-all duration-300 text-lg placeholder-slate-400"
	/>
	{#if searchQuery}
		<button
			type="button"
			onclick={() => (searchQuery = '')}
			class="absolute inset-y-0 right-0 pr-5 flex items-center text-slate-400 hover:text-emerald-500 transition-colors duration-200 focus:outline-none"
			aria-label="Suche löschen"
		>
			<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
		</button>
	{/if}
</div>
