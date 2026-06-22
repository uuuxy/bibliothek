<script>
  /**
   * @type {{
   *   dialogEl: HTMLDialogElement | undefined,
   *   state: any,
   *   onClose: () => void,
   *   onFinish: () => void
   * }}
   */
  let { dialogEl = $bindable(), state, onClose, onFinish } = $props();
</script>

<dialog bind:this={dialogEl} class="backdrop:bg-slate-900/50 backdrop:backdrop-blur-sm bg-transparent p-0 w-full max-w-md m-auto border-0 rounded-2xl shadow-2xl overflow-hidden">
  <div class="bg-white w-full">
    <div class="p-6 border-b border-slate-100 flex items-center space-x-3">
      <div class="p-2 bg-red-100 text-red-600 rounded-full">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
      </div>
      <div>
        <h2 class="text-xl font-bold text-slate-900">Inventur abschließen?</h2>
      </div>
    </div>
    <div class="p-6">
      <p class="text-slate-600">Du bist dabei, die aktuelle Inventur zu beenden.</p>
      <div class="mt-4 p-4 bg-red-50 rounded-xl border border-red-100">
        <p class="text-sm text-red-800 font-medium">Achtung: Alle <span class="font-bold">{Math.max(0, state.stats.erwartet - state.stats.erfasst)}</span> Bücher aus dem aktuellen Scope, die nicht gescannt wurden, werden unwiderruflich als <span class="font-bold">Verloren</span> markiert und ausgesondert.</p>
      </div>
    </div>
    <div class="p-4 bg-slate-50 border-t border-slate-100 flex justify-end space-x-3">
      <button onclick={onClose} class="px-5 py-2.5 text-sm font-semibold text-slate-600 hover:bg-slate-200 rounded-xl transition-colors">Abbrechen</button>
      <button onclick={onFinish} class="px-5 py-2.5 text-sm font-semibold text-white bg-red-600 hover:bg-red-700 rounded-xl shadow-sm transition-colors">Ja, unwiderruflich abschließen</button>
    </div>
  </div>
</dialog>
