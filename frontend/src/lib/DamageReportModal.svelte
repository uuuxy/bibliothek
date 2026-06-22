<script>
  import Button from "./components/ui/Button.svelte";
  let { book, onCancel, onSubmit, isSubmitting } = $props();

  let damageReason = $state("Verloren");
  let damageAmount = $state(15.00);

  function handleSubmit() {
    onSubmit(damageReason, damageAmount);
  }
</script>

{#if book}
  <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
    <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm pointer-events-none"></div>
    <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-md w-full relative z-10 border border-slate-200 animate-fade-in">
      <h3 class="text-xl font-bold text-slate-800 mb-2">Verlust/Schaden melden</h3>
      <p class="text-sm text-slate-500 mb-4">Für <strong>{book.titel}</strong> ({book.barcode_id}). Die Ausleihe wird beendet und eine Ersatzforderung an die Eltern generiert.</p>
      
      <div class="space-y-4">
        <div>
          <label for="damage-reason" class="block text-sm font-semibold text-slate-700 mb-1">Grund</label>
          <input id="damage-reason" type="text" bind:value={damageReason} placeholder="z.B. Wasserschaden, Verloren..." class="w-full bg-slate-50 border border-slate-200 rounded-xl p-3 text-slate-800 focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 outline-none transition-all" />
        </div>
        <div>
          <label for="damage-amount" class="block text-sm font-semibold text-slate-700 mb-1">Ersatzbetrag (€)</label>
          <input id="damage-amount" type="number" step="0.01" min="0" bind:value={damageAmount} class="w-full bg-slate-50 border border-slate-200 rounded-xl p-3 text-slate-800 focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 outline-none transition-all" />
        </div>
        <div class="flex gap-3 justify-end pt-4">
          <Button variant="ghost" onclick={onCancel} disabled={isSubmitting}>Abbrechen</Button>
          <Button variant="danger-solid" onclick={handleSubmit} disabled={isSubmitting || !damageReason.trim() || damageAmount < 0}>
            {isSubmitting ? 'Wird gemeldet...' : 'Melden & PDF generieren'}
          </Button>
        </div>
      </div>
    </div>
  </div>
{/if}
