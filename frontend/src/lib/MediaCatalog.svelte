<script>
  import InventurCatalog from "../inventur/routes/+page.svelte";
  import InventurAdmin from "../inventur/routes/admin/+page.svelte";

  let activeView = $state("catalog"); // "catalog" | "admin"
</script>

<div class="w-full space-y-6">


  <!-- Google-style underline Tabs -->
  <div class="border-b border-slate-200">
    <nav class="flex gap-6" aria-label="Medienkatalog Navigation">
      <button
        onclick={() => activeView = "catalog"}
        class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeView === 'catalog'
          ? 'text-blue-600'
          : 'text-slate-500 hover:text-slate-700'}"
        role="tab"
        aria-selected={activeView === "catalog"}
      >
        Suche & Filter
        {#if activeView === "catalog"}
          <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
        {/if}
      </button>
      <button
        onclick={() => activeView = "admin"}
        class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeView === 'admin'
          ? 'text-blue-600'
          : 'text-slate-500 hover:text-slate-700'}"
        role="tab"
        aria-selected={activeView === "admin"}
      >
        Titel-Verwaltung
        {#if activeView === "admin"}
          <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
        {/if}
      </button>
    </nav>
  </div>

  <!-- Content -->
  <div class="w-full">
    {#if activeView === "catalog"}
      <InventurCatalog />
    {:else if activeView === "admin"}
      <InventurAdmin />
    {/if}
  </div>
</div>
