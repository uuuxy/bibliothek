<script>
  import { kioskStore } from "../../stores/kiosk.svelte.js";
  import StudentProfile from "../../StudentProfile.svelte";
  import BorrowedBooksList from "../../BorrowedBooksList.svelte";

  // ── Svelte Action: Autofocus-Lock ───────────────────────────────────
  function keepFocus(node, options) {
    let active = options.active;

    function enforceFocus() {
      if (active && !node.disabled) {
        node.focus();
      }
    }

    requestAnimationFrame(enforceFocus);

    function onWindowClick(e) {
      if (!active || node.disabled) return;
      const isInteractive = e.target.closest('button, a, input, select, textarea, [role="button"], dialog, .modal');
      if (!isInteractive) {
        enforceFocus();
      }
    }

    function onBlur() {
      if (!active || node.disabled) return;
      setTimeout(() => {
        if (active && !node.disabled && (document.activeElement === document.body || document.activeElement === null)) {
          enforceFocus();
        }
      }, 50);
    }

    window.addEventListener('click', onWindowClick, { capture: true });
    node.addEventListener('blur', onBlur);

    return {
      update(newOptions) {
        active = newOptions.active;
        if (active) {
          requestAnimationFrame(enforceFocus);
        }
      },
      destroy() {
        window.removeEventListener('click', onWindowClick, { capture: true });
        node.removeEventListener('blur', onBlur);
      }
    };
  }
</script>

{#if !kioskStore.isStudentBlocked || kioskStore.overrideBlock}
  <div class="relative w-full mb-8 print:hidden {kioskStore.isShaking ? 'animate-shake' : ''}">
    <form onsubmit={(e) => { e.preventDefault(); kioskStore.handleBookSubmit(); }} class="relative w-full">
      <svg class="w-6 h-6 absolute left-5 top-1/2 -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
      <input type="text" id="kiosk-book-input" bind:value={kioskStore.bookInputVal} disabled={kioskStore.isScanningBook}
             use:keepFocus={{ active: !kioskStore.isScanningBook && !kioskStore.showVormerkenModal && !kioskStore.showChecklistModal && !kioskStore.showDamageInput }}
             placeholder="Buch-Barcode (B-) scannen..." autocomplete="off"
             class="w-full bg-white shadow-xl border-0 ring-1 ring-slate-200 focus:ring-4 focus:ring-emerald-500/20 rounded-full pl-14 pr-16 py-5 text-xl font-medium outline-none transition-all placeholder:text-slate-400 disabled:opacity-50 disabled:cursor-not-allowed" />
      <button type="button" onclick={() => kioskStore.showVormerkenModal = true} class="absolute right-4 top-1/2 -translate-y-1/2 p-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 rounded-full transition-colors cursor-pointer" title="Medium vormerken">
        <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
      </button>
    </form>
  </div>
{/if}

<!-- Ausleih-Sperre Meldung -->
{#if kioskStore.isStudentBlocked && !kioskStore.overrideBlock}
  <div class="bg-rose-100 border border-rose-200 text-rose-800 p-5 rounded-2xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8 print:hidden shadow-sm">
    <div class="flex items-start gap-3">
      <svg class="w-6 h-6 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
      <div>
        <h3 class="font-bold text-lg">Ausleihe blockiert</h3>
        {#if kioskStore.activeStudent?.is_manually_blocked}
          <p class="text-sm mt-1 font-semibold">Manuelle Sperre: <span class="font-normal">{kioskStore.activeStudent.block_reason || 'Ohne Grund'}</span></p>
        {:else}
          <p class="text-sm mt-1">Dieser Schüler hat überfällige Mahnungen offen und darf keine neuen Medien ausleihen.</p>
        {/if}
      </div>
    </div>
    <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 flex-wrap">
      <button onclick={() => kioskStore.overrideBlock = true} class="flex-1 sm:flex-none px-4 py-2.5 bg-white hover:bg-rose-50 text-rose-700 rounded-xl text-sm font-bold transition-colors shadow-sm cursor-pointer border border-rose-200">
        Einmalig ignorieren (Override)
      </button>
      {#if kioskStore.activeStudent?.is_manually_blocked}
      <button onclick={() => kioskStore.clearManualBlock()} class="flex-1 sm:flex-none px-4 py-2.5 bg-rose-600 hover:bg-rose-700 text-white rounded-xl text-sm font-bold transition-colors shadow-sm cursor-pointer">
        Sperre dauerhaft aufheben
      </button>
      {/if}
    </div>
  </div>
{/if}

<StudentProfile 
  student={kioskStore.activeStudent} 
  onDeselect={kioskStore.clearSession} 
  onReturnClick={(barcode) => {
    kioskStore.bookInputVal = barcode;
    kioskStore.handleBookSubmit();
  }} 
>
  {#snippet leftActions()}
    <button class="mt-4 w-full py-3 bg-slate-200 hover:bg-slate-300 text-slate-700 rounded-2xl font-bold transition-colors cursor-pointer"
            onclick={kioskStore.clearSession}>
      Sitzung beenden (Anderen Schüler scannen)
    </button>
  {/snippet}

  {#snippet rightTop()}
    {#if !kioskStore.isStudentBlocked && kioskStore.scannedBooks.length > 0}
      <div class="space-y-6 mb-6">
        <!-- Scanned Books List -->
        <div class="bg-white p-6 rounded-2xl shadow-xl border border-slate-100">
          <h4 class="font-bold text-slate-500 text-sm uppercase tracking-wider mb-4">Scans in dieser Sitzung</h4>
          <BorrowedBooksList books={kioskStore.scannedBooks} mode="scans" />
        </div>
      </div>
    {/if}
  {/snippet}
</StudentProfile>

<style>
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-8px); }
    75% { transform: translateX(8px); }
  }
  .animate-shake {
    animation: shake 0.3s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
