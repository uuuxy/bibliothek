<!--
  StartseitenHeader.svelte

  Enthält den kompletten Navigations- und Filterbereich der Startseite:
  - Tab-Umschalter (Buch-Suche / Jahrgänge / Schulklassen)
  - Das jeweilige Suchfeld/die Dropdown-Filter je nach aktivem Tab
-->
<script>
    import StartseitenHeaderTabs from "$lib/components/StartseitenHeaderTabs.svelte";
    import StartseitenHeaderSuche from "$lib/components/StartseitenHeaderSuche.svelte";
    import StartseitenHeaderJahrgaenge from "$lib/components/StartseitenHeaderJahrgaenge.svelte";
    import StartseitenHeaderKlassen from "$lib/components/StartseitenHeaderKlassen.svelte";

    // Alle Daten kommen von der Elternkomponente per Props
    let {
        viewMode = $bindable(),
        searchQuery = $bindable(),
        selectedZweig = $bindable(),
        selectedJahrgang = $bindable(),
        klasseSearchQuery = $bindable(),
        isKlasseDropdownOpen = $bindable(),
        schulzweige,
        jahrgaenge,
        filteredKlassenList,
        onSelectKlasse,
    } = $props();
</script>

<header class="pt-10 pb-8 px-4 sm:px-6 lg:px-8">
    <div class="max-w-5xl mx-auto flex flex-col items-center space-y-8">
        <StartseitenHeaderTabs
            bind:viewMode
            onWechsel={(modus) => (viewMode = modus)}
        />

        <!-- Dynamischer Filterbereich -->
        <div class="w-full max-w-3xl transition-all duration-300 ease-in-out">
            {#if viewMode === "suche"}
                <StartseitenHeaderSuche bind:searchQuery />
            {:else if viewMode === "jahrgaenge"}
                <StartseitenHeaderJahrgaenge
                    bind:selectedZweig
                    bind:selectedJahrgang
                    {schulzweige}
                    {jahrgaenge}
                />
            {:else if viewMode === "schulklassen"}
                <StartseitenHeaderKlassen
                    bind:klasseSearchQuery
                    bind:isKlasseDropdownOpen
                    {filteredKlassenList}
                    {onSelectKlasse}
                />
            {/if}
        </div>
    </div>
</header>
