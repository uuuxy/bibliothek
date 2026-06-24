<script>
  let { suppliers, onAddSupplier, onRemoveSupplier } = $props();
  
  let newName = $state("");
  let newEmail = $state("");
  let newCustNum = $state("");

  /** @param {SubmitEvent} e */
  function handleSubmit(e) {
    e.preventDefault();
    onAddSupplier(newName, newEmail, newCustNum);
    newName = ""; newEmail = ""; newCustNum = "";
  }
</script>

<div class="grid grid-cols-1 md:grid-cols-3 gap-8 items-start overflow-y-auto">
  <div class="space-y-4">
    <h2 class="text-base font-bold text-slate-800 border-b border-gray-200 pb-3">Neuer Lieferant</h2>
    <form onsubmit={handleSubmit} class="space-y-4 text-base">
      <div class="space-y-1.5"><label for="n" class="block font-medium text-gray-600 text-sm">Name</label><input id="n" type="text" bind:value={newName} required class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base" /></div>
      <div class="space-y-1.5"><label for="e" class="block font-medium text-gray-600 text-sm">E-Mail</label><input id="e" type="email" bind:value={newEmail} required class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base" /></div>
      <div class="space-y-1.5"><label for="c" class="block font-medium text-gray-600 text-sm">Kundennummer</label><input id="c" type="text" bind:value={newCustNum} required class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base" /></div>
      <button type="submit" class="w-full py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg cursor-pointer text-base">💾 Lieferanten speichern</button>
    </form>
  </div>

  <div class="md:col-span-2 space-y-4">
    <h2 class="text-base font-bold text-slate-800 border-b border-gray-200 pb-3">Aktive Lieferanten</h2>
    {#if !suppliers.length}
      <div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
    {:else}
      <table class="w-full text-left border-collapse text-base">
        <thead>
          <tr class="border-b border-gray-200 text-sm font-semibold text-gray-500"><th class="py-2.5">Name</th><th class="py-2.5">E-Mail</th><th class="py-2.5">Kundennummer</th><th class="py-2.5 text-right">Aktion</th></tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          {#each suppliers as s}
            <tr class="hover:bg-slate-50/40">
              <td class="py-3 font-bold text-slate-800">{s.name}</td>
              <td class="py-3 text-slate-650">{s.email}</td>
              <td class="py-3 text-slate-650">{s.customerNumber}</td>
              <td class="py-3 text-right"><button onclick={() => onRemoveSupplier(s.id)} class="text-slate-450 hover:text-rose-600 cursor-pointer">Löschen</button></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
