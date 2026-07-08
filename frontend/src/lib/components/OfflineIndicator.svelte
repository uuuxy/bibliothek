<script>
  import { offlineSync } from "../stores/offlineSync.svelte.js";
  import { CloudOff, Download, Upload } from "@lucide/svelte";
  import { toastStore } from "../stores/toastStore.svelte.js";

  // isOffline and global events are now handled centrally in offlineSync.svelte.js

  async function handleBackup() {
    await offlineSync.exportQueueAsJSON();
  }

  /** @type {HTMLInputElement | null} */
  let fileInput = $state(null);
  
  async function handleFileSelect(e) {
    const file = /** @type {HTMLInputElement} */ (e.target).files?.[0];
    if (!file) return;
    try {
      const count = await offlineSync.importQueueFromJSON(file);
      toastStore.addToast(`${count} Offline-Scans erfolgreich nachgetragen!`, "success");
    } catch (err) {
      toastStore.addToast(err instanceof Error ? err.message : String(err), "error");
    }
    /** @type {HTMLInputElement} */ (e.target).value = ""; // reset
  }
</script>

{#if offlineSync.pendingCount > 0 || offlineSync.isOffline}
  <div class="fixed top-0 left-0 right-0 z-9999 bg-rose-600 text-white shadow-2xl border-b-4 border-rose-800 animate-slide-down">
    <div class="max-w-7xl mx-auto px-6 py-4 flex flex-col md:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="bg-rose-500/50 p-3 rounded-2xl shrink-0">
          <CloudOff size={32} strokeWidth={2.5} class="text-white" />
        </div>
        <div>
          <h1 class="text-xl md:text-2xl font-black tracking-tight uppercase drop-shadow-md">
            Offline-Modus! Rechner nicht ausschalten - Datenverlust droht!
          </h1>
          <p class="text-rose-100 font-semibold mt-1">
            {offlineSync.pendingCount} ausstehende Aktion{offlineSync.pendingCount === 1 ? '' : 'en'} in der lokalen Warteschlange. 
            {#if offlineSync.isSyncing}
              <span class="animate-pulse ml-2">(Wird gerade synchronisiert...)</span>
            {:else if offlineSync.isOffline}
              (Wartet auf Internetverbindung...)
            {/if}
          </p>
        </div>
      </div>
      
      <div class="flex items-center gap-3 shrink-0">
        {#if offlineSync.pendingCount > 0}
          <button 
            onclick={handleBackup} 
            class="px-5 py-2.5 bg-white text-rose-700 hover:bg-rose-50 active:bg-rose-100 font-bold rounded-xl shadow-lg border border-rose-200 transition-all cursor-pointer flex items-center gap-2"
          >
            <Download size={18} strokeWidth={3} />
            Notfall-Backup auf USB-Stick speichern
          </button>
        {/if}
        
        <!-- Verstecktes File Input für Import -->
        <input 
          type="file" 
          accept=".json" 
          bind:this={fileInput} 
          onchange={handleFileSelect} 
          class="hidden" 
        />
        
        <button 
          onclick={() => fileInput?.click()} 
          class="px-4 py-2.5 bg-rose-700 hover:bg-rose-800 text-rose-50 font-bold rounded-xl shadow-inner border border-rose-800 transition-all cursor-pointer flex items-center gap-2"
          title="Backup einspielen (falls du an einem anderen PC den Stand nachträgst)"
        >
          <Upload size={18} strokeWidth={2.5} />
          Offline-Backup einspielen
        </button>
      </div>
    </div>
  </div>
{/if}
