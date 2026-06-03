<script>
  let { isOffline, offlineQueueCount, flushOfflineQueue } = $props();
</script>

{#if isOffline || offlineQueueCount > 0}
  <div class="fixed top-4 right-4 z-200 flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold shadow-lg border {isOffline ? 'bg-rose-50 border-rose-200 text-rose-700' : 'bg-amber-50 border-amber-200 text-amber-700'} animate-slide-down">
    {#if isOffline}
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636a9 9 0 010 12.728M5.636 5.636a9 9 0 000 12.728M12 12h.01"/>
      </svg>
      <span>Offline{offlineQueueCount > 0 ? ` · ${offlineQueueCount} Scan(s) ausstehend` : ""}</span>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      <span>{offlineQueueCount} Offline-Scan(s) ausstehend</span>
      <button onclick={flushOfflineQueue} class="ml-1 underline hover:no-underline cursor-pointer">Jetzt senden</button>
    {/if}
  </div>
{/if}
