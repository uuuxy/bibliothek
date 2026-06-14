<script>
  import { apiFetch, apiClient } from "./apiFetch.js";

  let { open = false, profile, onclose, onsuccess } = $props();

  let deleteError = $state("");
  let isDeleting = $state(false);

  $effect(() => {
    if (open) {
      deleteError = "";
      isDeleting = false;
    }
  });

  async function deleteStudent() {
    if (profile?.entliehene_buecher && profile.entliehene_buecher.length > 0) {
      deleteError = "Löschen nicht möglich: Schüler hat noch entliehene Bücher";
      return;
    }
    deleteError = "";
    isDeleting = true;
    try {
      const res = await apiFetch(`/api/schueler/${profile.id}`, { method: "DELETE" });
      if (res.ok) {
        onsuccess?.();
      } else {
        const errText = await res.text();
        try {
          const errObj = JSON.parse(errText);
          deleteError = errObj.error || "Fehler beim Löschen des Schülers.";
        } catch {
          deleteError = errText || "Fehler beim Löschen des Schülers.";
        }
      }
    } catch (err) {
      deleteError = "Netzwerkfehler beim Löschen des Schülers.";
      console.error(err);
    } finally {
      isDeleting = false;
    }
  }

  function handleClose() {
    onclose?.();
  }
</script>

{#if open && profile}
  <div class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4 animate-fade-in" role="dialog" aria-modal="true">
    <div class="w-full max-w-md rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800 text-left">
      <h3 class="text-lg font-bold text-rose-600 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
        <span>Schüler löschen</span>
      </h3>
      {#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
        <div class="mt-4 p-4 bg-rose-50 border border-rose-100 rounded-2xl text-sm font-semibold text-rose-700">
          Löschen nicht möglich: Schüler hat noch entliehene Bücher
        </div>
        <div class="mt-6 flex justify-end">
          <button onclick={handleClose} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 transition-colors cursor-pointer">Schließen</button>
        </div>
      {:else}
        <p class="mt-4 text-sm text-slate-600 leading-relaxed font-sans">
          Sind Sie sicher, dass Sie den Schüler <strong>{profile.vorname} {profile.nachname}</strong> unwiderruflich aus der Datenbank löschen möchten? Alle historischen Ausleihen werden anonymisiert.
        </p>
        {#if deleteError}
          <div class="mt-4 p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600">
            {deleteError}
          </div>
        {/if}
        <div class="mt-6 flex justify-end gap-3">
          <button onclick={handleClose} disabled={isDeleting} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer">Abbrechen</button>
          <button onclick={deleteStudent} disabled={isDeleting} class="rounded-xl bg-red-650 px-4 py-2 text-sm font-bold text-white hover:bg-red-750 disabled:opacity-60 transition-colors cursor-pointer">
            {#if isDeleting}Löschen...{:else}Unwiderruflich löschen{/if}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}
