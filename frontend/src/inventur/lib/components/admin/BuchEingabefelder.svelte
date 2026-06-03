<script>
    import { onMount } from "svelte";
    import IsbnFeld from "./IsbnFeld.svelte";
    import { klassenStufen, schulZweige } from "$lib/components/admin/buch_form_optionen.js";

    let { formular = $bindable(), wirdGescannt = $bindable() } = $props();

    let faecherListe = $state(/** @type {string[]} */ ([]));

    onMount(async () => {
        try {
            const antwort = await fetch("/api/subjects");
            if (antwort.ok) {
                const json = await antwort.json();
                faecherListe = (json.data || []).map((/** @type {any} */ fach) => fach.name);
            }
        } catch (fehler) {
            console.error("Fehler beim Laden der Fächer", fehler);
        }
    });

    $effect(() => {
        if (!formular.erweiterteEigenschaften) {
            formular.erweiterteEigenschaften = { standort: "" };
        } else if (typeof formular.erweiterteEigenschaften.standort !== "string") {
            formular.erweiterteEigenschaften.standort = "";
        }
    });
</script>

<div class="space-y-5">
    <div>
        <label
            for="buch-medientyp"
            class="block text-sm font-medium text-gray-700 mb-1">Medientyp</label
        >
        <div class="relative">
            <select
                id="buch-medientyp"
                bind:value={formular.medientyp}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition appearance-none cursor-pointer"
            >
                <option value="Buch">Buch</option>
                <option value="CD">CD</option>
                <option value="DVD">DVD</option>
            </select>
            <div class="absolute right-3 top-3 pointer-events-none">
                <svg
                    class="h-4 w-4 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
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
    </div>

    <div>
        <label
            for="buch-titel"
            class="block text-sm font-medium text-gray-700 mb-1">Titel</label
        >
        <input
            id="buch-titel"
            type="text"
            bind:value={formular.title}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
        />
    </div>

    <div class="grid grid-cols-2 gap-4">
        <div>
            <label
                for="buch-autor"
                class="block text-sm font-medium text-gray-700 mb-1"
                >{formular.medientyp === 'DVD' ? 'Regisseur' : 'Autor'}</label
            >
            <input
                id="buch-autor"
                type="text"
                bind:value={formular.author}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
            />
        </div>

        <!-- Extrahierte ISBN-Feld-Komponente -->
        <IsbnFeld bind:formular bind:wirdGescannt />
    </div>

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
                    {#each faecherListe as fach (fach)}
                        <option value={fach}>{fach}</option>
                    {/each}
                </select>
                <div class="absolute right-3 top-3 pointer-events-none">
                    <svg
                        class="h-4 w-4 text-gray-400"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
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
                    class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition appearance-none cursor-pointer"
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
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M19 9l-7 7-7-7"
                        />
                    </svg>
                </div>
            </div>
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
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M19 9l-7 7-7-7"
                    />
                </svg>
            </div>
        </div>
    </div>

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

    <div>
        <label
            for="buch-standort"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Standort / Regal</label
        >
        <input
            id="buch-standort"
            type="text"
            bind:value={formular.erweiterteEigenschaften.standort}
            placeholder="z. B. Krimi-Ecke oder Regal 3B"
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
        />
    </div>
</div>



