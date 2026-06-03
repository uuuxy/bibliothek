<script>
  import { apiFetch } from "./apiFetch.js";
  let data = $state.raw(/** @type {{ klassen: any[] } | null} */ (null));
  let loading = $state(true);
  let error = $state(/** @type {string|null} */ (null));

  // Modal state
  let modalOpen = $state(false);
  let modalKlasse = $state("");
  let modalEmail = $state("");
  let modalSending = $state(false);
  let modalMsg = $state(/** @type {{ type: 'success'|'error', text: string }|null} */ (null));

  // PDF download loading
  let pdfLoading = $state(false);

  // Expanded classes
  let expandedKlassen = $state(/** @type {Set<string>} */ (new Set()));

  async function fetchData() {
    loading = true;
    error = null;
    try {
      const res = await apiFetch("/api/mahnwesen");
      if (!res.ok) throw new Error(await res.text() || "Fehler beim Laden");
      const json = await res.json();
      data = json;
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  import { onMount } from "svelte";
  onMount(fetchData);

  /** @param {string} klasse */
  function toggleKlasse(klasse) {
    const s = new Set(expandedKlassen);
    if (s.has(klasse)) s.delete(klasse);
    else s.add(klasse);
    expandedKlassen = s;
  }

  async function downloadPDF() {
    pdfLoading = true;
    try {
      const res = await apiFetch("/api/mahnwesen/pdf");
      if (!res.ok) throw new Error("PDF-Erzeugung fehlgeschlagen");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `mahnliste_${new Date().toISOString().slice(0,10)}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      alert("Fehler: " + String(e));
    } finally {
      pdfLoading = false;
    }
  }

  /**
   * @param {string} klasse
   * @param {string|null} [email]
   */
  function openModal(klasse, email) {
    modalKlasse = klasse;
    modalEmail = email ?? "";
    modalMsg = null;
    modalOpen = true;
  }

  function closeModal() {
    modalOpen = false;
    modalKlasse = "";
    modalEmail = "";
    modalMsg = null;
  }

  async function sendMahnliste() {
    if (!modalEmail.trim()) { modalMsg = { type: 'error', text: 'E-Mail-Adresse angeben.' }; return; }
    modalSending = true;
    modalMsg = null;
    try {
      const res = await apiFetch("/api/mahnwesen/senden", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ klasse: modalKlasse, email: modalEmail }),
      });
      const json = await res.json();
      if (res.ok) {
        modalMsg = { type: 'success', text: json.message ?? "Gesendet." };
      } else {
        modalMsg = { type: 'error', text: json.error ?? json.message ?? "Fehler." };
      }
    } catch (e) {
      modalMsg = { type: 'error', text: String(e) };
    } finally {
      modalSending = false;
    }
  }

  const klassen = $derived(data?.klassen ?? []);
  const totalOverdue = $derived(
    klassen.reduce((/** @type {number} */ sum, /** @type {any} */ k) =>
      sum + k.schueler.reduce((/** @type {number} */ s2, /** @type {any} */ sch) => s2 + sch.medien.length, 0), 0)
  );
</script>

<div class="max-w-5xl mx-auto space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold text-slate-800">Mahnwesen</h1>
      <p class="text-sm text-slate-500 mt-0.5">Überfällige Ausleihen nach Klassen sortiert.</p>
    </div>
    <div class="flex items-center gap-2">
      <button
        onclick={fetchData}
        class="px-3 py-2 rounded-xl border border-slate-200 bg-white text-slate-600 hover:bg-slate-50 text-xs font-semibold transition-all flex items-center gap-1.5"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Aktualisieren
      </button>
      <button
        onclick={downloadPDF}
        disabled={pdfLoading}
        class="px-3 py-2 rounded-xl bg-slate-700 hover:bg-slate-800 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-1.5"
      >
        {#if pdfLoading}
          <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        {/if}
        Mahnliste (gesamt) als PDF
      </button>
    </div>
  </div>

  <!-- Stats bar -->
  {#if data && !loading}
    <div class="grid grid-cols-3 gap-4">
      <div class="bg-white rounded-2xl border border-slate-200 p-4 text-center">
        <p class="text-2xl font-bold text-rose-600">{totalOverdue}</p>
        <p class="text-xs text-slate-500 mt-0.5">Überfällige Medien</p>
      </div>
      <div class="bg-white rounded-2xl border border-slate-200 p-4 text-center">
        <p class="text-2xl font-bold text-slate-800">{klassen.length}</p>
        <p class="text-xs text-slate-500 mt-0.5">Betroffene Klassen</p>
      </div>
      <div class="bg-white rounded-2xl border border-slate-200 p-4 text-center">
        <p class="text-2xl font-bold text-slate-800">{klassen.reduce((s, k) => s + k.schueler.length, 0)}</p>
        <p class="text-xs text-slate-500 mt-0.5">Betroffene Schüler/innen</p>
      </div>
    </div>
  {/if}

  <!-- Loading / error -->
  {#if loading}
    <div class="flex justify-center py-20">
      <div class="w-8 h-8 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"></div>
    </div>
  {:else if error}
    <div class="bg-rose-50 border border-rose-200 rounded-2xl p-6 text-center text-rose-600 text-sm font-medium">{error}</div>
  {:else if !data || klassen.length === 0}
    <div class="bg-emerald-50 border border-emerald-200 rounded-2xl p-10 text-center">
      <p class="text-emerald-700 font-semibold">Keine überfälligen Ausleihen vorhanden. 🎉</p>
    </div>
  {:else}
    <!-- Class cards -->
    <div class="space-y-4">
      {#each klassen as klasse}
        {@const totalMediaInClass = klasse.schueler.reduce((/** @type {number} */ s, /** @type {any} */ sch) => s + sch.medien.length, 0)}
        <div class="bg-white rounded-2xl border border-slate-200 shadow-xs overflow-hidden">
          <!-- Class header -->
          <div
            role="button"
            tabindex="0"
            onclick={() => toggleKlasse(klasse.klasse)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleKlasse(klasse.klasse); } }}
            class="w-full flex items-center justify-between px-5 py-4 hover:bg-slate-50 transition-colors text-left cursor-pointer focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
          >
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-xl bg-rose-50 border border-rose-100 flex items-center justify-center">
                <span class="text-rose-600 font-bold text-sm">{klasse.klasse}</span>
              </div>
              <div>
                <p class="font-semibold text-slate-800">{klasse.klasse}</p>
                <p class="text-xs text-slate-400">{klasse.schueler.length} Schüler/innen · {totalMediaInClass} Medien überfällig</p>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <button
                onclick={(e) => { e.stopPropagation(); openModal(klasse.klasse, klasse.lehrer_email); }}
                class="px-3 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold transition-all"
              >
                Per E-Mail senden
              </button>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-slate-400 transition-transform {expandedKlassen.has(klasse.klasse) ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>

          <!-- Students -->
          {#if expandedKlassen.has(klasse.klasse)}
            <div class="border-t border-slate-100 divide-y divide-slate-50">
              {#each klasse.schueler as schueler}
                <div class="px-5 py-4">
                  <div class="flex items-center justify-between mb-3">
                    <div>
                      <p class="font-semibold text-sm text-slate-800">{schueler.name}</p>
                      <p class="text-xs text-slate-400">{schueler.medien.length} {schueler.medien.length === 1 ? 'Medium' : 'Medien'} überfällig</p>
                    </div>
                  </div>
                  <!-- Media list -->
                  <div class="space-y-2 ml-2">
                    {#each schueler.medien as medium}
                      <div class="flex items-center gap-3 p-2.5 rounded-xl bg-slate-50 border border-slate-100">
                        {#if medium.cover_url}
                          <img src={medium.cover_url} alt="Cover" class="w-9 h-11 object-cover rounded-lg border border-slate-200 shrink-0" loading="lazy" />
                        {:else}
                          <div class="w-9 h-11 rounded-lg bg-slate-200 border border-slate-300 shrink-0"></div>
                        {/if}
                        <div class="flex-1 min-w-0">
                          <p class="text-xs font-semibold text-slate-800 truncate">{medium.titel}</p>
                          <p class="text-[10px] text-slate-500">{medium.autor}</p>
                        </div>
                        <div class="text-right shrink-0">
                          <p class="text-[10px] text-slate-500">Fällig {medium.faellig_am}</p>
                          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold {medium.tage_ueberfaellig > 14 ? 'bg-rose-100 text-rose-700' : 'bg-amber-50 text-amber-700'}">
                            {medium.tage_ueberfaellig} Tage
                          </span>
                        </div>
                      </div>
                    {/each}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- E-Mail Modal -->
{#if modalOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="absolute inset-0 bg-black/20 backdrop-blur-sm" onclick={closeModal} aria-hidden="true"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-md p-6 space-y-5">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-bold text-slate-800">Mahnliste per E-Mail senden</h2>
        <button onclick={closeModal} aria-label="Modal schließen" class="p-1.5 rounded-lg text-slate-400 hover:bg-slate-100 transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="space-y-4">
        <div>
          <span class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Klasse</span>
          <p class="text-sm font-semibold text-slate-800">{modalKlasse}</p>
        </div>
        <div>
          <label for="modal-email" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">E-Mail-Adresse des Klassenlehrers</label>
          <input
            id="modal-email"
            type="email"
            bind:value={modalEmail}
            placeholder="lehrer@schule.de"
            class="w-full px-3 py-2.5 rounded-xl border border-slate-200 bg-slate-50 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
          />
          {#if !modalEmail.trim()}
            <p class="text-[10px] text-slate-400 mt-1">Die Adresse wird aus dem Klassenlehrer-Mapping vorausgefüllt, kann aber geändert werden.</p>
          {/if}
        </div>
      </div>

      {#if modalMsg}
        <div class="rounded-xl px-4 py-3 text-xs font-semibold {modalMsg.type === 'success' ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-rose-50 text-rose-600 border border-rose-200'}">
          {modalMsg.text}
        </div>
      {/if}

      <div class="flex justify-end gap-2">
        <button onclick={closeModal} class="px-4 py-2 rounded-xl border border-slate-200 text-slate-600 hover:bg-slate-50 text-xs font-semibold transition-all">Abbrechen</button>
        <button
          onclick={sendMahnliste}
          disabled={modalSending || modalMsg?.type === 'success'}
          class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-2"
        >
          {#if modalSending}
            <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          {/if}
          Senden
        </button>
      </div>
    </div>
  </div>
{/if}
