<script>
  import { showToast } from "../inventur/lib/store.svelte.js";
  import BookBorrowersTab from "./BookBorrowersTab.svelte";
  import BookExemplareTab from "./BookExemplareTab.svelte";
  import BookHistoryTab from "./BookHistoryTab.svelte";
  import BookVormerkungenTab from "./BookVormerkungenTab.svelte";
  import BookAkteMeta from "./BookAkteMeta.svelte";
  import { useBookAkte } from "./useBookAkte.svelte.js";

  /** @type {{ bookId: string | null, onBack: () => void }} */
  let { bookId, onBack } = $props();

  const akte = useBookAkte();

  $effect(() => {
    if (bookId) akte.loadAll(bookId);
  });
</script>

<div class="w-full max-w-6xl mx-auto space-y-6 animate-fade-in">
  <!-- Back Button + Breadcrumb -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <button
        onclick={onBack}
        class="flex items-center gap-2 px-3 py-2 rounded-xl text-slate-500 hover:text-slate-800 hover:bg-slate-100 transition-all text-sm font-semibold cursor-pointer"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        Zurück zum Katalog
      </button>
      <span class="text-slate-300">/</span>
      <span class="text-slate-500 text-sm truncate max-w-xs">{akte.book?.title ?? "Lade..."}</span>
    </div>
    {#if !akte.isLoading && akte.book}
      <div class="flex items-center gap-2">
        <button
          onclick={akte.editTitle}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-blue-600 hover:text-blue-700 hover:bg-blue-50 transition-all text-xs font-semibold cursor-pointer border border-blue-200"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
          Titel bearbeiten
        </button>
        <button
          onclick={() => akte.deleteTitle(showToast, onBack)}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-rose-600 hover:text-rose-700 hover:bg-rose-50 transition-all text-xs font-semibold cursor-pointer border border-rose-200"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
          Gesamten Titel löschen
        </button>
      </div>
    {/if}
  </div>

  {#if akte.isLoading}
    <div class="flex justify-center items-center py-32">
      <div class="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if akte.book}
    <BookAkteMeta 
      book={akte.book} 
      borrowers={akte.borrowers} 
      exemplare={akte.exemplare} 
      coverSrc={akte.coverSrc} 
      coverFailed={akte.coverFailed} 
      onCoverError={akte.onCoverError} 
      onCoverLoad={akte.onCoverLoad} 
    />

    <!-- Tabs -->
    <div class="border-b border-slate-200">
      <nav class="flex gap-6 overflow-x-auto no-scrollbar" aria-label="Buch-Akte Tabs">
        {#each [
          ["ausleiher", `Ausleiher (${akte.borrowers.length})`], 
          ["exemplare", `Exemplare (${akte.exemplare.length})`], 
          ["vormerkungen", `Vormerkungen (${akte.vormerkungen.length})`],
          ["historie", "Historie"]
        ] as [id, label]}
          <button
            onclick={() => akte.activeTab = id}
            class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {akte.activeTab === id ? 'text-blue-600' : 'text-slate-500 hover:text-slate-700'}"
            role="tab"
            aria-selected={akte.activeTab === id}
          >
            {label}
            {#if akte.activeTab === id}
              <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
            {/if}
          </button>
        {/each}
      </nav>
    </div>

    <!-- Tab Content -->
    <div class="w-full">
      {#if akte.activeTab === "ausleiher"}
        <BookBorrowersTab borrowers={akte.borrowers} book={akte.book} {onBack} />
      {:else if akte.activeTab === "exemplare"}
        <BookExemplareTab bind:exemplare={akte.exemplare} book={akte.book} loadAll={akte.loadAll} />
      {:else if akte.activeTab === "historie"}
        <BookHistoryTab history={akte.history} />
      {:else if akte.activeTab === "vormerkungen"}
        <BookVormerkungenTab bind:vormerkungen={akte.vormerkungen} book={akte.book} />
      {/if}
    </div>
  {:else}
    <div class="py-24 flex flex-col items-center text-slate-400 gap-3">
      <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <p class="font-semibold">Buch nicht gefunden.</p>
      <button onclick={onBack} class="text-blue-600 text-sm font-semibold hover:underline cursor-pointer">Zurück zum Katalog</button>
    </div>
  {/if}
</div>
