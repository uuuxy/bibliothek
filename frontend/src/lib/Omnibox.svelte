<script>
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";
  import OfflineQueueBanner from "./OfflineQueueBanner.svelte";
  import CameraScanner from "./CameraScanner.svelte";
  import OmniboxInput from "./components/OmniboxInput.svelte";
  import OmniboxResults from "./components/OmniboxResults.svelte";
  import OmniboxTeacherCard from "./OmniboxTeacherCard.svelte";
  import { loadQueue } from "./offlineQueue.js";
  import { omniboxStore } from "./stores/omnibox.svelte.js";
  import { appState } from "../inventur/lib/store.svelte.js";
  import { apiClient } from "./apiFetch.js";

  let { onSelectBook } = $props();

  let studentProfileComponent = $state(/** @type {any} */ (null));

  $effect(() => {
    if (appState.triggerStudentScan) {
      omniboxStore.queryVal = appState.triggerStudentScan;
      appState.triggerStudentScan = "";
      omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
    }
  });

  onMount(() => {
    // SSE für Live-Aktualisierung des Schülerprofils
    const source = new EventSource("/events");
    source.addEventListener("action", (e) => {
      try {
        const actionData = JSON.parse(/** @type {any} */ (e).data);
        if (omniboxStore.activeStudent && actionData.student_id === omniboxStore.activeStudent.id) {
          studentProfileComponent?.reloadProfile();
        }
      } catch (err) {
        console.error("SSE Parsing-Fehler in der Omnibox:", err);
      }
    });

    // Offline / Online Erkennung is now handled globally in offlineSync.svelte.js

    return () => {
      source.close();
    };
  });

  $effect(() => {
    if (!omniboxStore.isActive && !omniboxStore.isDropdownOpen && !omniboxStore.showCamera) {
      setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
    }
  });

  $effect(() => {
    /** @param {KeyboardEvent} e */
    function handleKeyDown(e) {
      if (e.key === "Escape") {
        omniboxStore.queryVal = "";
        omniboxStore.activeStudent = null;
        omniboxStore.activeTeacher = null;
        omniboxStore.lastFremdrueckgabe = null;
        omniboxStore.isDropdownOpen = false;
        if (omniboxStore.showCamera) {
            omniboxStore.showCamera = false;
            if (omniboxStore.cameraScanner) {
                try { omniboxStore.cameraScanner.stop(); } catch {}
                try { omniboxStore.cameraScanner.clear(); } catch {}
                omniboxStore.cameraScanner = null;
            }
            setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  });

  // HTML5 Kamera-Scanner (Mobile)
  async function startCamera() {
    omniboxStore.showCamera = true;
    await new Promise(r => setTimeout(r, 80));
    try {
      const { Html5Qrcode } = await import("html5-qrcode");
      omniboxStore.cameraScanner = new Html5Qrcode("camera-scan-region");
      await omniboxStore.cameraScanner.start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 260, height: 120 } },
        (/** @type {string} */ decodedText) => {
          omniboxStore.queryVal = decodedText.trim();
          stopCamera();
          omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
        },
        () => {}
      );
    } catch {
      omniboxStore.showCamera = false;
      omniboxStore.showToast("Kamera konnte nicht gestartet werden", "error");
    }
  }

  async function stopCamera() {
    omniboxStore.showCamera = false;
    if (omniboxStore.cameraScanner) {
      try { await omniboxStore.cameraScanner.stop(); } catch {}
      try { omniboxStore.cameraScanner.clear(); } catch {}
      omniboxStore.cameraScanner = null;
    }
    setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
  }
</script>

<!-- ── Screen-edge flash overlay (Apple-style full-border glow) ── -->
{#if omniboxStore.screenFlash}
  <div class="screen-flash screen-flash--{omniboxStore.screenFlash}" aria-hidden="true"></div>
{/if}

<!-- ── Offline / Queue banner was replaced by global OfflineIndicator ── -->

<div class="w-full mx-auto transition-all duration-500 ease-in-out {omniboxStore.isActive ? 'w-full pt-4 justify-start' : 'max-w-2xl min-h-[60vh] justify-center'} flex flex-col items-center space-y-6">
  <div class="w-full transition-all duration-500 {omniboxStore.isActive ? 'sticky -top-4 z-30 bg-slate-50/95 backdrop-blur-md py-4' : ''}">
    <form onsubmit={(e) => omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())} class="w-full relative py-5 px-8 rounded-3xl border shadow-2xl no-print transition-all duration-500 focus-within:border-blue-500 focus-within:ring-4 focus-within:ring-blue-50 {omniboxStore.isActive ? 'scale-100' : 'scale-105'} {(omniboxStore.isShaking || omniboxStore.scanError) ? 'animate-shake' : ''} {omniboxStore.scanError ? 'ring-2 ring-red-500 bg-red-50 border-red-500' : 'bg-white border-slate-200'} {omniboxStore.flashBorder === 'green' ? 'ring-4 ring-emerald-500/10 border-emerald-400' : omniboxStore.flashBorder === 'orange' ? 'ring-4 ring-amber-500/10 border-amber-400' : omniboxStore.flashBorder === 'red' && !omniboxStore.scanError ? 'ring-4 ring-red-500/30 border-red-500' : ''}">
      <OmniboxInput 
        bind:queryVal={omniboxStore.queryVal}
        isDropdownOpen={omniboxStore.isDropdownOpen}
        totalDropdownItems={omniboxStore.totalDropdownItems}
        isActive={omniboxStore.isActive}
        showCamera={omniboxStore.showCamera}
        onInput={omniboxStore.handleInput}
        onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
        onIndexChange={(idx) => omniboxStore.selectedDropdownIndex = idx}
        onEscape={() => omniboxStore.isDropdownOpen = false}
        onToggleCamera={omniboxStore.showCamera ? stopCamera : startCamera}
      />

      {#if omniboxStore.isDropdownOpen && omniboxStore.totalDropdownItems > 0}
        <OmniboxResults 
          unifiedSearchResults={omniboxStore.unifiedSearchResults} 
          selectedDropdownIndex={omniboxStore.selectedDropdownIndex} 
          onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)} 
        />
      {/if}
    </form>

    {#if omniboxStore.errorMessage}
      <div class="mt-3 p-3 bg-red-600 text-white font-bold rounded-xl shadow-lg text-center animate-slide-down">
        {omniboxStore.errorMessage}
      </div>
    {/if}
  </div>

  <!-- HTML5 Kamera-Scanner (Mobile) -->
  {#if omniboxStore.showCamera}
    <CameraScanner {stopCamera} bind:queryVal={omniboxStore.queryVal} submitAction={(e) => omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())} />
  {/if}

  {#if omniboxStore.activeStudent}
    {#if omniboxStore.lastFremdrueckgabe}
      <div class="w-full max-w-xl p-3 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-medium flex items-center space-x-2 animate-slide-up no-print mb-2">
        <span>⚠️</span>
        <span><strong>Fremdrückgabe:</strong> Buch wurde von <strong>{omniboxStore.lastFremdrueckgabe.vorbesitzerName}</strong> zurückgegeben und für {omniboxStore.activeStudent.vorname} verbucht.</span>
      </div>
    {/if}
    <StudentProfile bind:this={studentProfileComponent} student={omniboxStore.activeStudent} onDeselect={() => { omniboxStore.activeStudent = null; omniboxStore.lastFremdrueckgabe = null; }} onReturnClick={(barcode) => { omniboxStore.queryVal = barcode; omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile()); }} />
  {:else if omniboxStore.activeTeacher}
    <OmniboxTeacherCard teacher={omniboxStore.activeTeacher} onDeselect={() => { omniboxStore.activeTeacher = null; omniboxStore.lastFremdrueckgabe = null; }} />
  {/if}
</div>

<!-- Toast notifications -->
<div class="fixed top-24 left-1/2 -translate-x-1/2 w-full max-w-lg z-50 space-y-3 px-4 pointer-events-none">
  {#if omniboxStore.toast}
    <div class="p-4 rounded-xl shadow-xl flex items-center space-x-3 backdrop-blur-md animate-slide-down pointer-events-auto border
      {omniboxStore.toast.type === 'success' ? 'bg-emerald-50 border-emerald-100/50 text-emerald-700'
      : omniboxStore.toast.type === 'warning' ? 'bg-amber-50 border-amber-100/50 text-amber-700'
      : 'bg-red-600 border-red-700 text-white shadow-red-500/30'}">
      <span class="text-sm font-semibold">{omniboxStore.toast.message}</span>
    </div>
  {/if}
</div>

{#if omniboxStore.vormerkungAlert}
  <div class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4">
    <div class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500">
      <div class="text-6xl mb-4">🚨</div>
      <h2 class="text-2xl font-extrabold text-rose-700 mb-2">Achtung! Vorgemerkt!</h2>
      <p class="text-slate-700 mb-2">Dieses Medium wurde reserviert.</p>
      <p class="font-bold text-slate-900 mb-6">Achtung: Exemplar nicht ins Regal stellen!</p>
      {#if omniboxStore.vormerkungAlert.titel}
        <p class="text-sm text-slate-500 mb-2">„{omniboxStore.vormerkungAlert.titel}"</p>
      {/if}
      {#if omniboxStore.vormerkungAlert.user}
        <p class="text-md font-bold text-rose-800 bg-rose-100 py-3 px-4 rounded-xl border border-rose-200 mb-6">Vorgemerkt für: {omniboxStore.vormerkungAlert.user}</p>
      {/if}
      <button onclick={() => { omniboxStore.vormerkungAlert = null; }}
        class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
        Verstanden
      </button>
    </div>
  </div>
{/if}

{#if omniboxStore.blockAlert}
  <div class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4">
    <div class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500">
      <div class="text-6xl mb-4">⛔️</div>
      <h2 class="text-2xl font-extrabold text-rose-700 mb-2">Ausleihe blockiert</h2>
      <p class="text-slate-700 font-medium mb-6">{omniboxStore.blockAlert.message}</p>
      
      <div class="space-y-3">
        <button onclick={() => {
          const q = omniboxStore.blockAlert?.query;
          if (!q) return;
          omniboxStore.blockAlert = null;
          omniboxStore.queryVal = q;
          omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile(), true);
        }}
          class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
          Einmalig ignorieren (Override)
        </button>
        
        {#if omniboxStore.activeStudent?.is_manually_blocked}
          <button onclick={async () => {
            try {
              const res = await apiClient.post(`/api/schueler/${omniboxStore.activeStudent.id}/update`, {
                is_manually_blocked: false,
                block_reason: ""
              });
              if (res.ok) {
                const q = omniboxStore.blockAlert?.query;
                omniboxStore.blockAlert = null;
                if (q) omniboxStore.queryVal = q;
                omniboxStore.activeStudent.is_manually_blocked = false;
                omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
              }
            } catch(e) {
              console.error(e);
            }
          }}
            class="px-8 py-3 bg-white border-2 border-slate-200 hover:bg-slate-50 text-slate-700 font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
            Sperre dauerhaft aufheben
          </button>
        {/if}

        <button onclick={() => { omniboxStore.blockAlert = null; }}
          class="px-8 py-3 bg-transparent text-slate-500 hover:text-slate-700 font-bold rounded-xl text-sm transition-colors cursor-pointer w-full mt-2">
          Abbrechen
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* ── Screen-edge flash overlay ─────────────────────────────── */
  .screen-flash {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 9999;
    border-radius: 0;
    animation: screen-flash-fade 300ms ease-out forwards;
  }
  .screen-flash--success {
    box-shadow:
      inset 0 0 0 6px rgba(16, 185, 129, 0.85),  /* emerald-500 */
      inset 0 0 60px 10px rgba(16, 185, 129, 0.18);
  }
  .screen-flash--error {
    box-shadow:
      inset 0 0 0 6px rgba(239, 68, 68, 0.85),   /* red-500 */
      inset 0 0 60px 10px rgba(239, 68, 68, 0.18);
  }
  .screen-flash--warning {
    box-shadow:
      inset 0 0 0 6px rgba(245, 158, 11, 0.85),  /* amber-500 */
      inset 0 0 60px 10px rgba(245, 158, 11, 0.18);
  }
  @keyframes screen-flash-fade {
    0%   { opacity: 1; }
    60%  { opacity: 0.9; }
    100% { opacity: 0; }
  }

  /* ── Shake animation ─────────────────────────────────────── */
  @keyframes shake {
    0%, 100% { transform: translate(0, 0) scale(1.05); }
    15%, 45%, 75% { transform: translate(-8px, 0) scale(1.05); }
    30%, 60% { transform: translate(8px, 0) scale(1.05); }
  }
  @keyframes activeShake {
    0%, 100% { transform: translate(0, 0) scale(1); }
    15%, 45%, 75% { transform: translate(-8px, 0) scale(1); }
    30%, 60% { transform: translate(8px, 0) scale(1); }
  }
  .animate-shake {
    animation: shake 0.4s cubic-bezier(.36,.07,.19,.97) both;
  }
  :global(.pt-4) .animate-shake {
    animation: activeShake 0.4s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
