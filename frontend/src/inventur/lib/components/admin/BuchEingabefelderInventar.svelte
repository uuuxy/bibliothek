<script>
    let { formular = $bindable() } = $props();
</script>

<div class="grid grid-cols-2 gap-4">
    <div>
        <label
            for="buch-bestand"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Aktueller Bestand</label
        >
        <div class="relative">
            <input
                id="buch-bestand"
                type="number"
                bind:value={formular.stock}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
            />
            <div class="absolute right-3 top-2.5 text-gray-400 text-sm">
                Stück
            </div>
        </div>
    </div>
    <div>
        <label
            for="buch-zaehldatum"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Zähldatum</label
        >
        <input
            id="buch-zaehldatum"
            type="date"
            bind:value={formular.lastCounted}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
        />
    </div>
</div>

{#if formular.erweiterteEigenschaften}
    <!-- Signatur lebt jetzt als Pflichtfeld direkt unter Titel/Autor
         (BuchEingabefelder) und schreibt die echte DB-Spalte. -->
    <div class="grid grid-cols-2 gap-4">
        <div>
            <label for="buch-standort" class="block text-sm font-medium text-gray-700 mb-1">Standort / Regal</label>
            <input
                id="buch-standort"
                type="text"
                bind:value={formular.erweiterteEigenschaften.standort}
                placeholder="z. B. Krimi-Ecke oder Regal 3B"
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
            />
        </div>
    </div>
{/if}
