<script>
	import { apiFetch, apiClient } from "../../../../lib/apiFetch.js";
    import { fly, fade } from "svelte/transition";
    import StrichcodeScannerOverlay from "$lib/components/scanner/StrichcodeScannerOverlay.svelte";
    import BuchCoverUpload from "./BuchCoverUpload.svelte";
    import BuchEingabefelder from "./BuchEingabefelder.svelte";
    import BuchExemplareListe from "./BuchExemplareListe.svelte";

    let { formular = $bindable(), onClose, onSave, onCoverUpload } = $props();

    let wirdGescannt = $state(false);

    /** @param {string} code */
    async function handleScan(code) {
        formular.isbn = code;
        if (!formular.title) {
            try {
                const res = await apiFetch(`/api/lookup/${code}`);
                if (res.ok) {
                    const json = await res.json();
                    const data = json.data;
                    if (data.title) formular.title = data.title;
                    if (data.author) formular.author = data.author;
                    if (data.verlag) formular.verlag = data.verlag;
                    if (data.jahr) formular.erscheinungsjahr = parseInt(data.jahr) || formular.erscheinungsjahr;
                    if (data.coverUrl) formular.coverUrl = data.coverUrl;
                    if (data.subject) formular.subject = data.subject;
                    if (data.grade) formular.gradeLevel = parseInt(data.grade) || formular.gradeLevel;
                }
            } catch (e) {
                console.error("Lookup failed", e);
            }
        }
    }
</script>

<StrichcodeScannerOverlay bind:isScanning={wirdGescannt} onScan={handleScan} />

<!-- Full width block -->
<div
    class="flex flex-col w-full my-4"
    transition:fade={{ duration: 200 }}
>
    <!-- Drawer Header -->
    <div
        class="px-6 py-5 border-b border-gray-100 flex items-center justify-between bg-white sticky top-0 z-10 rounded-t-2xl"
    >
        <h2 class="text-xl font-bold text-gray-900">
            {formular.id ? "Buch bearbeiten" : "Neues Buch"}
        </h2>
        <button
            onclick={onClose}
            class="p-2 hover:bg-gray-100 rounded-full text-gray-500 transition"
            aria-label="Schließen"
        >
            <svg
                class="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M6 18L18 6M6 6l12 12"
                />
            </svg>
        </button>
    </div>

    <!-- Form Content -->
    <div class="p-6 space-y-8 flex-1">
        <BuchCoverUpload bind:formular {onCoverUpload} />
        <BuchEingabefelder bind:formular bind:wirdGescannt />
        {#if formular.id}
            <BuchExemplareListe bind:formular />
        {/if}
    </div>

    <!-- Drawer Footer -->
    <div
        class="p-6 border-t border-gray-100 bg-gray-50 flex justify-end gap-3 sticky bottom-0 rounded-b-2xl"
    >
        {#if formular.id}
            <button
                onclick={() => window.open(`/api/buecher/titel/${formular.id}/etiketten`, '_blank')}
                class="px-5 py-2.5 rounded-lg text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 transition-colors mr-auto flex items-center gap-2"
                title="A4 Zweckform Etikettenbogen für dieses Buch generieren"
            >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" /></svg>
                Barcodes drucken
            </button>
        {/if}

        <button
            onclick={onClose}
            class="px-5 py-2.5 rounded-lg text-sm font-medium text-gray-600 hover:text-gray-800 hover:bg-gray-200 transition-colors"
        >
            Abbrechen
        </button>
        <button
            onclick={onSave}
            class="px-5 py-2.5 rounded-lg text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-700 shadow-md shadow-emerald-200 transition-all active:scale-95"
        >
            Speichern
        </button>
    </div>
</div>

