<script>
  import { apiGet, apiPost, apiDelete } from "./apiFetch.js";
  import { onMount } from 'svelte';
  import { toastStore } from "./stores/toastStore.svelte.js";

  /** @type {{klasse: string, lehrer_email: string}[]} */
  let mappingRows = $state([]);
  let mappingLoading = $state(false);
  let newMappingKlasse = $state('');
  let newMappingEmail = $state('');
  let mappingSaving = $state(false);

  async function fetchMapping() {
    mappingLoading = true;
    try {
      mappingRows = await apiGet('/api/klassen-mapping') || [];
    } catch { /* ignore */ } finally {
      mappingLoading = false;
    }
  }

  onMount(async () => {
    await fetchMapping();
  });

  async function upsertMapping() {
    if (!newMappingKlasse.trim() || !newMappingEmail.trim()) return;
    mappingSaving = true;
    try {
      await apiPost('/api/klassen-mapping', { klasse: newMappingKlasse.trim(), lehrer_email: newMappingEmail.trim() });
      newMappingKlasse = '';
      newMappingEmail = '';
      await fetchMapping();
      toastStore.addToast('Mapping gespeichert.', 'success');
    } catch {
      // Toast already shown by apiPost
    } finally {
      mappingSaving = false;
    }
  }

  /** @param {string} klasse */
  async function deleteMapping(klasse) {
    try {
      await apiDelete(`/api/klassen-mapping/${encodeURIComponent(klasse)}`);
      await fetchMapping();
      toastStore.addToast(`Mapping für ${klasse} gelöscht.`, 'success');
    } catch {
      // Toast already shown by apiDelete
    }
  }
</script>

<div class="space-y-6">
  <!-- Klassenlehrer-Mapping Card -->
  <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
    <div class="mb-6">
      <h3 class="text-xl font-bold text-slate-900">E-Mail Routing für Mahnungen</h3>
      <p class="text-sm text-slate-500 mt-1">Ordnet jeder Klasse eine E-Mail-Adresse zu. Wird im Mahnwesen als Empfänger für Benachrichtigungen vorausgefüllt.</p>
    </div>

    {#if mappingLoading}
      <div class="py-8 flex justify-center"><div class="w-8 h-8 border-4 border-slate-400 border-t-transparent rounded-full animate-spin"></div></div>
    {:else if mappingRows.length === 0}
      <div class="p-6 bg-slate-50 rounded-2xl border border-slate-100 text-center">
        <p class="text-sm text-slate-500 font-medium">Noch keine Mappings vorhanden.</p>
      </div>
    {:else}
      <div class="rounded-2xl border border-slate-200 overflow-hidden mb-6">
        <table class="w-full text-sm">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200 text-xs font-bold text-slate-500 uppercase tracking-wider">
              <th class="text-left px-5 py-3">Klasse</th>
              <th class="text-left px-5 py-3">Lehrer-E-Mail</th>
              <th class="px-5 py-3 text-right">Aktion</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            {#each mappingRows as row}
              <tr class="hover:bg-slate-50 transition-colors">
                <td class="px-5 py-3 font-semibold text-slate-800">{row.klasse}</td>
                <td class="px-5 py-3 text-slate-600">{row.lehrer_email}</td>
                <td class="px-5 py-3 text-right">
                  <button onclick={() => deleteMapping(row.klasse)} class="p-2 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors cursor-pointer" title="Mapping löschen">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <!-- Add new mapping -->
    <div class="p-5 bg-slate-50 rounded-2xl border border-slate-200 flex flex-col md:flex-row items-end gap-4">
      <div class="w-full md:w-32">
        <label for="newMappingKlasseInput" class="text-[10px] font-bold text-slate-500 uppercase tracking-wider block mb-1.5">Klasse</label>
        <input id="newMappingKlasseInput" type="text" bind:value={newMappingKlasse} placeholder="z.B. 7a" class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all" />
      </div>
      <div class="flex-1 w-full">
        <label for="newMappingEmailInput" class="text-[10px] font-bold text-slate-500 uppercase tracking-wider block mb-1.5">E-Mail</label>
        <input id="newMappingEmailInput" type="email" bind:value={newMappingEmail} placeholder="lehrkraft@schule.de" class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all" />
      </div>
      <button
        onclick={upsertMapping}
        disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
        class="w-full md:w-auto px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap shadow-sm">
        {mappingSaving ? 'Lädt…' : 'Hinzufügen'}
      </button>
    </div>
  </div>
</div>
