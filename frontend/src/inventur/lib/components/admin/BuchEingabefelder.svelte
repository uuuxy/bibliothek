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

    // Belletristik-Vorschlag: erste 3 Buchstaben des Autor-Nachnamens
    // ("Rowling, J.K." → "Row", "Joanne K. Rowling" → "Row") — die klassische
    // Freihand-Systematik. Greift nur, wenn kein Schulbuch-Track gewählt ist.
    const autorKuerzel = $derived.by(() => {
        const autor = (formular.author ?? "").trim();
        if (!autor) return "";
        const nachname = autor.includes(",") ? autor.split(",")[0] : (autor.split(/\s+/).pop() ?? "");
        const k = nachname.trim().slice(0, 3);
        return k ? k.charAt(0).toUpperCase() + k.slice(1).toLowerCase() : "";
    });

    /** Neuanlage ohne Signatur → Speichern gesperrt (Material-Error-State am Feld). */
    const signaturFehlt = $derived(!formular.id && !(formular.signatur ?? "").trim());

    $effect(() => {
        if (!formular.erweiterteEigenschaften) {
            formular.erweiterteEigenschaften = { standort: "" };
        } else if (typeof formular.erweiterteEigenschaften.standort !== "string") {
            formular.erweiterteEigenschaften.standort = "";
        }

        // Defaults for Jahrgang
        if (formular.jahrgangVon === undefined) formular.jahrgangVon = 5;
        if (formular.jahrgangBis === undefined) formular.jahrgangBis = 10;

        // Auto-Signatur-Vorschlag (bestehendes Guard-Muster: überschreibt nie
        // eine manuelle Eingabe, nur den eigenen letzten Vorschlag).
        // Ziel ist seit Migration 038 die ECHTE Spalte formular.signatur.
        let autoSig = "";
        if (formular.subject && (isLmfTrack || isBibTrack)) {
            const sys = systematikListe.find(s => s.bezeichnung === formular.subject);
            const kuerzel = sys ? sys.kuerzel : "";
            autoSig = isLmfTrack ? (kuerzel ? `LMF ${kuerzel}` : "LMF") : (kuerzel ? `BIB ${kuerzel}` : "BIB");
        } else if (!formular.id && autorKuerzel) {
            autoSig = autorKuerzel; // Belletristik/Freihand-Neuzugang
        }

        if (autoSig) {
            if (!formular.signatur || formular.signatur === lastAutoSignatur) {
                formular.signatur = autoSig;
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

    <!-- Signatur: steht physisch auf dem Buchrücken-Etikett — prominent und
         bei Neuanlage Pflicht. Die DNB-Altersstufe füllt höchstens einen
         "BIB …"-Vorschlag vor (IsbnFeld); hier entscheidet sich, ob das Buch
         zur Littera-Systematik passt. -->
    <div class="rounded-xl border-2 p-4 transition-colors {signaturFehlt ? 'border-rose-300 bg-rose-50/40' : 'border-emerald-200 bg-emerald-50/30'}">
        <label for="buch-signatur" class="flex items-center gap-2 text-sm font-bold text-gray-800 mb-1">
            🏷️ Signatur (Buchrücken)
            {#if !formular.id}<span class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded {signaturFehlt ? 'bg-rose-100 text-rose-700' : 'bg-emerald-100 text-emerald-700'}">Pflicht</span>{/if}
        </label>
        <input
            id="buch-signatur"
            type="text"
            bind:value={formular.signatur}
            placeholder={autorKuerzel ? `z. B. "${autorKuerzel}" (Belletristik) oder "LMF M"` : 'z. B. LMF M, BIB ROM, Row …'}
            aria-invalid={signaturFehlt}
            class="w-full rounded-lg px-4 py-2.5 text-gray-900 outline-none transition border bg-white
                   {signaturFehlt
                     ? 'border-rose-400 focus:ring-2 focus:ring-rose-500 focus:border-rose-500'
                     : 'border-emerald-300 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500'}"
        />
        {#if signaturFehlt}
            <p class="mt-1.5 text-xs font-semibold text-rose-600">Ohne Signatur kein Etikett — bitte Systematik-Kürzel eintragen (Speichern ist bis dahin gesperrt).</p>
        {:else}
            <p class="mt-1.5 text-xs text-gray-500">Wird 1:1 auf das Rücken-Etikett gedruckt. Bestehende Littera-Signaturen werden von Importen nie überschrieben.</p>
        {/if}
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



