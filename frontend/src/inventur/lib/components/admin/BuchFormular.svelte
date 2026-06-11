<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
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

<!-- Backdrop -->
<button
    class="fixed inset-0 bg-black/30 backdrop-blur-sm z-40 transition-opacity border-none cursor-default w-full h-full block"
    transition:fade={{ duration: 200 }}
    onclick={onClose}
    aria-label="Close modal"
></button>

<!-- Drawer -->
<div
    class="fixed top-0 right-0 bottom-0 w-full md:w-[480px] bg-white shadow-2xl z-50 overflow-y-auto flex flex-col"
    transition:fly={{ x: 400, duration: 300, opacity: 1 }}
>
    <!-- Drawer Header -->
    <div
        class="px-6 py-5 border-b border-gray-100 flex items-center justify-between bg-white sticky top-0 z-10"
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
        class="p-6 border-t border-gray-100 bg-gray-50 flex justify-end gap-3 sticky bottom-0"
    >
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

