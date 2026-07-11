<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  /** @type {{ user: any }} */
  let { user } = $props();

  let searchQuery = $state("");
  let searchResults = $state.raw(/** @type {any[]} */ ([]));
  let isSearching = $state(false);

  // Per-book form state
  let reservierungForms = $state(/** @type {Record<string, { open: boolean, klasse: string, anzahl: number, notiz: string, loading: boolean, success: string|null, error: string|null }>} */ ({}));

  let searchTimeout = /** @type {any} */ (null);

  $effect(() => {
    const q = searchQuery;
    clearTimeout(searchTimeout);
    if (q.trim().length < 2) {
      searchResults = [];
      return () => clearTimeout(searchTimeout);
    }
    searchTimeout = setTimeout(async () => {
      isSearching = true;
      try {
        const res = await apiFetch(`/api/search?q=${encodeURIComponent(q)}&type=books`);
        if (res.ok) {
          const data = await res.json();
          searchResults = data.books ?? data ?? [];
        }
      } catch { /* ignore */ } finally {
        isSearching = false;
      }
    }, 300);
    return () => clearTimeout(searchTimeout);
  });

  /**
   * Legt das Formular-Objekt für einen Titel an, falls es fehlt.
   * Darf NUR aus Event-Handlern/asynchronem Code aufgerufen werden —
   * eine Zuweisung an $state während des Template-Renderns wirft in
   * Svelte 5 `state_unsafe_mutation` und bricht das Rendern der
   * Suchtreffer komplett ab (so konnten Lehrkräfte real nicht suchen).
   * @param {string} titelId
   */
  function ensureForm(titelId) {
    if (!reservierungForms[titelId]) {
      reservierungForms[titelId] = {
        open: false,
        klasse: user?.klasse ?? "",
        anzahl: 1,
        notiz: "",
        loading: false,
        success: null,
        error: null,
      };
    }
    return reservierungForms[titelId];
  }

  /**
   * Reine Lese-Sicht fürs Template — mutiert nie.
   * @param {string} titelId
   */
  function getForm(titelId) {
    return (
      reservierungForms[titelId] ?? {
        open: false,
        klasse: user?.klasse ?? "",
        anzahl: 1,
        notiz: "",
        loading: false,
        success: null,
        error: null,
      }
    );
  }

  /**
   * @param {string} titelId
   */
  function toggleForm(titelId) {
    const f = ensureForm(titelId);
    f.open = !f.open;
    f.success = null;
    f.error = null;
  }

  /**
   * @param {string} titelId
   */
  async function submitReservierung(titelId) {
    const f = ensureForm(titelId);
    if (!f.klasse.trim()) { f.error = "Bitte Klasse angeben."; return; }
    f.loading = true;
    f.error = null;
    f.success = null;
    try {
      const res = await apiFetch("/api/reservierungen/klassensatz", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ titel_id: titelId, klasse: f.klasse, anzahl: f.anzahl, notiz: f.notiz }),
      });
      if (res.ok) {
        f.success = "Reservierungsanfrage wurde gesendet!";
        f.open = false;
      } else {
        const txt = await res.text();
        f.error = txt || "Fehler beim Senden.";
      }
    } catch (e) {
      f.error = String(e);
    } finally {
      f.loading = false;
    }
  }
</script>

<div class="max-w-4xl mx-auto space-y-8">
  <!-- Header -->
  <div>
    <h1 class="text-2xl font-bold text-slate-800">Mein Lehrerportal</h1>
    <p class="text-sm text-slate-500 mt-1">Suche im Bibliothekskatalog und reserviere Klassensätze für deinen Unterricht.</p>
  </div>

  <!-- Search bar -->
  <div class="relative">
    <svg xmlns="http://www.w3.org/2000/svg" class="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-slate-400 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
    <input
      type="search"
      bind:value={searchQuery}
      placeholder="Titel, Autor oder ISBN suchen …"
      class="w-full pl-12 pr-4 py-3 rounded-2xl bg-white border border-slate-200 shadow-xs focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 text-slate-800 placeholder-slate-400 transition-all"
    />
    {#if isSearching}
      <div class="absolute right-4 top-1/2 -translate-y-1/2 w-4 h-4 border-2 border-blue-500/40 border-t-blue-500 rounded-full animate-spin"></div>
    {/if}
  </div>

  <!-- Results -->
  {#if searchResults.length > 0}
    <div class="space-y-4">
      {#each searchResults as book (book.id ?? book.titel_id)}
        {@const titelId = book.id ?? book.titel_id}
        {@const form = getForm(titelId)}
        <div class="w-full">
          <div class="flex gap-4 p-4">
            <!-- Cover -->
            <div class="w-16 h-20 rounded-xl bg-slate-100 border border-slate-200 shrink-0 overflow-hidden flex items-center justify-center">
              {#if book.cover_url}
                <img src={book.cover_url} alt="Cover" class="w-full h-full object-cover" loading="lazy" />
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
              {/if}
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-slate-800 text-sm leading-tight truncate">{book.titel ?? book.title ?? "Unbekannter Titel"}</h3>
              <p class="text-xs text-slate-500 mt-0.5">{book.autor ?? book.author ?? ""}</p>
              {#if book.isbn}
                <p class="text-[10px] text-slate-400 mt-1">ISBN {book.isbn}</p>
              {/if}
              {#if book.verfuegbare_exemplare != null}
                <p class="text-xs mt-1.5">
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold {book.verfuegbare_exemplare > 0 ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-600'}">
                    {book.verfuegbare_exemplare > 0 ? `${book.verfuegbare_exemplare} verfügbar` : 'nicht verfügbar'}
                  </span>
                </p>
              {/if}
            </div>

            <!-- Action -->
            <div class="shrink-0 flex flex-col items-end justify-between">
              {#if form.success}
                <span class="text-xs text-emerald-600 font-semibold">✓ Gesendet</span>
              {:else}
                <button
                  onclick={() => toggleForm(titelId)}
                  class="px-3 py-1.5 rounded-xl text-xs font-bold transition-all {form.open ? 'bg-slate-100 text-slate-600' : 'bg-blue-600 hover:bg-blue-700 text-white'}"
                >
                  {form.open ? 'Abbrechen' : 'Klassensatz reservieren'}
                </button>
              {/if}
            </div>
          </div>

          <!-- Inline reservation form -->
          {#if form.open}
            <div class="border-t border-slate-100 bg-slate-50 px-4 py-4">
              <p class="text-xs font-semibold text-slate-600 mb-3">Klassensatz-Reservierung</p>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label for="klasse-{titelId}" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Klasse *</label>
                  <input
                    id="klasse-{titelId}"
                    type="text"
                    bind:value={form.klasse}
                    placeholder="z. B. 8b"
                    class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400"
                  />
                </div>
                <div>
                  <label for="anzahl-{titelId}" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Anzahl</label>
                  <input
                    id="anzahl-{titelId}"
                    type="number"
                    bind:value={form.anzahl}
                    min="1"
                    max="200"
                    class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400"
                  />
                </div>
              </div>
              <div class="mt-3">
                <label for="notiz-{titelId}" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Notiz (optional)</label>
                <textarea
                  id="notiz-{titelId}"
                  bind:value={form.notiz}
                  rows="2"
                  placeholder="z. B. Benötigt ab 15. September …"
                  class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 resize-none"
                ></textarea>
              </div>
              {#if form.error}
                <p class="text-xs text-rose-500 mt-2">{form.error}</p>
              {/if}
              <div class="mt-3 flex justify-end">
                <button
                  onclick={() => submitReservierung(titelId)}
                  disabled={form.loading}
                  class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-2"
                >
                  {#if form.loading}
                    <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
                  {/if}
                  Anfrage senden
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {:else if searchQuery.trim().length >= 2 && !isSearching}
    <div class="text-center py-16 text-slate-400">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 mx-auto mb-3 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <p class="text-sm font-medium">Keine Bücher gefunden für <em>„{searchQuery}"</em></p>
    </div>
  {:else if searchQuery.trim().length === 0}
    <div class="text-center py-16 text-slate-400">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 mx-auto mb-3 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
      </svg>
      <p class="text-sm font-medium">Gib einen Suchbegriff ein, um Bücher zu finden.</p>
    </div>
  {/if}
</div>
