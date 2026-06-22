<script>
    import { klassenStufen, schulZweige } from "$lib/components/admin/buch_form_optionen.js";

    let { formular = $bindable(), systematikListe = [] } = $props();
</script>

<div class="grid grid-cols-2 gap-4">
    <div>
        <label
            for="buch-fach"
            class="block text-sm font-medium text-gray-700 mb-1">Fach</label
        >
        <div class="relative">
            <select
                id="buch-fach"
                bind:value={formular.subject}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition appearance-none cursor-pointer"
            >
                <option value="">-- Fach auswählen --</option>
                {#each systematikListe as sys (sys.id)}
                    <option value={sys.bezeichnung}>{sys.kuerzel} - {sys.bezeichnung}</option>
                {/each}
            </select>
            <div class="absolute right-3 top-3 pointer-events-none">
                <svg
                    class="h-4 w-4 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
            </div>
        </div>
    </div>
    <div>
        <label
            for="buch-klasse"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Klasse</label
        >
        <div class="relative">
            <select
                id="buch-klasse"
                bind:value={formular.gradeLevel}
                disabled={formular.track === 'Bibliothek'}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition appearance-none {formular.track === 'Bibliothek' ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}"
            >
                {#each klassenStufen as klasse (klasse)}
                    <option value={klasse}>{klasse}</option>
                {/each}
            </select>
            <div class="absolute right-3 top-3 pointer-events-none">
                <svg
                    class="h-4 w-4 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
            </div>
        </div>
    </div>
</div>

<div class="grid grid-cols-2 gap-4">
    <div>
        <label
            for="buch-jahrgang-von"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Verwendbar von Klasse</label
        >
        <input
            id="buch-jahrgang-von"
            type="number"
            min="1"
            max="13"
            bind:value={formular.jahrgangVon}
            disabled={formular.track === 'Bibliothek'}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition {formular.track === 'Bibliothek' ? 'opacity-50 cursor-not-allowed' : ''}"
        />
    </div>
    <div>
        <label
            for="buch-jahrgang-bis"
            class="block text-sm font-medium text-gray-700 mb-1"
            >bis Klasse</label
        >
        <input
            id="buch-jahrgang-bis"
            type="number"
            min="1"
            max="13"
            bind:value={formular.jahrgangBis}
            disabled={formular.track === 'Bibliothek'}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition {formular.track === 'Bibliothek' ? 'opacity-50 cursor-not-allowed' : ''}"
        />
    </div>
</div>

<div>
    <label
        for="buch-schulzweig"
        class="block text-sm font-medium text-gray-700 mb-1"
        >Schulzweig</label
    >
    <div class="relative">
        <select
            id="buch-schulzweig"
            bind:value={formular.track}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition appearance-none cursor-pointer"
        >
            {#each schulZweige as zweig (zweig)}
                <option value={zweig}>{zweig}</option>
            {/each}
        </select>
        <div class="absolute right-3 top-3 pointer-events-none">
            <svg
                class="h-4 w-4 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
        </div>
    </div>
</div>
