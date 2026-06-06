<script>
  import InventoryMode from "./InventoryMode.svelte";
  import RegalScanMode from "./RegalScanMode.svelte";

  let activeTab = $state("global"); // 'global' oder 'regal'
</script>

<div class="w-full">
  <div class="mb-6 flex space-x-1 bg-slate-100 p-1 rounded-xl w-fit">
    <button 
      onclick={() => activeTab = "global"} 
      class="px-4 py-2 text-sm font-semibold rounded-lg transition-colors cursor-pointer {activeTab === 'global' ? 'bg-white text-blue-700 shadow-sm' : 'text-slate-500 hover:text-slate-700'}"
    >
      Globale Inventur (Freier Scan)
    </button>
    <button 
      onclick={() => activeTab = "regal"} 
      class="px-4 py-2 text-sm font-semibold rounded-lg transition-colors cursor-pointer {activeTab === 'regal' ? 'bg-white text-blue-700 shadow-sm' : 'text-slate-500 hover:text-slate-700'}"
    >
      Regal-Scan (Titel-Abgleich)
    </button>
  </div>

  {#if activeTab === "global"}
    <InventoryMode />
  {:else}
    <RegalScanMode />
  {/if}
</div>
