<script>
  import { offlineSync } from "../stores/offlineSync.svelte.js";
  import { CloudOff } from "lucide-svelte";
</script>

{#if offlineSync.pendingCount > 0}
  <div class="fixed bottom-6 left-6 z-50 flex items-center gap-3 bg-rose-600 text-white px-5 py-3 rounded-2xl shadow-xl border border-rose-500 animate-slide-up">
    <div class="relative flex h-3 w-3">
      {#if offlineSync.isSyncing}
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-white opacity-75"></span>
      {/if}
      <span class="relative inline-flex rounded-full h-3 w-3 bg-white"></span>
    </div>
    
    <div class="flex flex-col">
      <span class="text-sm font-bold tracking-wide">OFFLINE QUEUE</span>
      <span class="text-xs font-semibold text-rose-100">
        {offlineSync.pendingCount} ausstehende Aktion{offlineSync.pendingCount === 1 ? '' : 'en'} 
        {#if offlineSync.isSyncing} (Synchronisiere...){/if}
      </span>
    </div>
    
    <div class="bg-rose-500/50 p-2 rounded-xl ml-2">
      <CloudOff size={20} strokeWidth={2.5} class="text-white" />
    </div>
  </div>
{/if}
