<script>
  import { onMount, onDestroy } from "svelte";
  import { appState } from "../inventur/lib/store.svelte.js";

  let isOpen = $state(false);
  let transactions = $state(/** @type {any[]} */ ([]));
  let isLoading = $state(false);
  let dropdownRef = $state(/** @type {HTMLDivElement|null} */ (null));
  let buttonRef = $state(/** @type {HTMLButtonElement|null} */ (null));

  let posX = $state(typeof window !== 'undefined' ? window.innerWidth - 80 : 0);
  let posY = $state(typeof window !== 'undefined' ? window.innerHeight - 80 : 0);
  let isDragging = $state(false);
  let startX = 0;
  let startY = 0;
  let startPosX = 0;
  let startPosY = 0;
  let wasDragged = false;

  let dropdownClass = $derived.by(() => {
    if (typeof window === 'undefined') return 'mt-3 right-0 origin-top-right';
    const isRightHalf = posX > window.innerWidth / 2;
    const isBottomHalf = posY > window.innerHeight / 2;
    
    let classes = [];
    if (isRightHalf) {
      classes.push("right-0");
      if (isBottomHalf) classes.push("bottom-full mb-3 origin-bottom-right");
      else classes.push("mt-3 origin-top-right");
    } else {
      classes.push("left-0");
      if (isBottomHalf) classes.push("bottom-full mb-3 origin-bottom-left");
      else classes.push("mt-3 origin-top-left");
    }
    return classes.join(" ");
  });

  async function fetchTransactions() {
    isLoading = true;
    try {
      const res = await fetch("/api/transactions/recent");
      if (res.ok) {
        transactions = await res.json() || [];
      } else {
        transactions = [];
      }
    } catch (err) {
      transactions = [];
    } finally {
      isLoading = false;
    }
  }

  function toggleDropdown() {
    isOpen = !isOpen;
    if (isOpen) {
      fetchTransactions();
    }
  }

  function handleToggleClick(/** @type {Event} */ e) {
    if (wasDragged) {
      wasDragged = false;
      return;
    }
    toggleDropdown();
  }

  /**
   * @param {string} barcode
   */
  function handleTransactionClick(barcode) {
    if (!barcode) return;
    appState.triggerStudentScan = barcode;
    isOpen = false;
  }

  function formatTime(/** @type {string} */ ts) {
    const d = new Date(ts);
    const now = new Date();
    const diffMs = now.getTime() - d.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return "gerade eben";
    if (diffMins < 60) return `vor ${diffMins} Min`;
    
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `vor ${diffHours} Std`;
    
    return d.toLocaleDateString("de-DE");
  }

  // --- Drag & Drop Logic ---
  function onDragStart(/** @type {MouseEvent|TouchEvent} */ e) {
    // Only left click or touch
    if (e instanceof MouseEvent && e.button !== 0) return; 
    
    isOpen = false;
    isDragging = true;
    wasDragged = false;
    
    const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
    const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;
    
    startX = clientX;
    startY = clientY;
    startPosX = posX;
    startPosY = posY;
    
    window.addEventListener('mousemove', onDragMove);
    window.addEventListener('mouseup', onDragEnd);
    window.addEventListener('touchmove', onDragMove, {passive: false});
    window.addEventListener('touchend', onDragEnd);
  }

  function onDragMove(/** @type {MouseEvent|TouchEvent} */ e) {
    if (!isDragging) return;
    
    const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
    const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;
    
    const dx = clientX - startX;
    const dy = clientY - startY;
    
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
      wasDragged = true;
    }
    
    posX = startPosX + dx;
    posY = startPosY + dy;
  }

  function onDragEnd() {
    if (!isDragging) return;
    isDragging = false;
    
    window.removeEventListener('mousemove', onDragMove);
    window.removeEventListener('mouseup', onDragEnd);
    window.removeEventListener('touchmove', onDragMove);
    window.removeEventListener('touchend', onDragEnd);

    // Snapping to screen bounds
    const buttonWidth = 48;
    const buttonHeight = 48;
    if (posX < 10) posX = 10;
    if (posY < 10) posY = 10;
    if (posX > window.innerWidth - buttonWidth - 10) posX = window.innerWidth - buttonWidth - 10;
    if (posY > window.innerHeight - buttonHeight - 10) posY = window.innerHeight - buttonHeight - 10;

    localStorage.setItem("bibliothek_history_icon_pos", JSON.stringify({ x: posX, y: posY }));
  }

  // Handle outside click
  /** @param {MouseEvent} event */
  function handleClickOutside(event) {
    if (isOpen && dropdownRef && buttonRef && !dropdownRef.contains(/** @type {Node} */ (event.target)) && !buttonRef.contains(/** @type {Node} */ (event.target))) {
      isOpen = false;
    }
  }

  onMount(() => {
    document.addEventListener("mousedown", handleClickOutside);
    const saved = localStorage.getItem("bibliothek_history_icon_pos");
    if (saved) {
      try {
        const p = JSON.parse(saved);
        // Additional bounds check for resized windows
        if (p.x > window.innerWidth) p.x = window.innerWidth - 80;
        if (p.y > window.innerHeight) p.y = window.innerHeight - 80;
        posX = p.x;
        posY = p.y;
      } catch(e) {}
    }
  });

  onDestroy(() => {
    document.removeEventListener("mousedown", handleClickOutside);
    if (typeof window !== 'undefined') {
      window.removeEventListener('mousemove', onDragMove);
      window.removeEventListener('mouseup', onDragEnd);
      window.removeEventListener('touchmove', onDragMove);
      window.removeEventListener('touchend', onDragEnd);
    }
  });
</script>

<div 
  class="fixed z-50 font-sans no-print flex shrink-0"
  style="left: {posX}px; top: {posY}px;"
>
  <button 
    bind:this={buttonRef}
    onclick={handleToggleClick}
    onmousedown={onDragStart}
    ontouchstart={onDragStart}
    class="relative flex items-center justify-center p-3 bg-white border border-slate-200 hover:border-slate-300 rounded-full text-slate-500 hover:text-blue-600 cursor-grab focus:outline-none focus:ring-4 focus:ring-blue-500/20 transition-shadow {isDragging ? 'cursor-grabbing scale-105 shadow-xl border-blue-300 ring-2 ring-blue-500/30' : 'shadow-md hover:shadow-lg transition-all'}"
    title="Letzte Transaktionen"
    aria-label="Transaktionsverlauf öffnen"
    aria-expanded={isOpen}
  >
    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  </button>

  {#if isOpen}
    <div 
      bind:this={dropdownRef}
      class="absolute w-80 md:w-96 bg-white border border-slate-100 shadow-2xl rounded-2xl overflow-hidden animate-slide-up transform {dropdownClass}"
    >
      <div class="px-5 py-4 border-b border-slate-100 bg-slate-50 flex items-center justify-between">
        <h3 class="font-bold text-slate-700 text-sm tracking-wide uppercase">Letzte Transaktionen</h3>
        <button onclick={() => fetchTransactions()} class="text-slate-400 hover:text-blue-600 transition-colors" title="Aktualisieren">
          <svg class="w-4 h-4 {isLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
        </button>
      </div>

      <div class="max-h-[60vh] overflow-y-auto custom-scrollbar">
        {#if isLoading && transactions.length === 0}
          <div class="p-8 flex justify-center">
            <div class="w-6 h-6 border-2 border-slate-200 border-t-blue-600 rounded-full animate-spin"></div>
          </div>
        {:else if transactions.length === 0}
          <div class="p-8 text-center text-slate-400 text-sm font-medium">
            Keine Transaktionen gefunden.
          </div>
        {:else}
          <ul class="divide-y divide-slate-50">
            {#each transactions as tx}
              {@const isCheckout = tx.aktion === 'CHECKOUT'}
              <li>
                <button 
                  onclick={() => handleTransactionClick(tx.schueler_barcode)}
                  class="w-full text-left p-4 hover:bg-slate-50 transition-colors flex items-start space-x-3 cursor-pointer group focus:outline-none focus:bg-slate-50"
                >
                  <div class="mt-0.5 shrink-0 w-8 h-8 rounded-full flex items-center justify-center {isCheckout ? 'bg-amber-100 text-amber-600' : 'bg-emerald-100 text-emerald-600'}">
                    {#if isCheckout}
                      <svg class="w-4 h-4 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>
                    {:else}
                      <svg class="w-4 h-4 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
                    {/if}
                  </div>
                  <div class="flex-1 min-w-0 pointer-events-none">
                    <div class="flex items-center justify-between mb-0.5">
                      <span class="font-bold text-slate-800 text-sm truncate pr-2 group-hover:text-blue-700 transition-colors">
                        {tx.schueler_vorname} {tx.schueler_nachname}
                      </span>
                      <span class="text-[10px] font-bold uppercase text-slate-400 whitespace-nowrap">{formatTime(tx.timestamp)}</span>
                    </div>
                    <p class="text-xs text-slate-500 truncate leading-snug">
                      <span class="font-medium {isCheckout ? 'text-amber-600' : 'text-emerald-600'}">{isCheckout ? 'Ausgeliehen:' : 'Zurückgegeben:'}</span>
                      <span class="italic ml-1">"{tx.buchtitel}"</span>
                    </p>
                  </div>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes slideUp {
    from { opacity: 0; transform: translateY(8px) scale(0.98); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }
  .animate-slide-up {
    animation: slideUp 0.2s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
</style>
