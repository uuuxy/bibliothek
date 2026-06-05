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
  <div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
    <h2 class="text-sm font-bold text-slate-800 border-b border-slate-100 pb-3">Neuer Lieferant</h2>
    <form onsubmit={handleSubmit} class="space-y-4 text-base">
      <div class="space-y-1"><label for="n" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">Name</label><input id="n" type="text" bind:value={newName} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
      <div class="space-y-1"><label for="e" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">E-Mail</label><input id="e" type="email" bind:value={newEmail} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
      <div class="space-y-1"><label for="c" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">Kundennummer</label><input id="c" type="text" bind:value={newCustNum} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
      <button type="submit" class="w-full py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg cursor-pointer text-base">💾 Lieferanten speichern</button>
    </form>
  </div>

  <div class="md:col-span-2 bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
    <h2 class="text-sm font-bold text-slate-800 border-b border-slate-100 pb-3">Aktive Lieferanten</h2>
    {#if !suppliers.length}
      <div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
    {:else}
      <table class="w-full text-left border-collapse text-base">
        <thead>
          <tr class="border-b border-slate-100 text-sm font-bold text-slate-400 uppercase tracking-wider"><th class="py-2.5">Name</th><th class="py-2.5">E-Mail</th><th class="py-2.5">Kundennummer</th><th class="py-2.5 text-right">Aktion</th></tr>
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
