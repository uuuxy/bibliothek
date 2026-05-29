<script>
  // State Runes with JSDoc type annotations
  let barcode = $state("");
  
  /** @type {any} */
  let lastResult = $state(null); // { barcode_id, titel, imRegalErwartet, success: bool, message: string }
  
  /** @type {any[]} */
  let scanLog = $state([]);

  // Kiosk aggressive focus: Keeps text cursor focused in the inventory input field at all times
  $effect(() => {
    const handleFocus = () => {
      const inp = document.getElementById("inventur-input");
      if (inp && document.activeElement !== inp) {
        inp.focus();
      }
    };
    document.addEventListener("click", handleFocus);
    handleFocus(); // Auto-focus on mount
    return () => document.removeEventListener("click", handleFocus);
  });

  // Submit scan to backend
  /**
   * @param {SubmitEvent} [e]
   */
  async function handleScan(e) {
    if (e) e.preventDefault();
    const b = barcode.trim();
    if (!b) return;

    barcode = ""; // Instantly reset input for rapid successive scanning
    lastResult = null;

    try {
      const res = await fetch("/api/inventur/scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ barcode_id: b })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Fehler beim Inventurscan");
      }

      const data = await res.json();
      lastResult = {
        barcode_id: data.barcode_id,
        titel: data.titel,
        cover_url: data.cover_url,
        imRegalErwartet: data.im_regal_erwartet,
        success: true,
        message: data.im_regal_erwartet 
          ? "Buch im Regal erwartet (Korrekt)" 
          : "Fehl am Platz: Buch sollte eigentlich verliehen sein!"
      };

      scanLog = [lastResult, ...scanLog];
    } catch (err) {
      const error = /** @type {any} */ (err);
      lastResult = {
        barcode_id: b,
        titel: "Fehlgeschlagen",
        cover_url: "",
        imRegalErwartet: false,
        success: false,
        message: error.message || String(error)
      };
      scanLog = [lastResult, ...scanLog];
    }
  }
</script>

<div class="w-full flex flex-col md:flex-row gap-8 items-stretch text-slate-800">
  
  <!-- Left Side: Distraction-Free Scanning Field & Large Status Indicators -->
  <div class="flex-1 flex flex-col justify-between space-y-6">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Bestandsabgleich</span>
      <h2 class="text-2xl font-bold text-slate-900">Inventur-Scanner</h2>
      <p class="text-xs text-slate-500 font-medium">Scanne Buchexemplare direkt am Regal, um den Standort abzugleichen.</p>
    </div>

    <!-- Live Giant Status Display Indicator (Riesige, weiche Kachel) -->
    <div class="grow flex items-center justify-center min-h-[260px]">
      {#if !lastResult}
        <div class="text-center p-12 border-2 border-dashed border-slate-200 bg-slate-50/50 rounded-3xl w-full flex flex-col items-center justify-center space-y-4">
          <span class="text-5xl animate-pulse">📡</span>
          <p class="text-slate-450 text-sm font-semibold">Warte auf Barcode-Scans...</p>
          <span class="text-[10px] text-slate-400 uppercase tracking-widest font-mono">Scanner bereit</span>
        </div>
      {:else}
        <!-- Green (Correct) / Red (Wrong) Status Boxes -->
        <div class="w-full p-8 rounded-3xl border transition-all duration-300 animate-scale-up shadow-sm flex items-center space-x-6 text-left
          {lastResult.success && lastResult.imRegalErwartet 
            ? 'bg-emerald-50/70 border-emerald-100 text-emerald-800' 
            : 'bg-rose-50/70 border-rose-100 text-rose-800'}">
          
          {#if lastResult.cover_url}
            <img src={lastResult.cover_url} class="w-20 aspect-3/4 object-cover rounded-xl shadow-md border border-slate-100/50 shrink-0" alt="Cover" />
          {:else}
            <div class="w-20 aspect-3/4 rounded-xl shadow-md shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-650 text-xl border border-indigo-600/10">
              {lastResult.titel ? lastResult.titel.charAt(0).toUpperCase() : '?'}
            </div>
          {/if}

          <div class="space-y-2 grow min-w-0">
            <div class="flex items-center justify-between">
              <span class="text-xs font-mono font-bold tracking-widest uppercase py-1 px-3 border rounded-full
                {lastResult.success && lastResult.imRegalErwartet 
                  ? 'bg-emerald-100/50 border-emerald-250/30 text-emerald-700' 
                  : 'bg-rose-100/50 border-rose-250/30 text-rose-700'}">
                {lastResult.barcode_id}
              </span>
              <div class="text-3xl">
                {lastResult.success && lastResult.imRegalErwartet ? '✅' : '❌'}
              </div>
            </div>
            <h3 class="text-lg font-bold tracking-tight truncate text-slate-900">{lastResult.titel}</h3>
            <p class="text-xs font-semibold leading-normal">{lastResult.message}</p>
          </div>
        </div>
      {/if}
    </div>

    <!-- Scanner Input -->
    <form onsubmit={handleScan} class="w-full relative">
      <input 
        id="inventur-input" 
        type="text" 
        autocomplete="off" 
        bind:value={barcode} 
        class="w-full py-4 pl-12 pr-4 bg-slate-50 border border-slate-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/10 focus:outline-none rounded-2xl text-center text-slate-800 font-mono tracking-widest text-sm" 
        placeholder="Barcode am Regal scannen..." 
      />
      <div class="absolute left-4 top-1/2 -translate-y-1/2 text-slate-400">
        🔍
      </div>
    </form>
  </div>

  <!-- Right Side: Scans List Logs for the Session -->
  <div class="w-full md:w-80 border border-slate-100 bg-slate-50/30 rounded-3xl p-6 flex flex-col justify-between">
    <div class="space-y-4 grow flex flex-col">
      <h3 class="font-bold text-slate-800 border-b border-slate-100 pb-3 text-xs uppercase tracking-wider font-mono">Scanverlauf</h3>
      
      {#if scanLog.length === 0}
        <div class="grow flex items-center justify-center text-center text-slate-400 text-xs py-20 font-mono font-medium">Keine Scans in dieser Sitzung</div>
      {:else}
        <div class="grow overflow-y-auto max-h-[360px] space-y-2 pr-1">
          {#each scanLog as logItem}
            <div class="p-3 rounded-2xl border flex items-center justify-between gap-3 text-xs shadow-xs transition-colors hover:bg-slate-50
              {logItem.success && logItem.imRegalErwartet 
                ? 'bg-emerald-50/50 border-emerald-100/50 text-emerald-800' 
                : 'bg-rose-50/50 border-rose-100/50 text-rose-800'}">
              
              <div class="flex items-center space-x-3 truncate">
                {#if logItem.cover_url}
                  <img src={logItem.cover_url} class="w-10 aspect-3/4 object-cover rounded-md shadow-xs border border-slate-100/50 shrink-0" alt="Cover" />
                {:else}
                  <div class="w-10 aspect-3/4 rounded-md shadow-xs shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-650 text-[10px] border border-indigo-600/10">
                    {logItem.titel ? logItem.titel.charAt(0).toUpperCase() : '?'}
                  </div>
                {/if}
                <div class="truncate">
                  <p class="font-bold font-mono text-[9px] tracking-wider opacity-60 leading-none">{logItem.barcode_id}</p>
                  <p class="text-[11px] font-semibold text-slate-750 truncate mt-1 leading-tight">{logItem.titel}</p>
                </div>
              </div>
              <span class="text-xs shrink-0">{logItem.success && logItem.imRegalErwartet ? '🟢' : '🔴'}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
