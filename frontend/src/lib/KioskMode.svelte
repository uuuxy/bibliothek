<script>
  import { onMount } from "svelte";
  import { appState } from "../inventur/lib/store.svelte.js";
  import { kioskStore } from "./stores/kiosk.svelte.js";
  import { flushOfflineQueue } from "./offlineQueue.js";
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
    
    const onlineHandler = async () => {
      await flushOfflineQueue((msg, type) => kioskStore.triggerFlash(/** @type {"success"|"error"|"warning"} */ (type), msg));
    };
    window.addEventListener("online", onlineHandler);
    return () => window.removeEventListener("online", onlineHandler);
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
