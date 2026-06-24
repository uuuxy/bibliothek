<script>
  import { Search, Plus, Printer } from "lucide-svelte";
  import Button from "../ui/Button.svelte";

  /**
   * @component StudentDirectoryToolbar
   * Flache, vollflächige Aktionsleiste des Schülerverzeichnisses. Fokus liegt auf der
   * Suche (primäre Interaktion); einziger prominenter Button ist "Neuer Schüler".
   *
   * @prop {string} searchQuery - Der aktuelle Suchbegriff (bindable).
   * @prop {string} role - Die Rolle des aktuellen Nutzers (z.B. 'admin', 'mitarbeiter').
   * @prop {number} totalCount - Gesamtanzahl der Schüler.
   * @prop {number} filteredCount - Anzahl der gefilterten Schüler.
   * @prop {() => void} oncreate - Callback wenn "Neuer Schüler" geklickt wird.
   * @prop {() => void} onprintclass - Callback wenn "Klassensatz drucken" geklickt wird.
   */

  /** @type {{ searchQuery?: string, role?: string, totalCount?: number, filteredCount?: number, oncreate?: () => void, onprintclass?: () => void }} */
  let {
    searchQuery = $bindable(""),
    role = "",
    totalCount = 0,
    filteredCount = 0,
    oncreate,
    onprintclass
  } = $props();
</script>

<!-- Flach und edge-to-edge: kein Kachel-Container, nur dezenter Abstand zu den Tabs -->
<div class="flex items-center gap-4 mt-4">
  <div class="relative flex-1 max-w-2xl">
    <Search class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
    <input
      type="text"
      aria-label="Schüler suchen"
      placeholder="Nach Name, Klasse oder Barcode filtern..."
      bind:value={searchQuery}
      class="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-xl text-base text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
    />
  </div>

  {#if role === 'admin' || role === 'mitarbeiter'}
    <Button variant="secondary" onclick={onprintclass} aria-label="Klassensatz drucken">
      <Printer class="w-4 h-4" />
      Klassensatz drucken
    </Button>
    <Button variant="primary" onclick={oncreate} aria-label="Neuen Schüler anlegen">
      <Plus class="w-4 h-4" />
      Neuer Schüler
    </Button>
  {/if}

  <div class="ml-auto shrink-0 text-xs font-semibold text-slate-500">
    Einträge: {filteredCount} / {totalCount}
  </div>
</div>
