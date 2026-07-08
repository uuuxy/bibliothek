<!-- @component PromoteStudentsView — Admin-Batch für den Schuljahreswechsel.
     Zählt Klassenbezeichnungen stur um eine Stufe hoch und archiviert Abschlussklassen.
     Zweistufige Bestätigung (State-Toggle, kein window.confirm/Modal) gegen Fehlklicks. -->
<script>
  import { apiFetch } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  /** @typedef {{ promoted_count: number, archived_count: number }} PromoteStudentsResponse */

  let awaitingConfirmation = $state(false);
  let loading = $state(false);
  /** @type {PromoteStudentsResponse | null} */
  let result = $state(null);
  /** @type {string | null} */
  let errorMessage = $state(null);

  const summaryRows = $derived(
    result
      ? [
          { key: "promoted", label: "Versetzte Schüler", hint: "Klasse wurde um eine Stufe hochgezählt", value: result.promoted_count, valueClass: "text-emerald-600" },
          { key: "archived", label: "Neue Abgänger", hint: "Abschlussklassen wurden archiviert", value: result.archived_count, valueClass: "text-rose-600" },
        ]
      : [],
  );

  function requestConfirmation() {
    awaitingConfirmation = true;
  }

  function cancelConfirmation() {
    awaitingConfirmation = false;
  }

  function reset() {
    result = null;
    awaitingConfirmation = false;
    errorMessage = null;
  }

  async function executePromotion() {
    if (loading) return;
    loading = true;
    errorMessage = null;
    try {
      const res = await apiFetch("/api/students/promote", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ confirm: true }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.error || "Schuljahreswechsel fehlgeschlagen.");
      }
      result = await res.json();
      awaitingConfirmation = false;
      toastStore.addToast("Schuljahreswechsel erfolgreich durchgeführt.", "success");
    } catch (err) {
      errorMessage = /** @type {any} */ (err).message || String(err);
      toastStore.addToast(errorMessage, "error");
    } finally {
      loading = false;
    }
  }
</script>

{#snippet summaryRow(row)}
  <li class="flex items-center justify-between py-3">
    <div class="min-w-0">
      <p class="text-sm font-bold text-slate-800">{row.label}</p>
      <p class="text-xs text-slate-450 mt-0.5">{row.hint}</p>
    </div>
    <span class="text-lg font-black tabular-nums shrink-0 ml-4 {row.valueClass}">{row.value}</span>
  </li>
{/snippet}

<div class="w-full max-w-2xl space-y-8">
  <div>
    <h2 class="text-base font-bold text-slate-900">Schuljahreswechsel</h2>
    <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-xl">
      Zählt die Klassenbezeichnung aller aktiven Schüler stur um eine Jahrgangsstufe hoch
      (z. B. 5a → 6a) und markiert Abschlussklassen automatisch als Abgänger. Ausnahmen wie
      Sitzenbleiber oder individuelle Klassenwechsel lassen sich danach gezielt per LUSD-Import
      korrigieren.
    </p>
  </div>

  {#if errorMessage}
    <div class="p-4 rounded-xl bg-rose-50 border border-rose-100 text-rose-650 text-xs font-semibold flex items-center gap-2"><span>⚠️</span><span>{errorMessage}</span></div>
  {/if}

  {#if result}
    <div class="p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-800 text-sm font-semibold flex items-center gap-2"><span>🎉</span><span>Schuljahreswechsel abgeschlossen.</span></div>
    <ul class="divide-y divide-slate-100">
      {#each summaryRows as row (row.key)}
        {@render summaryRow(row)}
      {/each}
    </ul>
    <button onclick={reset} class="px-5 py-2.5 rounded-full bg-slate-900 hover:bg-slate-800 text-white text-xs font-bold transition-colors cursor-pointer">
      Fertig
    </button>
  {:else}
    <div class="p-4 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-semibold flex items-start gap-2">
      <span>⚠️</span>
      <span>Dieser Vorgang ist <strong>irreversibel</strong> und betrifft alle aktiven Schüler
        gleichzeitig. Es gibt keinen automatischen Rückweg — nur ein erneuter LUSD-Import kann
        einzelne Datensätze danach noch korrigieren.</span>
    </div>

    {#if !awaitingConfirmation}
      <div class="flex justify-end">
        <button onclick={requestConfirmation} class="px-5 py-2.5 rounded-full bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold transition-all cursor-pointer">
          Schuljahr wechseln
        </button>
      </div>
    {:else}
      <div class="flex justify-end gap-3">
        <button onclick={cancelConfirmation} disabled={loading} class="px-4 py-2.5 rounded-full bg-slate-100 hover:bg-slate-200 text-slate-650 text-xs font-bold transition-colors cursor-pointer disabled:opacity-50">
          Abbrechen
        </button>
        <button onclick={executePromotion} disabled={loading} class="px-5 py-2.5 rounded-full bg-rose-600 hover:bg-rose-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-xs font-bold transition-all cursor-pointer flex items-center gap-2">
          {#if loading}
            <span class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"></span> Wird ausgeführt…
          {:else}
            Ja, unwiderruflich ausführen
          {/if}
        </button>
      </div>
    {/if}
  {/if}
</div>
