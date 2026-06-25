<script>
  import { omniboxStore } from "../stores/omnibox.svelte.js";
  import { apiClient } from "../apiFetch.js";

  /** @type {{ onReload: () => void }} */
  let { onReload } = $props();
</script>

{#if omniboxStore.blockAlert}
  <div class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4">
    <div class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500">
      <div class="text-6xl mb-4">⛔️</div>
      <h2 class="text-2xl font-extrabold text-rose-700 mb-2">Ausleihe blockiert</h2>
      <p class="text-slate-700 font-medium mb-6">{omniboxStore.blockAlert.message}</p>

      <div class="space-y-3">
        <button onclick={() => {
          const q = omniboxStore.blockAlert?.query;
          if (!q) return;
          omniboxStore.blockAlert = null;
          omniboxStore.queryVal = q;
          omniboxStore.submitAction(null, onReload, true);
        }}
          class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
          Einmalig ignorieren (Override)
        </button>

        {#if omniboxStore.activeStudent?.is_manually_blocked}
          <button onclick={async () => {
            try {
              const res = await apiClient.post(`/api/schueler/${omniboxStore.activeStudent.id}/update`, {
                is_manually_blocked: false,
                block_reason: ""
              });
              if (res.ok) {
                const q = omniboxStore.blockAlert?.query;
                omniboxStore.blockAlert = null;
                if (q) omniboxStore.queryVal = q;
                omniboxStore.activeStudent.is_manually_blocked = false;
                omniboxStore.submitAction(null, onReload);
              }
            } catch(e) {
              console.error(e);
            }
          }}
            class="px-8 py-3 bg-white border-2 border-slate-200 hover:bg-slate-50 text-slate-700 font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
            Sperre dauerhaft aufheben
          </button>
        {/if}

        <button onclick={() => { omniboxStore.blockAlert = null; }}
          class="px-8 py-3 bg-transparent text-slate-500 hover:text-slate-700 font-bold rounded-xl text-sm transition-colors cursor-pointer w-full mt-2">
          Abbrechen
        </button>
      </div>
    </div>
  </div>
{/if}
