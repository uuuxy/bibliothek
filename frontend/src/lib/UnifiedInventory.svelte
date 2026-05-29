<script>
  import InventoryMode from "./InventoryMode.svelte";
  import InventurCatalog from "../inventur/routes/+page.svelte";
  import InventurAdmin from "../inventur/routes/admin/+page.svelte";

  let currentView = $state("scanner"); // "scanner", "catalog", "admin"
</script>

<div class="w-full space-y-6">
  <!-- Sub-navigation Tabs (Segmented Control style) -->
  <div class="flex bg-slate-100 p-1 rounded-xl w-full max-w-2xl border border-slate-200/40 no-print">
    <button onclick={() => currentView = "scanner"} class="flex-1 text-center py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer {currentView === 'scanner' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-700'}">
      Barcode-Abgleich (Regal)
    </button>
    <button onclick={() => currentView = "catalog"} class="flex-1 text-center py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer {currentView === 'catalog' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-700'}">
      Filter & Suche (Katalog)
    </button>
    <button onclick={() => currentView = "admin"} class="flex-1 text-center py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer {currentView === 'admin' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-700'}">
      Titel-Verwaltung (Admin)
    </button>
  </div>

  <div class="w-full">
    {#if currentView === "scanner"}
      <InventoryMode />
    {:else if currentView === "catalog"}
      <InventurCatalog />
    {:else if currentView === "admin"}
      <InventurAdmin />
    {/if}
  </div>
</div>
