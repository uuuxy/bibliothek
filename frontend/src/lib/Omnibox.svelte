<script>
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";
  import CameraScanner from "./CameraScanner.svelte";
  import OmniboxInput from "./components/OmniboxInput.svelte";
  import OmniboxResults from "./components/OmniboxResults.svelte";
  import OmniboxTeacherCard from "./OmniboxTeacherCard.svelte";
  import OmniboxVormerkungAlert from "./components/OmniboxVormerkungAlert.svelte";
  import OmniboxBlockAlert from "./components/OmniboxBlockAlert.svelte";
  import OmniboxScreenFlash from "./components/OmniboxScreenFlash.svelte";
  import { omniboxStore } from "./stores/omnibox.svelte.js";
  import { appState } from "../inventur/lib/store.svelte.js";

  let { onSelectBook, role = "" } = $props();

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

<OmniboxScreenFlash />

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
        <span><strong>Fremdrückgabe:</strong> Buch war auf <strong>{omniboxStore.lastFremdrueckgabe.vorbesitzerName}</strong> verbucht und wurde dort zurückgegeben — <strong>nicht</strong> auf {omniboxStore.activeStudent.vorname} gebucht. Erneut scannen, um es auszuleihen.</span>
      </div>
    {/if}
    <StudentProfile bind:this={studentProfileComponent} student={omniboxStore.activeStudent} {role} onDeselect={() => { omniboxStore.activeStudent = null; omniboxStore.lastFremdrueckgabe = null; }} onReturnClick={(barcode) => { omniboxStore.queryVal = barcode; omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile()); }} />
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

<OmniboxVormerkungAlert />

<OmniboxBlockAlert onReload={() => studentProfileComponent?.reloadProfile()} />

<style>
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
