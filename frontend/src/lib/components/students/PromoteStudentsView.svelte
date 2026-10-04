<!-- @component PromoteStudentsView — Admin-Batch für den Schuljahreswechsel.
     Dreistufig: Dry-Run-Vorschau (Server rechnet identisches SQL und rollt zurück)
     → Ausführen-Knopf → rote Bestätigung. Kein window.confirm/Modal. -->
<script>
	import { TriangleAlert, CircleCheck, Info } from '@lucide/svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Button from '../ui/Button.svelte';

	/** @typedef {{ promoted_count: number, archived_count: number, dry_run: boolean, mapping_versetzt: number, mapping_entfernt: number, mapping_konflikte?: string[] }} PromoteStudentsResponse */

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
						valueClass: 'text-success'
					},
					{
						key: 'archived',
						label: 'Neue Abgänger',
						hint: 'Abschlussklassen werden archiviert',
						value: r.archived_count,
						valueClass: 'text-error'
					},
					{
						key: 'mapping',
						label: 'Klassenlehrer-Zuordnungen',
						hint:
							'rücken mit hoch, außer bei Abschluss, vor Klasse 7 und vor der Oberstufe' +
							(r.mapping_konflikte?.length
								? ' — Namenskonflikte: ' + r.mapping_konflikte.join(', ')
								: ''),
						value: r.mapping_versetzt + r.mapping_entfernt,
						valueClass: 'text-on-surface-variant'
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
	<ul class="divide-y divide-outline-variant">
		{#each rows(r) as row (row.key)}
			<li class="flex items-center justify-between py-3">
				<div class="min-w-0">
					<p class="text-sm font-bold text-on-surface">{row.label}</p>
					<p class="text-xs text-on-surface-variant mt-0.5">{row.hint}</p>
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
		<h3 class="text-base font-medium text-on-surface">Schuljahreswechsel</h3>
		<p class="mt-1 max-w-xl text-sm text-on-surface-variant">
			Zählt die Klassenbezeichnung aller aktiven Schüler stur um eine Jahrgangsstufe hoch (z. B. 5a
			→ 6a) und markiert Abschlussklassen automatisch als Abgänger. Ausnahmen wie Sitzenbleiber oder
			individuelle Klassenwechsel lassen sich danach gezielt per LUSD-Import korrigieren.
		</p>
	</div>

	{#if errorMessage}
		<div
			class="flex items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
		>
			<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" /><span>{errorMessage}</span>
		</div>
	{/if}

	{#if result}
		<div
			class="flex items-center gap-2 rounded-xl bg-success-container px-4 py-3 text-sm text-on-success-container"
		>
			<CircleCheck class="h-4 w-4 shrink-0" aria-hidden="true" />
			<span>Schuljahreswechsel abgeschlossen.</span>
		</div>
		{@render summaryRows(result)}
		<Button onclick={reset}>Fertig</Button>
	{:else if !preview}
		<!-- Stufe 1: erst die unverbindliche Vorschau — der Server rechnet das
         identische SQL in einer Transaktion und rollt zurück. -->
		<div class="flex justify-end">
			<Button onclick={runPreview} disabled={loading}>
				{#if loading}
					<Ladekreis size="sm" farbe="aktuell" /> Vorschau wird berechnet…
				{:else}
					Vorschau berechnen
				{/if}
			</Button>
		</div>
	{:else}
		<div
			class="flex items-center gap-2 rounded-xl bg-primary-container px-4 py-3 text-sm text-on-primary-container"
		>
			<Info class="h-4 w-4 shrink-0" aria-hidden="true" />
			<span>Unverbindliche Vorschau — es wurde noch nichts geändert.</span>
		</div>
		{@render summaryRows(preview)}

		<div
			class="flex items-start gap-2 rounded-xl bg-warning-container px-4 py-3 text-sm text-on-warning-container"
		>
			<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
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
						<Ladekreis size="sm" farbe="aktuell" /> Wird ausgeführt…
					{:else}
						Ja, unwiderruflich ausführen
					{/if}
				</Button>
			</div>
		{/if}
	{/if}
</div>
