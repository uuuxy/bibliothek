<script>
  import { onMount } from 'svelte';
  import { fade, slide } from 'svelte/transition';
  import { apiFetch } from './apiFetch.js';
  
  let state = $state({
    status: 'idle', // 'idle', 'active'
    scopeType: 'global',
    selectedSignatureId: '',
    signatures: [],
    stats: { erwartet: 0, erfasst: 0 },
    lastScan: null,
    barcodeInput: '',
    isScanning: false,
    showStartModal: false,
    showFinishModal: false,
    errorMessage: ''
  });

  let startDialog;
  let finishDialog;
  let barcodeInputEl;

  $effect(() => {
    if (state.showStartModal && startDialog) {
      startDialog.showModal();
    } else if (!state.showStartModal && startDialog) {
      startDialog.close();
    }
  });

  $effect(() => {
    if (state.showFinishModal && finishDialog) {
      finishDialog.showModal();
    } else if (!state.showFinishModal && finishDialog) {
      finishDialog.close();
    }
  });

  $effect(() => {
    if (state.status === 'active' && barcodeInputEl && !state.isScanning) {
      barcodeInputEl.focus();
    }
  });

  onMount(async () => {
    try {
      const res = await apiFetch('/api/signatures');
      if (res.ok) {
        state.signatures = await res.json();
      }
    } catch (e) {
      console.error("Failed to load signatures", e);
    }
  });

  async function startInventory() {
    state.errorMessage = '';
    const payload = { type: state.scopeType };
    if (state.scopeType === 'signature') {
      if (!state.selectedSignatureId) {
        state.errorMessage = 'Bitte wähle eine Signatur aus.';
        return;
      }
      payload.signature_id = Number(state.selectedSignatureId);
    }

    try {
      const res = await apiFetch('/api/inventur/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        const data = await res.json();
        state.stats.erwartet = data.erwartet;
        state.stats.erfasst = 0;
        state.lastScan = null;
        state.status = 'active';
        state.showStartModal = false;
      } else {
        state.errorMessage = 'Fehler beim Starten der Inventur.';
      }
    } catch (e) {
      state.errorMessage = 'Netzwerkfehler.';
    }
  }

  async function handleScan(e) {
    e.preventDefault();
    if (!state.barcodeInput.trim() || state.isScanning) return;
    
    state.isScanning = true;
    const barcode = state.barcodeInput.trim();
    state.barcodeInput = '';
    
    try {
      const res = await apiFetch('/api/inventur/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ barcode_id: barcode })
      });
      
      if (res.ok) {
        const data = await res.json();
        state.stats.erfasst++;
        state.lastScan = {
          success: true,
          barcode: data.barcode_id,
          title: data.titel,
          warnings: data.warnungen || []
        };
      } else {
        const err = await res.json();
        state.lastScan = {
          success: false,
          barcode: barcode,
          title: 'Unbekanntes Buch',
          warnings: [err.error || 'Scan fehlgeschlagen']
        };
      }
    } catch (e) {
      state.lastScan = {
        success: false,
        barcode: barcode,
        title: 'Fehler',
        warnings: ['Netzwerkfehler beim Scannen']
      };
    } finally {
      state.isScanning = false;
      if (barcodeInputEl) barcodeInputEl.focus();
    }
  }

  async function finishInventory() {
    try {
      const res = await apiFetch('/api/inventur/finish', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        alert(`Inventur abgeschlossen! ${data.verloren_gemeldet} Bücher wurden als verloren markiert.`);
        state.status = 'idle';
        state.showFinishModal = false;
        state.stats = { erwartet: 0, erfasst: 0 };
        state.lastScan = null;
      } else {
        alert('Fehler beim Abschließen der Inventur.');
      }
    } catch (e) {
      alert('Netzwerkfehler.');
    }
  }

  function getProgressPercent() {
    if (state.stats.erwartet === 0) return 0;
    return Math.min(100, Math.round((state.stats.erfasst / state.stats.erwartet) * 100));
  }
</script>

<div class="max-w-4xl mx-auto w-full p-4 md:p-6 space-y-6 animate-fade-in">
  {#if state.status === 'idle'}
    <div class="bg-white rounded-2xl shadow-sm border border-slate-200 p-12 text-center flex flex-col items-center justify-center space-y-6">
      <div class="w-20 h-20 bg-blue-50 text-blue-500 rounded-full flex items-center justify-center">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-10 h-10">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.35 3.836c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m8.9-4.414c.376.023.75.05 1.124.08 1.131.094 1.976 1.057 1.976 2.192V16.5A2.25 2.25 0 0118 18.75h-2.25m-7.5-10.5H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V18.75m-7.5-10.5h6.375c.621 0 1.125.504 1.125 1.125v9.375m-8.25-3l1.5 1.5 3-3.75" />
        </svg>
      </div>
      <div>
        <h3 class="text-xl font-bold text-slate-900">Keine Inventur aktiv</h3>
        <p class="text-slate-500 mt-2 max-w-md mx-auto">Starte einen neuen Inventur-Lauf. Du kannst entweder die gesamte Bibliothek prüfen oder gezielt nach einer bestimmten Signatur / Kategorie scannen.</p>
      </div>
      <button onclick={() => state.showStartModal = true} class="bg-blue-600 hover:bg-blue-700 text-white font-medium px-6 py-3 rounded-xl shadow-sm transition-colors flex items-center space-x-2">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        <span>Neue Bestandsprüfung starten</span>
      </button>
    </div>
  {:else}
    <div class="space-y-6">
      <!-- Progress & Stats -->
      <div class="bg-white rounded-2xl shadow-sm border border-slate-200 p-6">
        <div class="flex justify-between items-end mb-4">
          <div>
            <span class="text-sm font-semibold text-slate-500 uppercase tracking-wider">Aktueller Fortschritt</span>
            <div class="text-2xl font-bold text-slate-900 mt-1">{state.stats.erfasst} / {state.stats.erwartet} <span class="text-base font-medium text-slate-400">erfasst</span></div>
          </div>
          <div class="text-3xl font-bold text-blue-600">{getProgressPercent()}%</div>
        </div>
        <div class="w-full bg-slate-100 rounded-full h-3 overflow-hidden">
          <div class="bg-blue-600 h-3 rounded-full transition-all duration-500 ease-out" style="width: {getProgressPercent()}%"></div>
        </div>
      </div>

      <!-- Scanner Input -->
      <form onsubmit={handleScan} class="relative">
        <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-slate-400">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 6.75h.75v.75h-.75v-.75zM6.75 16.5h.75v.75h-.75v-.75zM16.5 6.75h.75v.75h-.75v-.75zM13.5 13.5h.75v.75h-.75v-.75zM13.5 19.5h.75v.75h-.75v-.75zM19.5 13.5h.75v.75h-.75v-.75zM19.5 19.5h.75v.75h-.75v-.75zM16.5 16.5h.75v.75h-.75v-.75z" />
          </svg>
        </div>
        <input 
          bind:this={barcodeInputEl}
          bind:value={state.barcodeInput}
          type="text" 
          placeholder="Barcode scannen..." 
          class="w-full pl-12 pr-4 py-4 bg-white border-2 border-blue-100 rounded-2xl shadow-sm text-lg font-medium focus:ring-4 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all placeholder-slate-400"
          disabled={state.isScanning}
        />
        {#if state.isScanning}
          <div class="absolute inset-y-0 right-0 pr-4 flex items-center">
            <div class="w-5 h-5 border-2 border-slate-300 border-t-blue-600 rounded-full animate-spin"></div>
          </div>
        {/if}
      </form>

      <!-- Feedback Area -->
      {#if state.lastScan}
        <div transition:slide class="rounded-2xl p-6 border {state.lastScan.success && state.lastScan.warnings.length === 0 ? 'bg-emerald-50 border-emerald-200' : (!state.lastScan.success ? 'bg-red-50 border-red-200' : 'bg-amber-50 border-amber-200')}">
          <div class="flex items-start space-x-4">
            {#if state.lastScan.success && state.lastScan.warnings.length === 0}
              <div class="p-2 bg-emerald-100 rounded-full text-emerald-600 shrink-0">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
              </div>
            {:else if !state.lastScan.success}
              <div class="p-2 bg-red-100 rounded-full text-red-600 shrink-0">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
              </div>
            {:else}
              <div class="p-2 bg-amber-100 rounded-full text-amber-600 shrink-0">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
              </div>
            {/if}
            
            <div class="flex-1">
              <h4 class="text-lg font-bold {state.lastScan.success && state.lastScan.warnings.length === 0 ? 'text-emerald-900' : (!state.lastScan.success ? 'text-red-900' : 'text-amber-900')}">
                {state.lastScan.title}
              </h4>
              <p class="text-sm font-medium mt-1 {state.lastScan.success && state.lastScan.warnings.length === 0 ? 'text-emerald-700' : (!state.lastScan.success ? 'text-red-700' : 'text-amber-700')}">
                Barcode: {state.lastScan.barcode}
              </p>
              
              {#if state.lastScan.warnings.length > 0}
                <ul class="mt-3 space-y-1">
                  {#each state.lastScan.warnings as warn}
                    <li class="flex items-start text-sm {!state.lastScan.success ? 'text-red-800' : 'text-amber-800'}">
                      <span class="mr-2 mt-0.5">•</span>
                      <span>{warn}</span>
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
          </div>
        </div>
      {/if}

      <div class="pt-8 border-t border-slate-200 flex justify-end">
        <button onclick={() => state.showFinishModal = true} class="bg-red-50 hover:bg-red-100 text-red-600 font-semibold px-6 py-3 rounded-xl border border-red-200 transition-colors">
          Inventur abschließen
        </button>
      </div>
    </div>
  {/if}
</div>

<!-- Start Modal -->
<dialog bind:this={startDialog} class="backdrop:bg-slate-900/50 backdrop:backdrop-blur-sm bg-transparent p-0 w-full max-w-md m-auto border-0 rounded-2xl shadow-2xl overflow-hidden">
  <div class="bg-white w-full">
    <div class="p-6 border-b border-slate-100">
      <h2 class="text-xl font-bold text-slate-900">Inventur-Scope wählen</h2>
      <p class="text-sm text-slate-500 mt-1">Welcher Teil der Bibliothek soll geprüft werden?</p>
    </div>
    <div class="p-6 space-y-6">
      {#if state.errorMessage}
        <div class="p-3 bg-red-50 text-red-700 text-sm font-medium rounded-lg">{state.errorMessage}</div>
      {/if}

      <div class="space-y-3">
        <label class="flex items-start space-x-3 p-3 border rounded-xl cursor-pointer transition-colors {state.scopeType === 'global' ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:bg-slate-50'}">
          <div class="flex items-center h-5 mt-0.5">
            <input type="radio" bind:group={state.scopeType} value="global" class="w-4 h-4 text-blue-600 border-slate-300 focus:ring-blue-600">
          </div>
          <div class="flex-1">
            <div class="font-bold text-slate-900">Komplette Bibliothek</div>
            <div class="text-xs text-slate-500">Prüfe den gesamten Bestand ab.</div>
          </div>
        </label>

        <label class="flex items-start space-x-3 p-3 border rounded-xl cursor-pointer transition-colors {state.scopeType === 'signature' ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:bg-slate-50'}">
          <div class="flex items-center h-5 mt-0.5">
            <input type="radio" bind:group={state.scopeType} value="signature" class="w-4 h-4 text-blue-600 border-slate-300 focus:ring-blue-600">
          </div>
          <div class="flex-1">
            <div class="font-bold text-slate-900">Nur bestimmte Signatur</div>
            <div class="text-xs text-slate-500">Grenze die Inventur auf eine Kategorie ein.</div>
          </div>
        </label>
      </div>

      {#if state.scopeType === 'signature'}
        <div transition:slide class="space-y-2">
          <label class="block text-sm font-medium text-slate-700">Signatur auswählen</label>
          <select bind:value={state.selectedSignatureId} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-3 text-slate-900 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none">
            <option value="" disabled selected>-- Bitte wählen --</option>
            {#each state.signatures as sig}
              <option value={sig.id}>{sig.name}</option>
            {/each}
          </select>
        </div>
      {/if}
    </div>
    <div class="p-4 bg-slate-50 border-t border-slate-100 flex justify-end space-x-3">
      <button onclick={() => state.showStartModal = false} class="px-5 py-2.5 text-sm font-semibold text-slate-600 hover:bg-slate-200 rounded-xl transition-colors">Abbrechen</button>
      <button onclick={startInventory} class="px-5 py-2.5 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-sm transition-colors">Inventur Starten</button>
    </div>
  </div>
</dialog>

<!-- Finish Modal -->
<dialog bind:this={finishDialog} class="backdrop:bg-slate-900/50 backdrop:backdrop-blur-sm bg-transparent p-0 w-full max-w-md m-auto border-0 rounded-2xl shadow-2xl overflow-hidden">
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
      <button onclick={() => state.showFinishModal = false} class="px-5 py-2.5 text-sm font-semibold text-slate-600 hover:bg-slate-200 rounded-xl transition-colors">Abbrechen</button>
      <button onclick={finishInventory} class="px-5 py-2.5 text-sm font-semibold text-white bg-red-600 hover:bg-red-700 rounded-xl shadow-sm transition-colors">Ja, unwiderruflich abschließen</button>
    </div>
  </div>
</dialog>
