<script>
	import { apiFetch, apiClient } from "../../../../lib/apiFetch.js";
    import { onMount } from "svelte";
    import IsbnFeld from "./IsbnFeld.svelte";
    import BuchEingabefelderKategorisierung from "./BuchEingabefelderKategorisierung.svelte";
    import BuchEingabefelderInventar from "./BuchEingabefelderInventar.svelte";
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

    <BuchEingabefelderKategorisierung bind:formular {systematikListe} />

    <BuchEingabefelderInventar bind:formular />

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


</div>



