<script>
	let {
		klasseSearchQuery = $bindable(),
		isKlasseDropdownOpen = $bindable(),
		filteredKlassenList,
		onSelectKlasse
	} = $props();
</script>

<div class="flex justify-center">
	<div class="relative w-full max-w-md">
		<input
			role="combobox"
			type="text"
			bind:value={klasseSearchQuery}
			onfocus={() => (isKlasseDropdownOpen = true)}
			onblur={() => setTimeout(() => (isKlasseDropdownOpen = false), 150)}
			onkeydown={(e) => {
				if (e.key === 'Enter' && filteredKlassenList.length > 0) {
					onSelectKlasse(filteredKlassenList[0]);
				}
			}}
			placeholder="Klasse suchen (z.B. 5f1)..."
			aria-label="Schulklasse suchen"
			aria-expanded={isKlasseDropdownOpen}
			aria-controls="klasse-dropdown"
			class="block w-full bg-white border border-slate-200 text-slate-700 py-4 pl-6 pr-12 rounded-2xl shadow-sm hover:border-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-300 focus:border-emerald-400 transition-all duration-200 text-lg font-medium placeholder-slate-400"
		/>
		<div class="absolute inset-y-0 right-0 flex items-center pr-4 gap-2">
			{#if klasseSearchQuery}
				<button
					type="button"
					onclick={() => {
						klasseSearchQuery = '';
						isKlasseDropdownOpen = false;
					}}
					class="p-1 text-slate-400 hover:text-emerald-500 transition-colors duration-200 focus:outline-none"
					aria-label="Suche löschen"
				>
					<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			{/if}
			<div class="text-slate-400">
				<svg
					class="h-6 w-6 transition-transform duration-200 {isKlasseDropdownOpen
						? 'rotate-180'
						: ''}"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 9l-7 7-7-7"
					/>
				</svg>
			</div>
		</div>

		{#if isKlasseDropdownOpen && filteredKlassenList.length > 0}
			<ul
				id="klasse-dropdown"
				class="absolute z-10 w-full mt-2 bg-white border border-slate-100 rounded-xl shadow-lg max-h-60 overflow-y-auto py-2"
			>
				{#each filteredKlassenList as klasse (klasse)}
					<li>
						<button
							type="button"
							class="w-full text-left px-6 py-3 text-slate-700 hover:bg-emerald-50 hover:text-emerald-700 transition-colors duration-200 cursor-pointer text-lg font-medium"
							onclick={() => onSelectKlasse(klasse)}
						>
							Klasse {klasse}
						</button>
					</li>
				{/each}
			</ul>
		{:else if isKlasseDropdownOpen && filteredKlassenList.length === 0}
			<div
				class="absolute z-10 w-full mt-2 bg-white border border-slate-100 rounded-xl shadow-lg py-4 px-6 text-slate-500 text-center"
			>
				Keine Klasse gefunden.
			</div>
		{/if}
	</div>
</div>
