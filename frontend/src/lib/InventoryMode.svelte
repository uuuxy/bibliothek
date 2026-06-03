<script>
  import { apiFetch } from "./apiFetch.js";
  import { playSuccessBeep, playErrorBeep } from "./audio.js";
  // State Runes with JSDoc type annotations
  let barcode = $state("");
  
  /** @type {any} */
  let lastResult = $state(null); // { barcode_id, titel, imRegalErwartet, success: bool, message: string }
  
  /** @type {any[]} */
  let scanLog = $state([]);

  // Fehlbestand state
  let fehlbestandTage = $state(30);
  let fehlbestandLoading = $state(false);
  /** @type {any[] | null} */
  let fehlbestandListe = $state(null);

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
      const res = await apiFetch("/api/inventur/scan", {
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

      if (lastResult.imRegalErwartet) {
        playSuccessBeep();
      } else {
        playErrorBeep();
      }

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
      playErrorBeep();
      scanLog = [lastResult, ...scanLog];
    } finally {
      // Force refocus on the input field
      const inp = document.getElementById("inventur-input");
      if (inp) {
        setTimeout(() => inp.focus(), 10);
      }
    }
  }

  async function ermittleFehlbestand() {
    fehlbestandLoading = true;
    fehlbestandListe = null;
    try {
      const res = await apiFetch(`/api/inventur/fehlbestand?tage=${fehlbestandTage}`);
      if (!res.ok) throw new Error(await res.text());
      fehlbestandListe = await res.json();
    } catch (err) {
      const error = /** @type {any} */ (err);
      fehlbestandListe = [];
      console.error("Fehlbestand-Fehler:", error.message);
    } finally {
      fehlbestandLoading = false;
    }
  }

  /**
   * @param {Date | null | undefined} d
   */
  function formatDate(d) {
    if (!d) return "Nie geprüft";
    return new Date(d).toLocaleDateString("de-DE", { day: "2-digit", month: "2-digit", year: "numeric" });
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
          <span class="text-[10px] text-slate-400 uppercase tracking-widest">Scanner bereit</span>
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
              <span class="text-xs font-bold tracking-widest uppercase py-1 px-3 border rounded-full
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
        class="w-full py-4 pl-12 pr-4 bg-slate-50 border border-slate-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/10 focus:outline-none rounded-2xl text-center text-slate-800 tracking-widest text-sm" 
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
      <h3 class="font-bold text-slate-800 border-b border-slate-100 pb-3 text-xs uppercase tracking-wider">Scanverlauf</h3>
      
      {#if scanLog.length === 0}
        <div class="grow flex items-center justify-center text-center text-slate-400 text-xs py-20 font-medium">Keine Scans in dieser Sitzung</div>
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
                  <p class="font-bold text-[9px] tracking-wider opacity-60 leading-none">{logItem.barcode_id}</p>
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

<!-- Fehlbestand Panel -->
<div class="mt-8 w-full border border-slate-100 bg-white rounded-3xl p-6 space-y-5">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Vermisst-Liste</span>
      <h2 class="text-xl font-bold text-slate-900">Fehlbestand ermitteln</h2>
      <p class="text-xs text-slate-500 font-medium">Zeigt nicht ausgeliehene Bücher, deren Inventur-Scan älter als X Tage ist.</p>
    </div>
    <div class="flex items-center gap-3 shrink-0">
      <label for="fehlbestandTageInput" class="text-xs font-semibold text-slate-500 whitespace-nowrap">Älter als</label>
      <input
        id="fehlbestandTageInput"
        type="number"
        min="1"
        max="3650"
        bind:value={fehlbestandTage}
        class="w-20 py-2 px-3 bg-slate-50 border border-slate-200 focus:border-blue-400 focus:ring-2 focus:ring-blue-400/10 focus:outline-none rounded-xl text-center text-sm text-slate-800 font-semibold"
      />
      <span class="text-xs font-semibold text-slate-500">Tage</span>
      <button
        onclick={ermittleFehlbestand}
        disabled={fehlbestandLoading}
        class="px-5 py-2 rounded-xl bg-amber-500 hover:bg-amber-600 text-white font-bold text-sm transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed whitespace-nowrap"
      >
        {fehlbestandLoading ? 'Lädt...' : '🔍 Fehlbestand ermitteln'}
      </button>
    </div>
  </div>

  {#if fehlbestandListe !== null}
    {#if fehlbestandListe.length === 0}
      <div class="py-10 flex flex-col items-center justify-center text-center space-y-3">
        <span class="text-4xl">✅</span>
        <p class="text-sm font-semibold text-slate-700">Kein Fehlbestand! Alle nicht ausgeliehenen Bücher wurden in den letzten {fehlbestandTage} Tagen geprüft.</p>
      </div>
    {:else}
      <div class="border border-amber-100 rounded-2xl overflow-hidden">
        <div class="bg-amber-50 px-4 py-2.5 flex items-center justify-between border-b border-amber-100">
          <span class="text-xs font-bold text-amber-800 uppercase tracking-wider">
            {fehlbestandListe.length} Exemplar{fehlbestandListe.length !== 1 ? 'e' : ''} vermutlich vermisst
          </span>
          <span class="text-[10px] text-amber-600 font-semibold">Nicht ausgeliehen, aber nicht kürzlich geprüft</span>
        </div>
        <div class="divide-y divide-slate-100 max-h-96 overflow-y-auto">
          {#each fehlbestandListe as item}
            <div class="px-4 py-3 flex items-center gap-4 hover:bg-slate-50 transition-colors">
              {#if item.cover_url}
                <img src={item.cover_url} class="w-10 aspect-3/4 object-cover rounded-lg shadow-xs border border-slate-100 shrink-0" alt="Cover" />
              {:else}
                <div class="w-10 aspect-3/4 rounded-lg shadow-xs shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-400 to-purple-600 text-[10px] border border-indigo-500/10">
                  {item.titel ? item.titel.charAt(0).toUpperCase() : '?'}
                </div>
              {/if}
              <div class="flex-1 min-w-0">
                <p class="text-xs font-bold text-slate-700 truncate">{item.titel}</p>
                <p class="text-[10px] text-slate-500 truncate">{item.autor}</p>
                <span class="text-[9px] font-bold text-blue-600 bg-blue-50 border border-blue-100 px-1.5 py-0.5 rounded-full">{item.barcode_id}</span>
              </div>
              <div class="text-right shrink-0">
                <p class="text-[10px] font-semibold text-amber-700">Letzte Prüfung</p>
                <p class="text-xs font-bold text-slate-700">{formatDate(item.inventur_geprueft_am)}</p>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
