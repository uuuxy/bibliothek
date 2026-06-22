<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import AntolinBadge from './AntolinBadge.svelte';
  import BookBorrowersList from './BookBorrowersList.svelte';
  import BookCopiesManager from './BookCopiesManager.svelte';
  import { appState } from "../inventur/lib/store.svelte.js";

  // Props
  let { title = { id: "1", titel: "LMF-Mathe 9", autor: "Dr. L. Müller", verlag: "Klett", erscheinungsjahr: 2023 } } = $props();

  // State Runes
  /** @type {any[]} */
  let copies = $state.raw([]);
  let loadingCopies = $state(false);

  /** @type {any[]} */
  let borrowers = $state.raw([]);
  let loadingBorrowers = $state(false);

  async function loadBorrowers() {
    if (!title || !title.id || title.id === "1") {
      borrowers = [];
      return;
    }
    loadingBorrowers = true;
    try {
      const res = await apiFetch(`/api/buecher/titel/${title.id}/ausleiher`);
      if (res.ok) {
        borrowers = await res.json() || [];
      } else {
        borrowers = [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Ausleiher:", err);
      borrowers = [];
    } finally {
      loadingBorrowers = false;
    }
  }

  async function loadCopies() {
    if (!title || !title.id || title.id === "1") {
      copies = [
        { id: "e1", barcode_id: "B-20031", zustand_notiz: "Leichte Kratzer", ist_ausleihbar: true },
        { id: "e2", barcode_id: "B-20032", zustand_notiz: "Eselsohren auf S. 12", ist_ausleihbar: true },
        { id: "e3", barcode_id: "B-20033", zustand_notiz: "Unleserlicher Barcode", ist_ausleihbar: true },
        { id: "e4", barcode_id: "B-20034", zustand_notiz: "Neuwertig", ist_ausleihbar: true }
      ];
      return;
    }

    loadingCopies = true;
    try {
      const res = await apiFetch(`/api/buecher/titel/${title.id}/exemplare`);
      if (res.ok) {
        copies = await res.json();
      } else {
        copies = [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Exemplare:", err);
      copies = [];
    } finally {
      loadingCopies = false;
    }
  }

  $effect(() => {
    loadCopies();
    loadBorrowers();
  });
</script>

{#if !title}
  <div class="py-12 flex flex-col items-center justify-center text-slate-400 space-y-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
    <span class="text-xs font-semibold">Kein Buch ausgewählt. Bitte suche einen Buchtitel über die Ausleihe.</span>
  </div>
{:else}
  <div class="w-full space-y-6 text-slate-800">
    
    <!-- Title Info Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-100 pb-5">
      <div class="flex items-center space-x-4">
        {#if title.coverUrl}
          <img src={title.coverUrl} class="w-14 h-20 object-cover rounded-xl shadow-md border border-slate-100/50 shrink-0" alt="Cover" />
        {:else}
          <div class="w-14 h-20 rounded-xl shadow-md shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-650 text-lg border border-indigo-600/10">
            {title.titel ? title.titel.charAt(0).toUpperCase() : '?'}
          </div>
        {/if}
        <div>
          <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Lehrmittelfreiheit (LMF) Klassensatz</span>
          <h2 class="text-2xl font-bold text-slate-900 leading-tight">{title.titel}</h2>
          <p class="text-xs text-slate-500">
            {title.medientyp === 'DVD' ? 'Regisseur' : 'Autor'}: {title.autor} · 
            {title.medientyp === 'CD' || title.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {title.isbn || '-'} · 
            Verlag: {title.verlag} ({title.erscheinungsjahr})
          </p>
          {#if title.isbn && title.medientyp !== 'CD' && title.medientyp !== 'DVD'}
            <AntolinBadge isbn={title.isbn} />
          {/if}
          {#if title.erweiterteEigenschaften?.standort || title.erweiterteEigenschaften?.signatur}
            <div class="mt-1.5 flex flex-wrap gap-2">
              {#if title.erweiterteEigenschaften?.standort}
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-amber-50 text-amber-800 border border-amber-100">
                  📍 Standort: {title.erweiterteEigenschaften.standort}
                </span>
              {/if}
              {#if title.erweiterteEigenschaften?.signatur}
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-blue-50 text-blue-800 border border-blue-100">
                  📚 Signatur: {title.erweiterteEigenschaften.signatur}
                </span>
              {/if}
            </div>
          {/if}
        </div>
      </div>
      <div class="text-sm bg-slate-50 border border-slate-100 rounded-2xl py-2 px-4 flex items-center gap-3">
        <span class="text-slate-400">Exemplare:</span>
        <span class="font-bold text-slate-700">{copies.length} im Bestand</span>
      </div>
    </div>

    <BookCopiesManager bind:copies {title} {loadingCopies} />

    <!-- Active Borrowers List -->
    <BookBorrowersList {borrowers} />
  </div>

{/if}
