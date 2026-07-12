<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche, Jahrgängen oder Schulklassen.
	Refactored: Clean SaaS light-mode design with Google-style tabs.
-->
<script>
	/**
	 * @type {{
	 *   viewMode: string,
	 *   searchQuery: string,
	 *   selectedZweig: string,
	 *   selectedJahrgang: string,
	 *   klasseSearchQuery: string,
	 *   isKlasseDropdownOpen: boolean,
	 *   filteredKlassenList: string[],
	 *   onSelectKlasse: (klasse: string) => void
	 * }}
	 */
	let {
		viewMode = $bindable('suche'),
		searchQuery = $bindable(''),
		selectedZweig = $bindable(''),
		selectedJahrgang = $bindable(''),
		klasseSearchQuery = $bindable(''),
		isKlasseDropdownOpen = $bindable(false),
		filteredKlassenList = [],
		onSelectKlasse
	} = $props();

	const schulzweige = ['Gymnasium', 'Realschule', 'Hauptschule'];
	const jahrgaenge = ['5', '6', '7', '8', '9', '10', '11', '12', '13'];
</script>

<header class="pt-6 pb-6 px-4 sm:px-6 lg:px-8">
	<div class="max-w-5xl mx-auto flex flex-col items-center space-y-6">
		<!-- Google-style underline Tabs -->
		<div
			class="border-b border-slate-200 w-full max-w-md"
			role="tablist"
			aria-label="Ansichtsmodus"
		>
			<nav class="flex gap-6 justify-center">
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'suche'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'suche')}
					role="tab"
					id="tab-suche"
					aria-selected={viewMode === 'suche'}
					aria-controls="filter-suche content-suche"
				>
					Buch-Suche
					{#if viewMode === 'suche'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'jahrgaenge'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'jahrgaenge')}
					role="tab"
					id="tab-jahrgaenge"
					aria-selected={viewMode === 'jahrgaenge'}
					aria-controls="filter-jahrgaenge content-jahrgaenge"
				>
					Jahrgänge
					{#if viewMode === 'jahrgaenge'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'schulklassen'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'schulklassen')}
					role="tab"
					id="tab-schulklassen"
					aria-selected={viewMode === 'schulklassen'}
					aria-controls="filter-schulklassen content-schulklassen"
				>
					Schulklassen
					{#if viewMode === 'schulklassen'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
			</nav>
		</div>

		<!-- Dynamic Filter Area -->
		<div class="w-full max-w-3xl transition-all duration-300 ease-in-out">
			{#if viewMode === 'suche'}
				<div class="relative group" id="filter-suche">
					<div class="absolute inset-y-0 left-0 pl-5 flex items-center pointer-events-none">
						<svg
							class="h-5 w-5 text-slate-400 group-focus-within:text-blue-500 transition-colors duration-200"
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
						aria-label="Suchen nach Titel, Fach, Klasse oder Autor"
						placeholder="Suchen nach Titel, Fach, Klasse (z.B. 'Mathe 5' oder 'Gymnasium')..."
						class="block w-full pl-14 pr-6 py-3.5 bg-white border border-slate-300 rounded-xl text-slate-800 shadow-sm hover:border-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all duration-200 text-sm placeholder-slate-400"
					/>
				</div>
			{:else if viewMode === 'jahrgaenge'}
				<div class="flex flex-col sm:flex-row gap-3 justify-center" id="filter-jahrgaenge">
					<div class="relative w-full sm:w-56">
						<select
							bind:value={selectedZweig}
							aria-label="Schulzweig filtern"
							class="appearance-none block w-full bg-white border border-slate-300 text-slate-800 py-3 pl-4 pr-10 rounded-xl shadow-sm hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 cursor-pointer text-sm"
						>
							<option value="">Alle Zweige</option>
							{#each schulzweige as zweig (zweig)}
								<option value={zweig}>{zweig}</option>
							{/each}
						</select>
						<div
							class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-slate-400"
						>
							<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 9l-7 7-7-7"
								/></svg
							>
						</div>
					</div>
					<div class="relative w-full sm:w-56">
						<select
							bind:value={selectedJahrgang}
							aria-label="Jahrgang filtern"
							class="appearance-none block w-full bg-white border border-slate-300 text-slate-800 py-3 pl-4 pr-10 rounded-xl shadow-sm hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 cursor-pointer text-sm"
						>
							<option value="">Alle Jahrgänge</option>
							{#each jahrgaenge as jahrgang (jahrgang)}
								<option value={jahrgang}>Klasse {jahrgang}</option>
							{/each}
						</select>
						<div
							class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-slate-400"
						>
							<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 9l-7 7-7-7"
								/></svg
							>
						</div>
					</div>
				</div>
			{:else if viewMode === 'schulklassen'}
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
								class="h-5 w-5 transition-transform duration-200 {isKlasseDropdownOpen
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
						{#if isKlasseDropdownOpen && filteredKlassenList.length > 0}
							<ul
								class="absolute z-10 w-full mt-1.5 bg-white border border-slate-200 rounded-xl shadow-lg max-h-60 overflow-y-auto py-1"
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
								class="absolute z-10 w-full mt-1.5 bg-white border border-slate-200 rounded-xl shadow-lg py-4 px-5 text-slate-400 text-center text-sm"
							>
								Keine Klasse gefunden.
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
</header>
