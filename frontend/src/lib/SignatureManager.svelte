<script>
  import { apiClient, apiFetch } from './apiFetch.js';

  /** @type {{ id: number, name: string, description: string }[]} */
  let signatures = $state([]);
  let loading = $state(true);
  /** @type {{ msg: string, type: 'success' | 'error' } | null} */
  let toast = $state(null);

  // Modal state
  let modalOpen = $state(false);
  let editingId = $state(/** @type {number|null} */ (null));
  let formName = $state('');
  let formDescription = $state('');
  let formSaving = $state(false);
  let deleteConfirmId = $state(/** @type {number|null} */ (null));

  /**
   * @param {string} msg
   * @param {'success'|'error'} [type]
   */
  function showToast(msg, type = 'success') {
    toast = { msg, type };
    setTimeout(() => { toast = null; }, 4000);
  }

  async function loadSignatures() {
    loading = true;
    try {
      const res = await apiClient.get('/api/signatures');
      if (res.ok) {
        signatures = await res.json();
      }
    } catch { /* ignore */ } finally {
      loading = false;
    }
  }

  $effect(() => {
    loadSignatures();
  });

  function openCreateModal() {
    editingId = null;
    formName = '';
    formDescription = '';
    modalOpen = true;
  }

  /** @param {{ id: number, name: string, description: string }} sig */
  function openEditModal(sig) {
    editingId = sig.id;
    formName = sig.name;
    formDescription = sig.description;
    modalOpen = true;
  }

  function closeModal() {
    modalOpen = false;
  }

  async function saveSignature() {
    if (!formName.trim()) return;
    formSaving = true;
    try {
      const payload = { name: formName.trim(), description: formDescription.trim() };
      const res = editingId
        ? await apiClient.put(`/api/signatures/${editingId}`, payload)
        : await apiClient.post('/api/signatures', payload);

      if (res.ok) {
        closeModal();
        await loadSignatures();
        showToast(editingId ? 'Signatur aktualisiert.' : 'Signatur angelegt.');
      } else if (res.status === 409) {
        showToast('Eine Signatur mit diesem Namen existiert bereits.', 'error');
      } else {
        showToast((await res.text()) || 'Fehler beim Speichern.', 'error');
      }
    } catch {
      showToast('Netzwerkfehler.', 'error');
    } finally {
      formSaving = false;
    }
  }

  /** @param {number} id */
  async function deleteSignature(id) {
    deleteConfirmId = null;
    try {
      const res = await apiFetch(`/api/signatures/${id}`, { method: 'DELETE' });
      if (res.ok || res.status === 204) {
        await loadSignatures();
        showToast('Signatur gelöscht.');
      } else if (res.status === 409) {
        showToast('Signatur kann nicht gelöscht werden, da ihr noch Bücher zugeordnet sind.', 'error');
      } else {
        showToast('Fehler beim Löschen.', 'error');
      }
    } catch {
      showToast('Netzwerkfehler.', 'error');
    }
  }
</script>

<!-- Toast -->
{#if toast}
  <div class="fixed top-6 right-6 z-200 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in flex items-center gap-2
    {toast.type === 'error' ? 'bg-rose-600 text-white' : 'bg-emerald-600 text-white'}">
    {#if toast.type === 'error'}
      <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
    {:else}
      <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
    {/if}
    {toast.msg}
  </div>
{/if}

<!-- Delete Confirm Dialog -->
{#if deleteConfirmId !== null}
  <div class="fixed inset-0 z-150 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm animate-fade-in">
    <div class="bg-white rounded-3xl shadow-2xl p-6 w-full max-w-sm mx-4 space-y-4">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-2xl bg-rose-100 flex items-center justify-center shrink-0">
          <svg class="w-5 h-5 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
        </div>
        <div>
          <h4 class="text-base font-bold text-slate-900">Signatur löschen?</h4>
          <p class="text-xs text-slate-500 mt-0.5">Diese Aktion kann nicht rückgängig gemacht werden.</p>
        </div>
      </div>
      <div class="flex gap-2 justify-end pt-1">
        <button onclick={() => (deleteConfirmId = null)} class="px-4 py-2 text-sm font-semibold text-slate-600 bg-slate-100 hover:bg-slate-200 rounded-xl transition-colors cursor-pointer">
          Abbrechen
        </button>
        <button onclick={() => deleteSignature(/** @type {number} */ (deleteConfirmId))} class="px-4 py-2 text-sm font-semibold text-white bg-rose-600 hover:bg-rose-700 rounded-xl transition-colors cursor-pointer">
          Löschen
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Create/Edit Modal -->
{#if modalOpen}
  <div class="fixed inset-0 z-150 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm animate-fade-in">
    <div class="bg-white rounded-3xl shadow-2xl p-6 w-full max-w-md mx-4 space-y-5">
      <div class="flex items-center justify-between">
        <h4 class="text-base font-bold text-slate-900">
          {editingId ? 'Signatur bearbeiten' : 'Neue Signatur anlegen'}
        </h4>
        <button onclick={closeModal} aria-label="Schließen" class="w-8 h-8 flex items-center justify-center rounded-xl text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
        </button>
      </div>

      <div class="space-y-3">
        <div>
          <label for="sig-name-input" class="text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-1">Name *</label>
          <input
            id="sig-name-input"
            type="text"
            bind:value={formName}
            placeholder="z. B. Romane, Sachbücher, Regal A1"
            class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2.5 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800"
          />
        </div>
        <div>
          <label for="sig-desc-input" class="text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-1">Beschreibung (optional)</label>
          <textarea
            id="sig-desc-input"
            bind:value={formDescription}
            rows="3"
            placeholder="Kurze Beschreibung dieser Kategorie oder dieses Standorts…"
            class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2.5 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800 resize-none"
          ></textarea>
        </div>
      </div>

      <div class="flex gap-2 justify-end pt-1">
        <button onclick={closeModal} class="px-4 py-2.5 text-sm font-semibold text-slate-600 bg-slate-100 hover:bg-slate-200 rounded-xl transition-colors cursor-pointer">
          Abbrechen
        </button>
        <button
          onclick={saveSignature}
          disabled={formSaving || !formName.trim()}
          class="px-5 py-2.5 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2">
          {#if formSaving}
            <div class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
          {/if}
          {editingId ? 'Speichern' : 'Anlegen'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Main Card -->
<div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-5">
  <!-- Header -->
  <div class="flex items-start justify-between gap-4 border-b border-slate-100 pb-4">
    <div>
      <h3 class="text-base font-bold text-slate-900 flex items-center gap-2">
        <svg class="w-5 h-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
        </svg>
        Signaturen-Verwaltung
      </h3>
      <p class="text-xs text-slate-500 mt-1 max-w-lg">
        Definiert Kategorien und Standorte, die Medientiteln zugewiesen werden können.
      </p>
    </div>
    <button
      onclick={openCreateModal}
      class="shrink-0 flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-bold rounded-xl transition-colors cursor-pointer shadow-sm">
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
      Neue Signatur
    </button>
  </div>

  <!-- Content -->
  {#if loading}
    <div class="py-8 flex justify-center">
      <div class="w-6 h-6 border-2 border-indigo-400 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if signatures.length === 0}
    <div class="py-10 flex flex-col items-center gap-3 text-slate-400">
      <svg class="w-10 h-10 opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
      </svg>
      <p class="text-sm font-medium">Noch keine Signaturen vorhanden.</p>
      <button onclick={openCreateModal} class="text-xs font-bold text-indigo-600 hover:underline cursor-pointer">Erste Signatur anlegen →</button>
    </div>
  {:else}
    <div class="rounded-2xl border border-slate-100 overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-500 uppercase tracking-wider">
            <th class="text-left px-4 py-3">Name</th>
            <th class="text-left px-4 py-3 hidden sm:table-cell">Beschreibung</th>
            <th class="px-4 py-3 w-24"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-50">
          {#each signatures as sig (sig.id)}
            <tr class="hover:bg-slate-50/80 transition-colors group">
              <td class="px-4 py-3">
                <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-indigo-50 text-indigo-700 font-semibold text-xs">
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/></svg>
                  {sig.name}
                </span>
              </td>
              <td class="px-4 py-3 text-slate-500 text-xs hidden sm:table-cell max-w-xs truncate">
                {sig.description || '—'}
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onclick={() => openEditModal(sig)}
                    title="Bearbeiten"
                    class="p-1.5 rounded-lg text-slate-400 hover:text-blue-600 hover:bg-blue-50 transition-colors cursor-pointer">
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                  </button>
                  <button
                    onclick={() => (deleteConfirmId = sig.id)}
                    title="Löschen"
                    class="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 transition-colors cursor-pointer">
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="text-[10px] text-slate-400 text-right">{signatures.length} Signatur{signatures.length !== 1 ? 'en' : ''} gesamt</p>
  {/if}
</div>
