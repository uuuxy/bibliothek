<script>
  import AuditLog from "./AuditLog.svelte";
  import AdminAuditLog from "./AdminAuditLog.svelte";
  import { authStore } from "./stores/authStore.svelte.js";
  
  let activeTab = $state("system");
  const isAdmin = $derived(authStore.currentUser?.rolle === 'admin');
</script>

<div class="w-full flex flex-col h-full bg-slate-50">
  <div class="px-8 pt-8 pb-4 border-b border-slate-200 bg-white shadow-sm shrink-0">
    <div class="max-w-6xl mx-auto flex items-end justify-between">
      <div>
        <h2 class="text-3xl font-bold text-slate-900 tracking-tight">System-Logs</h2>
        <p class="text-slate-500 mt-1">Überwache alle Systemereignisse und sicherheitsrelevanten Vorgänge.</p>
      </div>
    </div>
    
    <div class="max-w-6xl mx-auto mt-6 flex gap-6">
      <button 
        onclick={() => activeTab = "system"}
        class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'system' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
      >
        Allgemeines Logbuch
      </button>
      {#if isAdmin}
        <button 
          onclick={() => activeTab = "admin"}
          class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'admin' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Admin-Audit-Log
        </button>
      {/if}
    </div>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if activeTab === "system"}
      <div class="animate-fade-in h-full">
        <AuditLog />
      </div>
    {:else if activeTab === "admin" && isAdmin}
      <div class="animate-fade-in h-full">
        <AdminAuditLog />
      </div>
    {/if}
  </div>
</div>
