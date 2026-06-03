<script>
  let { returnedBook = $bindable(), showDamageInput = $bindable(), damageDescription = $bindable(), isSubmittingDamage, handleDamageOk, handleDamageSubmit } = $props();
</script>

{#if returnedBook}
  <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
    <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm pointer-events-none"></div>
    <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-md w-full relative z-10 border border-slate-200">
      <h3 class="text-xl font-bold text-slate-800 mb-2">Zustand in Ordnung?</h3>
      <p class="text-sm text-slate-500 mb-6">Bitte überprüfe <strong>{returnedBook.titel}</strong> ({returnedBook.barcode_id}) auf Schäden.</p>
      
      {#if !showDamageInput}
        <div class="grid grid-cols-2 gap-4">
          <button onclick={() => showDamageInput = true} class="py-3 px-4 rounded-xl bg-rose-50 hover:bg-rose-100 text-rose-700 font-bold transition-colors cursor-pointer">Nein, Mangel melden</button>
          <button onclick={handleDamageOk} class="py-3 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition-colors shadow-sm focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 outline-none cursor-pointer">Ja, alles okay</button>
        </div>
      {:else}
        <div class="space-y-4">
          <label for="damage-description" class="block text-sm font-semibold text-slate-700">Art des Mangels (Notiz)</label>
          <textarea id="damage-description" bind:value={damageDescription} rows="3" placeholder="z.B. Wasserschaden, Seite 15 fehlt..." class="w-full bg-slate-50 border border-slate-200 rounded-xl p-3 text-slate-800 focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 outline-none resize-none transition-all"></textarea>
          <div class="flex gap-3 justify-end pt-2">
            <button onclick={() => showDamageInput = false} disabled={isSubmittingDamage} class="px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 rounded-xl transition-colors cursor-pointer">Abbrechen</button>
            <button onclick={handleDamageSubmit} disabled={isSubmittingDamage || !damageDescription.trim()} class="px-4 py-2 text-sm font-bold text-white bg-rose-600 hover:bg-rose-700 disabled:opacity-50 rounded-xl transition-colors shadow-sm cursor-pointer">Mangel speichern</button>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}
