<script>
  import Button from "./components/ui/Button.svelte";
  import { Printer, FileText, Lock, Unlock, AlertTriangle } from "lucide-svelte";

  /**
   * @typedef {Object} Props
   * @property {any} profile
   * @property {boolean} kontoauszugPdfLoading
   * @property {boolean} rechnungPdfLoading
   * @property {() => void} downloadKontoauszugPDF
   * @property {() => void} downloadRechnungPDF
   * @property {() => void} showLockModal
   */
  /** @type {Props} */
  let { 
    profile, 
    kontoauszugPdfLoading, 
    rechnungPdfLoading, 
    downloadKontoauszugPDF, 
    downloadRechnungPDF, 
    showLockModal 
  } = $props();
</script>

<!-- Aktionen / Dokumente -->
<div class="bg-slate-50 border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col gap-3">
  <h4 class="text-xs font-bold text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
    <FileText class="w-3.5 h-3.5" />
    Dokumente & Aktionen
  </h4>
  {#snippet spinner()}
    <div class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"></div>
  {/snippet}

  <div class="flex flex-wrap gap-3 items-center">
    <!-- Druck- & Export-Aktionen: linksbündig gruppiert -->
    <Button variant="secondary" onclick={downloadKontoauszugPDF} disabled={kontoauszugPdfLoading || !(profile.entliehene_buecher?.length > 0)}>
      {#if kontoauszugPdfLoading}{@render spinner()}{:else}<Printer class="w-4 h-4 text-blue-600" />{/if}
      Kontoauszug
    </Button>

    <Button variant="secondary" onclick={downloadRechnungPDF} disabled={rechnungPdfLoading || !profile.has_open_damages} title={!profile.has_open_damages ? 'Keine offenen Forderungen' : 'Ersatzforderung drucken'}>
      {#if rechnungPdfLoading}{@render spinner()}{:else}<AlertTriangle class="w-4 h-4 text-rose-600" />{/if}
      Forderung
    </Button>

    <Button variant="secondary" onclick={() => window.print()} disabled={!(profile.entliehene_buecher?.length > 0)} title={!(profile.entliehene_buecher?.length > 0) ? 'Keine offenen Ausleihen' : 'Druckansicht der Ausleihen'}>
      <Printer class="w-4 h-4 text-slate-500" />
      Ausleihen-Liste
    </Button>

    <!-- Sperr-Aktion: optisch getrennt ganz nach rechts -->
    <Button class="ml-auto" variant={profile.is_manually_blocked ? 'success' : 'danger'} onclick={showLockModal}>
      {#if profile.is_manually_blocked}
        <Unlock class="w-4 h-4" /> Sperre aufheben
      {:else}
        <Lock class="w-4 h-4" /> Schüler sperren
      {/if}
    </Button>
  </div>
</div>
