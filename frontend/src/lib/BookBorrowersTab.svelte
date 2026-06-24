<script>
  import { appState } from "../inventur/lib/store.svelte.js";

  /** @type {{ borrowers: any[], book: any, onBack: () => void }} */
  let { borrowers, book, onBack } = $props();

  let filterKlasse = $state("Alle");
  let filterName = $state("");

  let availableKlassen = $derived(
    ["Alle", ...Array.from(new Set(borrowers.map((b) => b.klasse || "Unbekannt"))).sort()]
  );

  let filteredBorrowers = $derived(
    borrowers.filter((b) => {
      const matchKlasse = filterKlasse === "Alle" || (b.klasse || "Unbekannt") === filterKlasse;
      const matchName =
        filterName === "" ||
        `${b.schueler_name} ${b.schueler_nachname}`.toLowerCase().includes(filterName.toLowerCase());
      return matchKlasse && matchName;
    })
  );

  /** @param {string} d */
  function fmtDate(d) {
    if (!d) return "-";
    try { return new Date(d).toLocaleDateString("de-DE"); } catch { return d; }
  }

  function printAusleiher() {
    const printWindow = window.open('', '_blank', 'width=800,height=600');
    if (!printWindow) {
      alert("Bitte erlaube Popups, um die Liste zu drucken.");
      return;
    }
    
    const printDate = new Date().toLocaleDateString("de-DE");
    let html = `
      <!DOCTYPE html>
      <html>
      <head>
        <title>Mahnliste: ${book?.title || 'Buch'}</title>
        <style>
          body { font-family: system-ui, -apple-system, sans-serif; padding: 2rem; color: #1e293b; }
          h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
          p.meta { margin: 0 0 1.5rem 0; color: #64748b; font-size: 0.875rem; }
          table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
          th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #e2e8f0; }
          th { background: #f8fafc; font-weight: 600; font-size: 0.875rem; color: #475569; }
          .overdue { color: #e11d48; font-weight: bold; }
          @media print { @page { margin: 1cm; } }
        </style>
      </head>
      <body>
        <h1>Ausleiher-Liste: ${book?.title || 'Buch'}</h1>
        <p class="meta">Erstellt am: ${printDate} | Filter: Klasse ${filterKlasse}</p>
        <table>
          <thead>
            <tr>
              <th>Schüler/in</th>
              <th>Klasse</th>
              <th>Exemplar</th>
              <th>Ausgeliehen am</th>
              <th>Rückgabe bis</th>
            </tr>
          </thead>
          <tbody>
    `;

    for (const b of filteredBorrowers) {
      const isOverdue = new Date(b.rueckgabe_frist) < new Date();
      html += `
        <tr>
          <td>${b.schueler_name} ${b.schueler_nachname}</td>
          <td>${b.klasse || '-'}</td>
          <td style="font-family: monospace; font-size: 0.875rem;">${b.exemplar_barcode}</td>
          <td>${fmtDate(b.ausgeliehen_am)}</td>
          <td class="${isOverdue ? 'overdue' : ''}">${fmtDate(b.rueckgabe_frist)}</td>
        </tr>
      `;
    }

    html += `
          </tbody>
        </table>
        \x3Cscript>
          window.onload = () => { setTimeout(() => window.print(), 200); }
        \x3C/script>
      </body>
      </html>
    `;
    
    printWindow.document.open();
    printWindow.document.write(html);
    printWindow.document.close();
  }
</script>

{#if borrowers.length === 0}
  <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
    <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
    <p class="font-semibold text-sm">Aktuell niemand hat dieses Buch ausgeliehen.</p>
  </div>
{:else}
  <!-- Filters -->
  <div class="flex gap-3 mb-4">
    <select bind:value={filterKlasse} aria-label="Nach Klasse filtern" class="px-3 py-2 bg-white border border-slate-200 rounded-xl text-sm font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/30 cursor-pointer">
      {#each availableKlassen as k}<option value={k}>{k}</option>{/each}
    </select>
    <div class="relative flex-1 max-w-xs">
      <svg aria-hidden="true" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
      <input type="text" bind:value={filterName} aria-label="Nach Name filtern" placeholder="Name filtern..." class="w-full pl-9 pr-3 py-2 bg-white border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/30 placeholder:text-slate-400" />
    </div>
    {#if filteredBorrowers.length !== borrowers.length}
      <span class="text-xs text-slate-400 self-center">{filteredBorrowers.length} von {borrowers.length}</span>
    {/if}
    <div class="flex-1"></div>
    <button onclick={printAusleiher} class="flex items-center gap-1.5 px-3 py-2 bg-white border border-slate-200 rounded-xl text-sm font-semibold text-slate-600 hover:text-slate-800 hover:bg-slate-50 transition-colors shadow-sm cursor-pointer">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" /></svg>
      Mahnliste drucken
    </button>
  </div>

  <!-- List -->
  <div class="w-full">
    <ul class="divide-y divide-slate-50">
      {#each filteredBorrowers as b}
        <li class="px-5 py-3.5 hover:bg-slate-50 transition-colors flex items-center justify-between group">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold text-xs shrink-0">
              {b.schueler_name?.[0] ?? ""}{b.schueler_nachname?.[0] ?? ""}
            </div>
            <div class="min-w-0">
              <button
                onclick={() => { appState.triggerStudentScan = b.schueler_barcode; onBack(); }}
                class="text-sm font-semibold text-slate-800 hover:text-indigo-600 text-left cursor-pointer truncate block"
              >
                {b.schueler_name} {b.schueler_nachname}
                <span class="text-xs font-normal text-slate-400 ml-1">({b.klasse || "Unbekannt"})</span>
              </button>
              <p class="text-xs text-slate-400 font-mono mt-0.5">Exemplar: {b.exemplar_barcode}</p>
            </div>
          </div>
          <div class="text-right shrink-0 ml-4 flex gap-6 items-center">
            <div class="text-right hidden sm:block">
              <p class="text-[10px] font-medium text-slate-400">Ausgeliehen</p>
              <p class="text-sm font-semibold text-slate-600">
                {fmtDate(b.ausgeliehen_am)}
              </p>
            </div>
            <div class="text-right">
              <p class="text-[10px] font-medium text-slate-400">Rückgabe bis</p>
              <p class="text-sm font-bold {new Date(b.rueckgabe_frist) < new Date() ? 'text-rose-600' : 'text-slate-700'}">
                {fmtDate(b.rueckgabe_frist)}
              </p>
            </div>
          </div>
        </li>
      {/each}
    </ul>
    {#if filteredBorrowers.length === 0}
      <div class="py-8 text-center text-sm text-slate-400">Keine Ausleihen entsprechen dem Filter.</div>
    {/if}
  </div>
{/if}
