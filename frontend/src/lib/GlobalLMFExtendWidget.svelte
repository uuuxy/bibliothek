<script>
  import { apiFetch } from "./apiFetch.js";

  /** @type {string} */
  let klasse = $state('');
  /** @type {string} */
  let neuesDatum = $state('');
  /** @type {boolean} */
  let isExtending = $state(false);

  async function handleGlobalExtend() {
    if (!klasse.trim() || !neuesDatum) {
      alert("Bitte Klasse und neues Rückgabedatum eingeben.");
      return;
    }

    const confirmed = confirm(`ACHTUNG: Möchten Sie wirklich alle LMF-Ausleihen der Klasse ${klasse} auf den ${neuesDatum} verlängern?\nDies verändert möglicherweise hunderte Datensätze gleichzeitig!`);
    if (!confirmed) return;

    isExtending = true;
    try {
      const res = await apiFetch('/api/ausleihen/global-extend-lmf', {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          klasse: klasse.trim(),
          neues_rueckgabe_datum: neuesDatum
        })
      });

      if (res.ok) {
        const data = await res.json();
        alert(`Erfolgreich: ${data.updated_count} Ausleihen wurden verlängert!`);
        klasse = '';
        neuesDatum = '';
      } else {
        const errText = await res.text();
        alert(`Fehler: ${errText}`);
      }
    } catch (e) {
      console.error(e);
      alert("Netzwerkfehler beim Senden der Anfrage.");
    } finally {
      isExtending = false;
    }
  }
</script>

<div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-5">
  <div>
    <h3 class="text-base font-bold text-slate-900">LMF-Massenverlängerung (Klasse)</h3>
    <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">Verlängert alle aktiven LMF-Ausleihen (Schulbücher) einer bestimmten Klasse auf ein neues fixes Rückgabedatum.</p>
  </div>

  <div class="flex items-end gap-4 flex-wrap">
    <div>
      <label for="extendKlasse" class="text-xs font-semibold text-slate-600 block mb-1">Klasse (z.B. 10b)</label>
      <input id="extendKlasse" type="text" bind:value={klasse} placeholder="10b" class="w-32 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800" />
    </div>
    
    <div>
      <label for="extendDatum" class="text-xs font-semibold text-slate-600 block mb-1">Neues Rückgabedatum</label>
      <input id="extendDatum" type="date" bind:value={neuesDatum} class="w-48 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800" />
    </div>

    <button onclick={handleGlobalExtend} disabled={isExtending || !klasse.trim() || !neuesDatum} class="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed">
      {isExtending ? 'Wird verarbeitet...' : 'Klassen-LMF global verlängern'}
    </button>
  </div>
</div>
