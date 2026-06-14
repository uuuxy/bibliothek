<script>
	import { apiFetch, apiClient } from "../../../../lib/apiFetch.js";
    import { onMount } from "svelte";
    import IsbnFeld from "./IsbnFeld.svelte";
    import { klassenStufen, schulZweige } from "$lib/components/admin/buch_form_optionen.js";

    let { formular = $bindable(), wirdGescannt = $bindable() } = $props();

    /** @type {any[]} */
    let systematikListe = $state([]);

    onMount(async () => {
        try {
            const antwort = await apiFetch("/api/systematics");
            if (antwort.ok) {
                systematikListe = await antwort.json() || [];
            }
        } catch (fehler) {
            console.error("Fehler beim Laden der Systematik", fehler);
        }
    });

    import { lmfFaecher, bibKategorien } from "./signatur_optionen.js";

    let lastAutoSignatur = "";
    
    // Computed states for the template
    let isLmfTrack = $derived(["Gymnasium", "Realschule", "Hauptschule", "Förderstufe", "Oberstufe"].includes(formular.track));
    let isBibTrack = $derived(formular.track === "Bibliothek");

    $effect(() => {
        if (!formular.erweiterteEigenschaften) {
            formular.erweiterteEigenschaften = { standort: "", signatur: "" };
        } else {
            if (typeof formular.erweiterteEigenschaften.standort !== "string") {
                formular.erweiterteEigenschaften.standort = "";
            }
            if (typeof formular.erweiterteEigenschaften.signatur !== "string") {
                formular.erweiterteEigenschaften.signatur = "";
            }
        }
        
        // Defaults for Jahrgang
        if (formular.jahrgangVon === undefined) formular.jahrgangVon = 5;
        if (formular.jahrgangBis === undefined) formular.jahrgangBis = 10;

        // Auto-Generate Signatur
        let autoSig = "";

        if (formular.subject) {
            const sys = systematikListe.find(s => s.bezeichnung === formular.subject);
            const kuerzel = sys ? sys.kuerzel : "";
            if (isLmfTrack) {
                autoSig = kuerzel ? `LMF ${kuerzel}` : "LMF";
            } else if (isBibTrack) {
                autoSig = kuerzel ? `BIB ${kuerzel}` : "BIB";
            }
        }

        if (autoSig) {
            if (!formular.erweiterteEigenschaften.signatur || formular.erweiterteEigenschaften.signatur === lastAutoSignatur) {
                formular.erweiterteEigenschaften.signatur = autoSig;
                lastAutoSignatur = autoSig;
            }
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

    <div>
        <label
            for="buch-untertitel"
            class="block text-sm font-medium text-gray-700 mb-1">Untertitel</label
        >
        <input
            id="buch-untertitel"
            type="text"
            bind:value={formular.untertitel}
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
                for="buch-verlag"
                class="block text-sm font-medium text-gray-700 mb-1"
                >Verlag</label
            >
            <input
                id="buch-verlag"
                type="text"
                bind:value={formular.verlag}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
            />
        </div>
        <div>
            <label
                for="buch-jahr"
                class="block text-sm font-medium text-gray-700 mb-1"
                >Erscheinungsjahr</label
            >
            <input
                id="buch-jahr"
                type="number"
                bind:value={formular.erscheinungsjahr}
                class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
            />
        </div>
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
            for="buch-beschreibung"
            class="block text-sm font-medium text-gray-700 mb-1">Beschreibung / Klappentext</label
        >
        <textarea
            id="buch-beschreibung"
            rows="3"
            bind:value={formular.beschreibung}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
        ></textarea>
    </div>

    {#if formular.erweiterteEigenschaften}
        <div class="grid grid-cols-2 gap-4">
            <div>
                <label for="buch-signatur" class="block text-sm font-medium text-gray-700 mb-1">Signatur</label>
                <input
                    id="buch-signatur"
                    type="text"
                    bind:value={formular.erweiterteEigenschaften.signatur}
                    placeholder="z. B. LMF M, BIB ROM, ..."
                    class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
                />
            </div>
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
</div>



