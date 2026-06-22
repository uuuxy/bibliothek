<script>
  import { Search, Plus, Upload, Printer } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  /**
   * @component StudentDirectoryToolbar
   * Toolbar für das Schülerverzeichnis mit Suche, "Neuer Schüler", LUSD-Import und Druck-Optionen.
   *
   * @prop {string} searchQuery - Der aktuelle Suchbegriff (bindable).
   * @prop {string} role - Die Rolle des aktuellen Nutzers (z.B. 'admin', 'mitarbeiter').
   * @prop {number} totalCount - Gesamtanzahl der Schüler.
   * @prop {number} filteredCount - Anzahl der gefilterten Schüler.
   * @prop {() => void} oncreate - Callback wenn "Neuer Schüler" geklickt wird.
   * @prop {(file: File) => void} onimport - Callback wenn eine LUSD-Datei ausgewählt wird.
   * @prop {() => void} onprint - Callback wenn "Klassensatz drucken" geklickt wird.
   */
  let {
    searchQuery = $bindable(""),
    role = "",
    totalCount = 0,
    filteredCount = 0,
    oncreate,
    onimport,
    onprint
  } = $props();

  /** @type {HTMLInputElement | null} */
  let fileInputEl = $state(null);

  function triggerImportPicker() {
    fileInputEl?.click();
  }

  /** @param {Event} event */
  function handleFileChange(event) {
    const target = /** @type {HTMLInputElement} */ (event.target);
    const file = target.files?.[0];
    if (file && onimport) {
      onimport(file);
    }
    target.value = ""; // Reset für erneute Auswahl
  }
</script>

<div class="flex items-center gap-4 bg-white p-4 rounded-2xl border border-slate-100 shadow-xs justify-between">
  <div class="flex flex-1 items-center gap-4">
    <div class="relative w-full max-w-md">
      <Search class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
      <input 
        type="text" 
        aria-label="Schüler suchen" 
        placeholder="Nach Name, Klasse oder Barcode filtern..." 
        bind:value={searchQuery} 
        class="w-full pl-10 pr-4 py-2 bg-slate-55 border border-slate-200 rounded-xl text-base text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" 
      />
    </div>

    {#if role === 'admin' || role === 'mitarbeiter'}
      <Button variant="primary" onclick={oncreate} aria-label="Neuen Schüler anlegen">
        <Plus class="w-4 h-4" />
        Neuer Schüler
      </Button>
    {/if}

    {#if role === 'admin'}
      <input type="file" accept=".csv" bind:this={fileInputEl} onchange={handleFileChange} class="hidden" aria-label="LUSD Datei auswählen" />
      <Button variant="secondary" onclick={triggerImportPicker} aria-label="LUSD-Import starten">
        <Upload class="w-4 h-4" />
        LUSD Import (CSV)
      </Button>
    {/if}

    {#if role === 'admin' || role === 'mitarbeiter'}
      <Button variant="secondary" onclick={onprint} aria-label="Klassensatz drucken">
        <Printer class="w-4 h-4" />
        Klassensatz drucken
      </Button>
    {/if}
  </div>
  
  <div class="text-xs font-semibold text-slate-500">
    Einträge: {filteredCount} / {totalCount}
  </div>
</div>
