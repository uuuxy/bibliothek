<script>
  /** @type {{ id: string, name: string }[]} */
  let { suppliers = [] } = $props();

  // Report-Typ
  /** @type {"monat" | "jahr" | "lieferant"} */
  let typ = $state("monat");

  // Monatsbericht
  const now = new Date();
  let monatJahr = $state(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`);

  // Jahresbericht
  let jahr = $state(String(now.getFullYear()));

  // Lieferantenabrechnung
  let lieferantId = $state(suppliers[0]?.id ?? "");
  let vonDatum = $state(firstOfMonth(now));
  let bisDatum = $state(today());

  function today() {
    return now.toISOString().slice(0, 10);
  }
  function firstOfMonth(d) {
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-01`;
  }
  function lastOfMonth(yyyyMM) {
    const [y, m] = yyyyMM.split("-").map(Number);
    return new Date(y, m, 0).toISOString().slice(0, 10);
  }

  function buildURL() {
    const base = "/api/bestellhistorie/bericht";
    if (typ === "monat") {
      const von = monatJahr + "-01";
      const bis = lastOfMonth(monatJahr);
      const [y, m] = monatJahr.split("-");
      const titel = `Monatsbericht ${monthLabel(Number(m))} ${y}`;
      return `${base}?von=${von}&bis=${bis}&titel=${encodeURIComponent(titel)}`;
    }
    if (typ === "jahr") {
      const von = `${jahr}-01-01`;
      const bis = `${jahr}-12-31`;
      return `${base}?von=${von}&bis=${bis}&jahresansicht=true&titel=${encodeURIComponent("Jahresbericht " + jahr)}`;
    }
    // lieferant
    const name = suppliers.find((s) => s.id === lieferantId)?.name ?? "Lieferant";
    return `${base}?von=${vonDatum}&bis=${bisDatum}&lieferant_id=${lieferantId}&titel=${encodeURIComponent("Lieferantenabrechnung: " + name)}`;
  }

  const monthLabels = ["Januar","Februar","März","April","Mai","Juni","Juli","August","September","Oktober","November","Dezember"];
  function monthLabel(n) { return monthLabels[n - 1] ?? ""; }

  // Jahres-Optionen: aktuelles Jahr + 4 zurück
  const yearOptions = Array.from({ length: 5 }, (_, i) => String(now.getFullYear() - i));

  $effect(() => {
    // Wenn Lieferanten geladen werden, Vorauswahl setzen
    if (suppliers.length > 0 && !lieferantId) {
      lieferantId = suppliers[0].id;
    }
  });
</script>

<div class="space-y-8 max-w-xl">
  <div class="border-b border-slate-200 pb-4">
    <h2 class="text-base font-bold text-slate-800">Berichte & Auswertungen</h2>
    <p class="text-sm text-slate-500 mt-0.5">PDF-Ausdruck für Abrechnung und Schulleitung</p>
  </div>

  <!-- Berichtstyp -->
  <div class="space-y-3">
    <p class="text-sm font-semibold text-slate-600 uppercase tracking-wide">Berichtstyp</p>
    <div class="flex flex-col gap-2">
      {#each [
        { value: "monat", label: "Monatsbericht", desc: "Alle Bestellungen eines Monats mit Titeln und Summe" },
        { value: "jahr",  label: "Jahresbericht",  desc: "Monatliche Übersicht + Aufteilung nach Lieferant" },
        { value: "lieferant", label: "Lieferantenabrechnung", desc: "Alle Bestellungen bei einem Lieferanten in einem Zeitraum" },
      ] as opt}
        <label class="flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition-colors {typ === opt.value ? 'border-blue-400 bg-blue-50' : 'border-slate-200 hover:border-slate-300'}">
          <input type="radio" bind:group={typ} value={opt.value} class="mt-0.5 accent-blue-600" />
          <div>
            <div class="font-bold text-sm text-slate-800">{opt.label}</div>
            <div class="text-xs text-slate-500">{opt.desc}</div>
          </div>
        </label>
      {/each}
    </div>
  </div>

  <!-- Parameter -->
  <div class="space-y-4">
    <p class="text-sm font-semibold text-slate-600 uppercase tracking-wide">Parameter</p>

    {#if typ === "monat"}
      <div class="space-y-1.5">
        <label class="block text-sm font-medium text-slate-700" for="monat">Monat</label>
        <input
          id="monat"
          type="month"
          bind:value={monatJahr}
          class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
        />
      </div>

    {:else if typ === "jahr"}
      <div class="space-y-1.5">
        <label class="block text-sm font-medium text-slate-700" for="jahr">Jahr</label>
        <select
          id="jahr"
          bind:value={jahr}
          class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
        >
          {#each yearOptions as y}
            <option value={y}>{y}</option>
          {/each}
        </select>
      </div>

    {:else}
      <div class="space-y-1.5">
        <label class="block text-sm font-medium text-slate-700" for="lieferant">Lieferant</label>
        {#if suppliers.length === 0}
          <p class="text-sm text-slate-400 italic">Keine Lieferanten vorhanden.</p>
        {:else}
          <select
            id="lieferant"
            bind:value={lieferantId}
            class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
          >
            {#each suppliers as s}
              <option value={s.id}>{s.name}</option>
            {/each}
          </select>
        {/if}
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-1.5">
          <label class="block text-sm font-medium text-slate-700" for="von">Von</label>
          <input
            id="von"
            type="date"
            bind:value={vonDatum}
            class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
          />
        </div>
        <div class="space-y-1.5">
          <label class="block text-sm font-medium text-slate-700" for="bis">Bis</label>
          <input
            id="bis"
            type="date"
            bind:value={bisDatum}
            class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
          />
        </div>
      </div>
    {/if}
  </div>

  <!-- Download-Button -->
  <a
    href={buildURL()}
    target="_blank"
    rel="noopener"
    class="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-xl shadow transition-colors text-sm"
  >
    🖨️ PDF herunterladen
  </a>

  <p class="text-xs text-slate-400">
    Das PDF öffnet sich im Browser — von dort ausdrucken oder als Datei speichern.
  </p>
</div>
