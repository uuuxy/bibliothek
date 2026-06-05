<script>
  let { printSuggestion, onPrint } = $props();
</script>

{#if printSuggestion}
  <div class="bg-amber-50 border border-amber-200 rounded-xl p-5 shadow-2xs space-y-3.5 text-left animate-fade-in no-print">
    <div class="flex items-start gap-2.5">
      <span class="text-xl">⚠️</span>
      <div>
        <h3 class="text-xs font-bold text-amber-800 uppercase tracking-wider">Etikettendruck erforderlich</h3>
        <p class="text-xs text-amber-700 font-medium leading-relaxed mt-1">Es gibt {printSuggestion.length} Exemplare in dieser freigegebenen Lieferung, für die noch keine Barcode-Etiketten gedruckt wurden (z.B. Amazon-Bestellung).</p>
      </div>
    </div>
    <button onclick={onPrint} class="w-full py-2 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg text-xs shadow-sm cursor-pointer transition-all active:scale-95">
      🖨️ Etiketten für diese Lieferung drucken
    </button>
  </div>
{/if}
