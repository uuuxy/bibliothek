<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche, Jahrgängen oder Schulklassen.
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
        viewMode = $bindable("suche"),
        searchQuery = $bindable(""),
        selectedZweig = $bindable(""),
        selectedJahrgang = $bindable(""),
        klasseSearchQuery = $bindable(""),
        isKlasseDropdownOpen = $bindable(false),
        filteredKlassenList = [],
        onSelectKlasse,
    } = $props();

    const schulzweige = ["Gymnasium", "Realschule", "Hauptschule"];
    const jahrgaenge = ["5", "6", "7", "8", "9", "10", "11", "12", "13"];
</script>

<header class="pt-10 pb-8 px-4 sm:px-6 lg:px-8">
    <div class="max-w-5xl mx-auto flex flex-col items-center space-y-8">
        <!-- Segmented Control (Tabs) -->
        <div
            class="relative flex p-1.5 bg-zinc-950/40 border border-zinc-800/40 rounded-full shadow-inner w-full max-w-md"
            role="tablist"
            aria-label="Ansichtsmodus"
        >
            <button
                class="relative flex-1 py-2 text-xs font-semibold rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50 cursor-pointer {viewMode ===
                'suche'
                    ? 'text-zinc-950 bg-emerald-500 shadow-md font-bold'
                    : 'text-zinc-400 hover:text-zinc-200'}"
                onclick={() => (viewMode = "suche")}
                role="tab"
                id="tab-suche"
                aria-selected={viewMode === "suche"}
                aria-controls="filter-suche content-suche">Buch-Suche</button
            >
            <button
                class="relative flex-1 py-2 text-xs font-semibold rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50 cursor-pointer {viewMode ===
                'jahrgaenge'
                    ? 'text-zinc-950 bg-emerald-500 shadow-md font-bold'
                    : 'text-zinc-400 hover:text-zinc-200'}"
                onclick={() => (viewMode = "jahrgaenge")}
                role="tab"
                id="tab-jahrgaenge"
                aria-selected={viewMode === "jahrgaenge"}
                aria-controls="filter-jahrgaenge content-jahrgaenge"
                >Jahrgänge</button
            >
            <button
                class="relative flex-1 py-2 text-xs font-semibold rounded-full transition-all duration-300 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50 cursor-pointer {viewMode ===
                'schulklassen'
                    ? 'text-zinc-950 bg-emerald-500 shadow-md font-bold'
                    : 'text-zinc-400 hover:text-zinc-200'}"
                onclick={() => (viewMode = "schulklassen")}
                role="tab"
                id="tab-schulklassen"
                aria-selected={viewMode === "schulklassen"}
                aria-controls="filter-schulklassen content-schulklassen"
                >Schulklassen</button
            >
        </div>

        <!-- Dynamic Filter Area -->
        <div class="w-full max-w-3xl transition-all duration-300 ease-in-out">
            {#if viewMode === "suche"}
                <div class="relative group" id="filter-suche">
                    <div
                        class="absolute inset-y-0 left-0 pl-5 flex items-center pointer-events-none"
                    >
                        <svg
                            class="h-6 w-6 text-zinc-500 group-focus-within:text-emerald-400 transition-colors duration-200"
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
                        aria-label="Suchen nach Titel, ISBN oder Autor"
                        placeholder="Titel, ISBN oder Autor suchen..."
                        class="block w-full pl-14 pr-6 py-4 bg-zinc-950 border border-zinc-850 rounded-full text-zinc-100 shadow-sm hover:border-emerald-500/50 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/50 focus:outline-none transition-all duration-300 text-lg placeholder-zinc-500"
                    />
                </div>
            {:else if viewMode === "jahrgaenge"}
                <div
                    class="flex flex-col sm:flex-row gap-4 justify-center"
                    id="filter-jahrgaenge"
                >
                    <div class="relative w-full sm:w-64">
                        <select
                            bind:value={selectedZweig}
                            aria-label="Schulzweig filtern"
                            class="appearance-none block w-full bg-zinc-950 border border-zinc-855 text-zinc-300 py-3.5 pl-5 pr-10 rounded-2xl shadow-sm hover:border-emerald-500/30 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all duration-200 cursor-pointer text-base"
                        >
                            <option value="">Alle Zweige</option>
                            {#each schulzweige as zweig (zweig)}
                                <option value={zweig}>{zweig}</option>
                            {/each}
                        </select>
                        <div
                            class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-zinc-500"
                        >
                            <svg
                                class="h-5 w-5"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                                ><path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M19 9l-7 7-7-7"
                                /></svg
                            >
                        </div>
                    </div>
                    <div class="relative w-full sm:w-64">
                        <select
                            bind:value={selectedJahrgang}
                            aria-label="Jahrgang filtern"
                            class="appearance-none block w-full bg-zinc-950 border border-zinc-855 text-zinc-300 py-3.5 pl-5 pr-10 rounded-2xl shadow-sm hover:border-emerald-500/30 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all duration-200 cursor-pointer text-base"
                        >
                            <option value="">Alle Jahrgänge</option>
                            {#each jahrgaenge as jahrgang (jahrgang)}
                                <option value={jahrgang}
                                    >Klasse {jahrgang}</option
                                >
                            {/each}
                        </select>
                        <div
                            class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-zinc-500"
                        >
                            <svg
                                class="h-5 w-5"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
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
            {:else if viewMode === "schulklassen"}
                <div class="flex justify-center" id="filter-schulklassen">
                    <div class="relative w-full max-w-md">
                        <input
                            type="text"
                            bind:value={klasseSearchQuery}
                            aria-label="Klasse suchen"
                            onfocus={() => (isKlasseDropdownOpen = true)}
                            onblur={() =>
                                setTimeout(
                                    () => (isKlasseDropdownOpen = false),
                                    150,
                                )}
                            placeholder="Klasse suchen (z.B. 5f1)..."
                            class="block w-full bg-zinc-950 border border-zinc-850 text-zinc-100 py-4 pl-6 pr-12 rounded-2xl shadow-sm hover:border-emerald-500/30 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all duration-200 text-lg font-medium placeholder-zinc-500"
                        />
                        <div
                            class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-5 text-zinc-500"
                        >
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
                        {#if isKlasseDropdownOpen && filteredKlassenList.length > 0}
                            <ul
                                class="absolute z-10 w-full mt-2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl max-h-60 overflow-y-auto py-2"
                            >
                                {#each filteredKlassenList as klasse (klasse)}
                                    <li>
                                        <button
                                            type="button"
                                            class="w-full text-left px-6 py-3 text-zinc-300 hover:bg-emerald-500/10 hover:text-emerald-400 transition-colors duration-200 cursor-pointer text-lg font-medium"
                                            onclick={() =>
                                                onSelectKlasse?.(klasse)}
                                        >
                                            Klasse {klasse}
                                        </button>
                                    </li>
                                {/each}
                            </ul>
                        {:else if isKlasseDropdownOpen && filteredKlassenList.length === 0}
                            <div
                                class="absolute z-10 w-full mt-2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl py-4 px-6 text-zinc-500 text-center"
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

