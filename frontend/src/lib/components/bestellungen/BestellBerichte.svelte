<script>
  import { localISO, lastOfMonth } from "../../utils/dates.js";

  /** @type {{ suppliers?: { id: string, name: string }[] }} */
  let { suppliers = [] } = $props();

  /** @type {"monat" | "jahr" | "lieferant"} */
  let typ = $state("monat");

  const now = new Date();

  // Monatsbericht
  let monatJahr = $state(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`);

  // Jahresbericht
  let jahr = $state(String(now.getFullYear()));

  // Lieferantenabrechnung — Vorauswahl folgt den Props, Nutzer-Auswahl überstimmt sie
  let lieferantId = $derived(suppliers[0]?.id ?? "");
  let vonDatum = $state(localISO(new Date(now.getFullYear(), now.getMonth(), 1)));
  let bisDatum = $state(localISO(now));

  const monthLabels = ["Januar","Februar","März","April","Mai","Juni","Juli","August","September","Oktober","November","Dezember"];

  // Jahres-Optionen: aktuelles Jahr + 4 zurück
  const yearOptions = Array.from({ length: 5 }, (_, i) => String(now.getFullYear() - i));

  const berichtOptionen = [
    { value: "monat", label: "Monatsbericht", desc: "Alle Bestellungen eines Monats mit Titeln und Summe" },
    { value: "jahr", label: "Jahresbericht", desc: "Monatliche Übersicht + Aufteilung nach Lieferant" },
    { value: "lieferant", label: "Lieferantenabrechnung", desc: "Alle Bestellungen bei einem Lieferanten in einem Zeitraum" },
  ];

  let rangeInvalid = $derived(typ === "monat" && vonDatum > bisDatum);
  let canDownload = $derived(!rangeInvalid && (typ !== "lieferant" || lieferantId !== ""));

  let downloadURL = $derived.by(() => {
    const base = "/api/bestellhistorie/bericht";
    if (typ === "monat") {
      const [y, m] = monatJahr.split("-");
      const params = new URLSearchParams({
        von: `${monatJahr}-01`,
        bis: lastOfMonth(monatJahr),
        titel: `Monatsbericht ${monthLabels[Number(m) - 1] ?? ""} ${y}`,
      });
      return `${base}?${params}`;
    }
    if (typ === "jahr") {
      const params = new URLSearchParams({
        von: `${jahr}-01-01`,
        bis: `${jahr}-12-31`,
        jahresansicht: "true",
        titel: `Jahresbericht ${jahr}`,
      });
      return `${base}?${params}`;
    }
    const name = suppliers.find((s) => s.id === lieferantId)?.name ?? "Lieferant";
    const params = new URLSearchParams({
      von: vonDatum,
      bis: bisDatum,
      lieferant_id: lieferantId,
      titel: `Lieferantenabrechnung: ${name}`,
    });
    return `${base}?${params}`;
  });
</script>

<div class="max-w-3xl space-y-8 overflow-y-auto">
  <!-- Berichtstyp: flache Liste, kein Kachel-Design -->
  <section class="space-y-3">
    <div class="border-b border-slate-200 pb-3">
      <h2 class="text-lg font-bold text-slate-800">Bericht erstellen</h2>
    </div>
    <div class="divide-y divide-slate-100">
      {#each berichtOptionen as opt}
        <label class="flex items-start gap-3 py-3 pl-3 border-l-2 cursor-pointer transition-colors {typ === opt.value ? 'border-blue-600 bg-blue-50/40' : 'border-transparent hover:bg-slate-50/60'}">
          <input type="radio" bind:group={typ} value={opt.value} class="mt-0.5 accent-blue-600" />
          <div>
            <div class="font-bold text-sm text-slate-800">{opt.label}</div>
            <div class="text-xs text-slate-500">{opt.desc}</div>
          </div>
        </label>
      {/each}
    </div>
  </section>

  <!-- Parameter -->
  <section class="space-y-4">
    <p class="text-sm font-medium text-slate-700">Parameter</p>

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
            class="w-full px-3 py-2.5 rounded-lg border bg-white text-base {rangeInvalid ? 'border-rose-400' : 'border-slate-200'}"
          />
        </div>
        <div class="space-y-1.5">
          <label class="block text-sm font-medium text-slate-700" for="bis">Bis</label>
          <input
            id="bis"
            type="date"
            bind:value={bisDatum}
            class="w-full px-3 py-2.5 rounded-lg border bg-white text-base {rangeInvalid ? 'border-rose-400' : 'border-slate-200'}"
          />
        </div>
      </div>
      {#if rangeInvalid}
        <p class="text-sm text-rose-600 font-medium">Das Von-Datum liegt nach dem Bis-Datum.</p>
      {/if}
    {/if}
  </section>

  <!-- Download -->
  <section class="space-y-3">
    <a
      href={canDownload ? downloadURL : undefined}
      target="_blank"
      rel="noopener"
      aria-disabled={!canDownload}
      class="inline-flex items-center gap-2 px-6 py-3 font-bold rounded-lg transition-colors text-sm {canDownload ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'bg-slate-200 text-slate-400 pointer-events-none'}"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
      </svg>
      PDF herunterladen
    </a>
    <p class="text-xs text-slate-400">
      Das PDF öffnet sich im Browser — von dort ausdrucken oder als Datei speichern.
    </p>
  </section>
</div>
