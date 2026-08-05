<!-- @component PromoteStudentsView — Admin-Batch für den Schuljahreswechsel.
     Dreistufig: Dry-Run-Vorschau (Server rechnet identisches SQL und rollt zurück)
     → Ausführen-Knopf → rote Bestätigung. Kein window.confirm/Modal. -->
<script>
	import { AlertTriangle, CircleCheck } from '@lucide/svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Button from '../ui/Button.svelte';

	/** @typedef {{ promoted_count: number, archived_count: number, dry_run: boolean }} PromoteStudentsResponse */

	let awaitingConfirmation = $state(false);
	let loading = $state(false);
	/** @type {PromoteStudentsResponse | null} */
	let preview = $state(null);
	/** @type {PromoteStudentsResponse | null} */
	let result = $state(null);
	/** @type {string | null} */
	let errorMessage = $state(null);

	/** @param {PromoteStudentsResponse | null} r */
	function rows(r) {
		return r
			? [
					{
						key: 'promoted',
						label: 'Versetzte Schüler',
						hint: 'Klasse wird um eine Stufe hochgezählt',
						value: r.promoted_count,
						valueClass: 'text-emerald-600'
					},
					{
						key: 'archived',
						label: 'Neue Abgänger',
						hint: 'Abschlussklassen werden archiviert',
						value: r.archived_count,
						valueClass: 'text-rose-600'
					}
				]
			: [];
	}

	function reset() {
		preview = null;
		result = null;
		awaitingConfirmation = false;
		errorMessage = null;
	}

	/** @param {{ dry_run?: boolean, confirm?: boolean }} body */
	async function callPromote(body) {
		const res = await apiFetch('/api/students/promote', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body)
		});
		if (!res.ok) {
			const data = await res.json().catch(() => null);
			throw new Error(data?.error || 'Schuljahreswechsel fehlgeschlagen.');
		}
		return res.json();
	}

	async function runPreview() {
		if (loading) return;
		loading = true;
		errorMessage = null;
		try {
			preview = await callPromote({ dry_run: true });
		} catch (err) {
			errorMessage = /** @type {any} */ (err).message || String(err);
		} finally {
			loading = false;
		}
	}

	// Rückmeldung läuft NUR über die Banner in dieser Ansicht — kein zusätzlicher Toast.
	// Beides zusammen zeigte dieselbe Meldung doppelt (im Erfolgs- wie im Fehlerfall).
	// Die Ansicht ist kurz, die Banner stehen direkt beim Vorgang, und der Ergebnisblock
	// bleibt bis „Fertig" stehen — ein Toast, der nach Sekunden verschwindet, kann für
	// einen irreversiblen Massenvorgang ohnehin nicht der tragende Kanal sein.
	async function executePromotion() {
		if (loading) return;
		loading = true;
		errorMessage = null;
		try {
			result = await callPromote({ confirm: true });
		} catch (err) {
			errorMessage = /** @type {any} */ (err).message || String(err);
		} finally {
			awaitingConfirmation = false;
			loading = false;
		}
	}
</script>

{#snippet summaryRows(r)}
	<ul class="divide-y divide-slate-100">
		{#each rows(r) as row (row.key)}
			<li class="flex items-center justify-between py-3">
				<div class="min-w-0">
					<p class="text-sm font-bold text-slate-800">{row.label}</p>
					<p class="text-xs text-slate-500 mt-0.5">{row.hint}</p>
				</div>
				<span class="text-lg font-black tabular-nums shrink-0 ml-4 {row.valueClass}"
					>{row.value}</span
				>
			</li>
		{/each}
	</ul>
{/snippet}

<div class="w-full max-w-2xl space-y-8">
	<div>
		<h2 class="text-base font-bold text-slate-900">Schuljahreswechsel</h2>
		<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-xl">
			Zählt die Klassenbezeichnung aller aktiven Schüler stur um eine Jahrgangsstufe hoch (z. B. 5a
			→ 6a) und markiert Abschlussklassen automatisch als Abgänger. Ausnahmen wie Sitzenbleiber oder
			individuelle Klassenwechsel lassen sich danach gezielt per LUSD-Import korrigieren.
		</p>
	</div>

	{#if errorMessage}
		<div
			class="p-4 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold flex items-center gap-2"
		>
			<AlertTriangle class="h-4 w-4" aria-hidden="true" /><span>{errorMessage}</span>
		</div>
	{/if}

	{#if result}
		<div
			class="p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-800 text-sm font-semibold flex items-center gap-2"
		>
			<CircleCheck class="h-4 w-4" aria-hidden="true" /><span
				>Schuljahreswechsel abgeschlossen.</span
			>
		</div>
		{@render summaryRows(result)}
		<Button onclick={reset}>Fertig</Button>
	{:else if !preview}
		<!-- Stufe 1: erst die unverbindliche Vorschau — der Server rechnet das
         identische SQL in einer Transaktion und rollt zurück. -->
		<div class="flex justify-end">
			<Button onclick={runPreview} disabled={loading}>
				{#if loading}
					<span
						class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"
					></span> Vorschau wird berechnet…
				{:else}
					Vorschau berechnen
				{/if}
			</Button>
		</div>
	{:else}
		<div
			class="p-4 rounded-xl bg-blue-50 border border-blue-100 text-blue-800 text-xs font-semibold flex items-center gap-2"
		>
			<span>🔍</span><span>Unverbindliche Vorschau — es wurde noch nichts geändert.</span>
		</div>
		{@render summaryRows(preview)}

		<div
			class="p-4 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-semibold flex items-start gap-2"
		>
			<AlertTriangle class="h-4 w-4" aria-hidden="true" />
			<span
				>Dieser Vorgang ist <strong>irreversibel</strong> und betrifft alle aktiven Schüler gleichzeitig.
				Es gibt keinen automatischen Rückweg — nur ein erneuter LUSD-Import kann einzelne Datensätze danach
				noch korrigieren.</span
			>
		</div>

		{#if !awaitingConfirmation}
			<div class="flex justify-end gap-3">
				<Button variant="secondary" onclick={reset} disabled={loading}>Abbrechen</Button>
				<Button onclick={() => (awaitingConfirmation = true)} disabled={loading}>
					Schuljahr wechseln
				</Button>
			</div>
		{:else}
			<div class="flex justify-end gap-3">
				<Button
					variant="secondary"
					onclick={() => (awaitingConfirmation = false)}
					disabled={loading}
				>
					Abbrechen
				</Button>
				<Button variant="danger-solid" onclick={executePromotion} disabled={loading}>
					{#if loading}
						<span
							class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"
						></span> Wird ausgeführt…
					{:else}
						Ja, unwiderruflich ausführen
					{/if}
				</Button>
			</div>
		{/if}
	{/if}
</div>
