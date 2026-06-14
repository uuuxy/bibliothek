<script>
  import { onMount, onDestroy } from "svelte";
  import { appState } from "../inventur/lib/store.svelte.js";
  import { kioskStore } from "./stores/kiosk.svelte.js";
  import KioskReservationModal from "./KioskReservationModal.svelte";
  import KioskChecklistModal from "./KioskChecklistModal.svelte";
  import KioskDamageModal from "./KioskDamageModal.svelte";
  import KioskIdle from "./components/kiosk/KioskIdle.svelte";
  import KioskActiveSession from "./components/kiosk/KioskActiveSession.svelte";

  $effect(() => {
    if (appState.triggerStudentScan) {
      kioskStore.studentInputVal = appState.triggerStudentScan;
      appState.triggerStudentScan = "";
      kioskStore.handleStudentSubmit();
    }
  });

  onMount(() => {
    kioskStore.fetchSettings();
    kioskStore.focusStudentInput();
  });
</script>

<!-- Flash Overlay -->
{#if kioskStore.screenFlash}
  <div class="fixed inset-0 z-50 pointer-events-none transition-colors duration-300
    {kioskStore.screenFlash === 'success' ? 'bg-emerald-500/20' : kioskStore.screenFlash === 'warning' ? 'bg-amber-500/20' : 'bg-rose-500/30'}"></div>
{/if}

<!-- Toast -->
{#if kioskStore.toast}
  <div class="fixed top-8 left-1/2 -translate-x-1/2 z-50 p-4 rounded-xl shadow-xl text-white font-medium
    {kioskStore.toast.type === 'error' ? 'bg-rose-600' : kioskStore.toast.type === 'warning' ? 'bg-amber-500' : 'bg-emerald-600'}">
    {kioskStore.toast.message}
  </div>
{/if}

<div class="w-full space-y-8 relative font-sans">
  {#if !kioskStore.activeStudent}
    <KioskIdle />
  {:else}
    <KioskActiveSession />
  {/if}
</div>

<KioskReservationModal 
  bind:showVormerkenModal={kioskStore.showVormerkenModal} 
  bind:vormerkenQuery={kioskStore.vormerkenQuery} 
  isSearchingVormerken={kioskStore.isSearchingVormerken} 
  vormerkenResults={kioskStore.vormerkenResults} 
  isSubmittingVormerken={kioskStore.isSubmittingVormerken} 
  handleVormerkenSearch={kioskStore.handleVormerkenSearch} 
  handleVormerkenSubmit={kioskStore.handleVormerkenSubmit} 
/>

<KioskChecklistModal 
  bind:showChecklistModal={kioskStore.showChecklistModal} 
  bind:pendingGeraet={kioskStore.pendingGeraet} 
  checklistItems={kioskStore.checklistItems} 
  bind:checkedItems={kioskStore.checkedItems} 
  isSubmittingChecklist={kioskStore.isSubmittingChecklist} 
  handleChecklistSubmit={kioskStore.handleChecklistSubmit} 
/>

<KioskDamageModal 
  bind:returnedBook={kioskStore.returnedBook} 
  bind:showDamageInput={kioskStore.showDamageInput} 
  bind:damageDescription={kioskStore.damageDescription} 
  isSubmittingDamage={kioskStore.isSubmittingDamage} 
  handleDamageOk={kioskStore.handleDamageOk} 
  handleDamageSubmit={kioskStore.handleDamageSubmit} 
/>

{#if kioskStore.vormerkungAlert}
  <div class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4 font-sans">
    <div class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500">
      <div class="text-6xl mb-4">🚨</div>
      <h2 class="text-2xl font-extrabold text-rose-700 mb-2">Achtung! Vorgemerkt!</h2>
      <p class="text-slate-700 mb-2">Dieses Medium wurde reserviert.</p>
      <p class="font-bold text-slate-900 mb-6">Achtung: Exemplar nicht ins Regal stellen!</p>
      {#if kioskStore.vormerkungAlert.titel}
        <p class="text-sm text-slate-500 mb-2">„{kioskStore.vormerkungAlert.titel}"</p>
      {/if}
      {#if kioskStore.vormerkungAlert.user}
        <p class="text-md font-bold text-rose-800 bg-rose-100 py-3 px-4 rounded-xl border border-rose-200 mb-6">Vorgemerkt für: {kioskStore.vormerkungAlert.user}</p>
      {/if}
      <button onclick={() => { kioskStore.vormerkungAlert = null; }}
        class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
        Verstanden
      </button>
    </div>
  </div>
{/if}
