<!-- @component KlassenSuchfeld — Klassensuche mit Vorschlagsliste für die Startseite.

     Bewusst KEIN Select: Hier wird getippt und gefiltert, nicht aus einer festen
     Liste gewählt — bei über hundert Klassen ist Tippen der kürzere Weg. Der
     150-ms-Verzug beim Verlassen ist Absicht: Ohne ihn schließt die Liste, bevor
     der Klick auf einen Eintrag ankommt. -->
<script>
	/**
	 * @type {{
	 *   klasseSearchQuery: string,
	 *   isKlasseDropdownOpen: boolean,
	 *   filteredKlassenList: string[],
	 *   onSelectKlasse?: (klasse: string) => void
	 * }}
	 */
	let {
		klasseSearchQuery = $bindable(''),
		isKlasseDropdownOpen = $bindable(false),
		filteredKlassenList = [],
		onSelectKlasse
	} = $props();
</script>

<div class="flex justify-center" id="filter-schulklassen">
	<div class="relative w-full max-w-md">
		<input
			type="text"
			bind:value={klasseSearchQuery}
			aria-label="Klasse suchen"
			onfocus={() => (isKlasseDropdownOpen = true)}
			onblur={() => setTimeout(() => (isKlasseDropdownOpen = false), 150)}
			placeholder="Klasse suchen (z.B. 5f1)..."
			class="block w-full bg-white border border-slate-300 text-slate-800 py-3.5 pl-5 pr-12 rounded-xl shadow-sm hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 text-sm font-medium placeholder-slate-400"
		/>
		<div
			class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-slate-400"
		>
			<svg
				class="h-5 w-5 transition-transform duration-200 {isKlasseDropdownOpen ? 'rotate-180' : ''}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</div>
		{#if isKlasseDropdownOpen && filteredKlassenList.length > 0}
			<ul
				class="absolute z-10 w-full mt-1.5 bg-surface-container rounded-sm shadow-xl max-h-60 overflow-y-auto py-1"
			>
				{#each filteredKlassenList as klasse (klasse)}
					<li>
						<button
							type="button"
							class="w-full text-left px-5 py-2.5 text-slate-700 hover:bg-blue-50 hover:text-blue-700 transition-colors duration-150 cursor-pointer text-sm font-medium"
							onclick={() => onSelectKlasse?.(klasse)}
						>
							Klasse {klasse}
						</button>
					</li>
				{/each}
			</ul>
		{:else if isKlasseDropdownOpen && filteredKlassenList.length === 0}
			<div
				class="absolute z-10 w-full mt-1.5 bg-surface-container rounded-sm shadow-xl py-4 px-5 text-slate-400 text-center text-sm"
			>
				Keine Klasse gefunden.
			</div>
		{/if}
	</div>
</div>
