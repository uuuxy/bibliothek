<script>
  let { profile, role, rechnungPdfLoading, onDownloadRechnung, onEdit } = $props();

  import { apiClient } from "./apiFetch.js";
  let isUpdatingBlock = $state(false);

  function formatDate(dateString) {
    if (!dateString) return "Keine Angabe";
    try {
      const d = new Date(dateString);
      return d.toLocaleDateString("de-DE", { day: '2-digit', month: '2-digit', year: 'numeric' });
    } catch {
      return dateString;
    }
  }

  async function toggleManualBlock() {
    if (!profile) return;
    isUpdatingBlock = true;
    try {
      const newVal = !profile.is_manually_blocked;
      const res = await apiClient.post(`/api/schueler/${profile.id}/update`, {
        is_manually_blocked: newVal,
        block_reason: newVal ? "Manuell gesperrt" : ""
      });
      if (res.ok) {
        profile.is_manually_blocked = newVal;
      }
    } catch (e) {
      console.error("Failed to toggle block", e);
    } finally {
      isUpdatingBlock = false;
    }
  }
</script>

<div class="w-full pt-2 animate-fade-in space-y-8">
  <div class="flex justify-between items-center border-b border-slate-100 pb-4">
    <h3 class="text-xl font-bold text-slate-800 flex items-center gap-2">
      <svg class="w-6 h-6 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2"/></svg>
      Stammdaten & Adresse
    </h3>
    <div class="flex items-center gap-2">
      {#if role === 'admin' || role === 'mitarbeiter'}
        <button onclick={onDownloadRechnung} disabled={rechnungPdfLoading} class="px-5 py-2.5 bg-slate-100 text-slate-700 hover:bg-slate-200 disabled:opacity-50 rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
          {#if rechnungPdfLoading}
            <div class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"></div>
          {:else}
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
          {/if}
          Ersatzforderung drucken
        </button>
      {/if}
      {#if role === 'admin'}
        <button onclick={onEdit} class="px-5 py-2.5 bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
          Bearbeiten
        </button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
    <div class="space-y-6">
      <div>
        <p class="text-xs font-bold text-slate-600 uppercase tracking-wider mb-1">Geburtsdatum</p>
        <p class="text-slate-800 font-semibold">{formatDate(profile.geburtsdatum)}</p>
      </div>
      <div>
        <p class="text-xs font-bold text-slate-600 uppercase tracking-wider mb-1">LUSD ID</p>
        <p class="text-slate-800 font-semibold">{profile.lusd_id || 'Keine Angabe'}</p>
      </div>
      <div>
        <p class="text-xs font-bold text-slate-600 uppercase tracking-wider mb-1">System-ID</p>
        <p class="text-slate-600 font-mono text-xs">{profile.id}</p>
      </div>
    </div>

    <div class="space-y-6">
      <div>
        <p class="text-xs font-bold text-slate-600 uppercase tracking-wider mb-1">Postanschrift</p>
        {#if profile.strasse}
          <p class="text-slate-800 font-semibold">{profile.strasse} {profile.hausnummer}</p>
          <p class="text-slate-800 font-semibold">{profile.plz} {profile.ort}</p>
        {:else}
          <p class="text-slate-600 italic text-sm">Keine Adresse hinterlegt</p>
        {/if}
      </div>
      <div>
        <p class="text-xs font-bold text-slate-600 uppercase tracking-wider mb-1">Eltern E-Mail</p>
        {#if profile.eltern_email}
          <a href="mailto:{profile.eltern_email}" class="text-blue-600 hover:underline font-semibold">{profile.eltern_email}</a>
        {:else}
          <p class="text-slate-600 italic text-sm">Keine E-Mail hinterlegt</p>
        {/if}
      </div>
      {#if role === 'admin' || role === 'mitarbeiter'}
        <div class="p-4 bg-slate-50 border border-slate-200 rounded-xl mt-4">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm font-bold text-slate-900">Manuelle Ausleihsperre</p>
              <p class="text-xs text-slate-500 mt-1">Schüler blockieren (z.B. wegen ausstehender Zahlungen)</p>
            </div>
            <button
              type="button"
              onclick={toggleManualBlock}
              disabled={isUpdatingBlock}
              aria-label="Manuelle Ausleihsperre umschalten"
              class="relative inline-flex h-7 w-12 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 {profile.is_manually_blocked ? 'bg-red-500' : 'bg-slate-300'} {isUpdatingBlock ? 'opacity-50' : ''}"
              role="switch"
              aria-checked={profile.is_manually_blocked}
            >
              <span class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-white shadow-sm transition duration-200 ease-in-out {profile.is_manually_blocked ? 'translate-x-5' : 'translate-x-0'}"></span>
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
