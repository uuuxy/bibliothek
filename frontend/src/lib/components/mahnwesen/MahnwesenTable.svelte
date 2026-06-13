<script>
  import { mahnwesenStore } from "../../stores/mahnwesen.svelte.js";
</script>

{#if mahnwesenStore.loading}
  <div class="flex justify-center py-20">
    <div class="w-8 h-8 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"></div>
  </div>
{:else if mahnwesenStore.error}
  <div class="bg-rose-50 border border-rose-200 rounded-2xl p-6 text-center text-rose-600 text-sm font-medium">{mahnwesenStore.error}</div>
{:else if !mahnwesenStore.data || mahnwesenStore.klassen.length === 0}
  <div class="bg-emerald-50 border border-emerald-200 rounded-2xl p-10 text-center">
    <p class="text-emerald-700 font-semibold">Keine überfälligen Ausleihen vorhanden. 🎉</p>
  </div>
{:else}
  <!-- Class cards -->
  <div class="space-y-4">
    {#each mahnwesenStore.klassen as klasse}
      {@const totalMediaInClass = klasse.schueler.reduce((/** @type {number} */ s, /** @type {any} */ sch) => s + sch.medien.length, 0)}
      <div class="bg-white rounded-2xl border border-slate-200 shadow-xs overflow-hidden">
        <!-- Class header -->
        <div
          role="button"
          tabindex="0"
          onclick={() => mahnwesenStore.toggleKlasse(klasse.klasse)}
          onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); mahnwesenStore.toggleKlasse(klasse.klasse); } }}
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
          <div class="flex items-center gap-3 print:hidden">
            <button
              onclick={(e) => { e.stopPropagation(); mahnwesenStore.openModal(klasse.klasse, klasse.lehrer_email); }}
              class="px-3 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold transition-all"
            >
              Per E-Mail senden
            </button>
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-slate-400 transition-transform {mahnwesenStore.expandedKlassen.has(klasse.klasse) ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </div>
        </div>

        <!-- Students -->
        {#if mahnwesenStore.expandedKlassen.has(klasse.klasse)}
          <div class="border-t border-slate-100 divide-y divide-slate-50">
            {#each klasse.schueler as schueler}
              <div class="px-5 py-4">
                <div class="flex items-center justify-between mb-3">
                  <div>
                    <p class="font-semibold text-sm text-slate-800">{schueler.name}</p>
                    <p class="text-xs text-slate-400">{schueler.medien.length} {schueler.medien.length === 1 ? 'Medium' : 'Medien'} überfällig</p>
                  </div>
                  
                  {#if schueler.eltern_email}
                    <div class="flex items-center gap-2">
                      {#if mahnwesenStore.studentMessages[schueler.schueler_id]}
                        <span class="text-xs font-semibold px-2 py-1 rounded-md {mahnwesenStore.studentMessages[schueler.schueler_id]?.type === 'success' ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-rose-50 text-rose-600 border border-rose-200'}">
                          {mahnwesenStore.studentMessages[schueler.schueler_id]?.text}
                        </span>
                      {/if}
                      <button
                        onclick={() => mahnwesenStore.sendStudentMahnung(schueler.schueler_id)}
                        disabled={mahnwesenStore.sendingStudentId === schueler.schueler_id || mahnwesenStore.studentMessages[schueler.schueler_id]?.type === 'success'}
                        class="p-2 rounded-xl bg-blue-50 text-blue-600 hover:bg-blue-100 disabled:opacity-50 transition-colors"
                        title="Mahnung an Eltern senden"
                      >
                        {#if mahnwesenStore.sendingStudentId === schueler.schueler_id}
                          <div class="w-4 h-4 border-2 border-blue-600/30 border-t-blue-600 rounded-full animate-spin"></div>
                        {:else}
                          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                          </svg>
                        {/if}
                      </button>
                    </div>
                  {:else}
                    <span class="text-[10px] px-2 py-1 bg-slate-100 text-slate-400 rounded-md font-semibold">Keine Eltern-E-Mail</span>
                  {/if}
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
                        {#if mahnwesenStore.mahnMode === 'datum'}
                          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold {medium.tage_ueberfaellig > 14 ? 'bg-rose-100 text-rose-700' : 'bg-amber-50 text-amber-700'}">
                            {medium.tage_ueberfaellig} Tage
                          </span>
                        {:else}
                          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold {medium.tage_ueberfaellig > 0 ? 'bg-rose-100 text-rose-700' : 'bg-amber-50 text-amber-700'}">
                            +{medium.tage_ueberfaellig} Jahre
                          </span>
                        {/if}
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

<!-- E-Mail Modal -->
{#if mahnwesenStore.modalOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="absolute inset-0 bg-black/20 backdrop-blur-sm" onclick={mahnwesenStore.closeModal} aria-hidden="true"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-md p-6 space-y-5">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-bold text-slate-800">Mahnliste per E-Mail senden</h2>
        <button onclick={mahnwesenStore.closeModal} aria-label="Modal schließen" class="p-1.5 rounded-lg text-slate-400 hover:bg-slate-100 transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="space-y-4">
        <div>
          <span class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Klasse</span>
          <p class="text-sm font-semibold text-slate-800">{mahnwesenStore.modalKlasse}</p>
        </div>
        <div>
          <label for="modal-email" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">E-Mail-Adresse des Klassenlehrers</label>
          <input
            id="modal-email"
            type="email"
            bind:value={mahnwesenStore.modalEmail}
            placeholder="lehrer@schule.de"
            class="w-full px-3 py-2.5 rounded-xl border border-slate-200 bg-slate-50 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
          />
          {#if !mahnwesenStore.modalEmail.trim()}
            <p class="text-[10px] text-slate-400 mt-1">Die Adresse wird aus dem Klassenlehrer-Mapping vorausgefüllt, kann aber geändert werden.</p>
          {/if}
        </div>
      </div>

      {#if mahnwesenStore.modalMsg}
        <div class="rounded-xl px-4 py-3 text-xs font-semibold {mahnwesenStore.modalMsg.type === 'success' ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-rose-50 text-rose-600 border border-rose-200'}">
          {mahnwesenStore.modalMsg.text}
        </div>
      {/if}

      <div class="flex justify-end gap-2">
        <button onclick={mahnwesenStore.closeModal} class="px-4 py-2 rounded-xl border border-slate-200 text-slate-600 hover:bg-slate-50 text-xs font-semibold transition-all">Abbrechen</button>
        <button
          onclick={mahnwesenStore.sendMahnliste}
          disabled={mahnwesenStore.modalSending || mahnwesenStore.modalMsg?.type === 'success'}
          class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-2"
        >
          {#if mahnwesenStore.modalSending}
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
