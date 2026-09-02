<!-- @component LusdImportView — Preview-to-Commit Flow für den LUSD-Schuljahreswechsel-Import.
     Go-Handler sind zustandslos (keine Preview-Session), daher wird dieselbe Datei bei
     „Import finalisieren“ erneut an /api/lusd/import gesendet — samt der bestätigten
     Umbenennungs-Paare (LusdUmbenennungen).

     Drei Zuordnungsstufen, die Datei entscheidet (Backend: api/lusd_parser.go):
     LUSD-ID → Name + Geburtsdatum → nur Name. Der Export der Schule hat keine Schüler-ID;
     die Vorschau sagt, welche Stufe gilt, und was unangetastet bleibt. Texte und Rubriken
     stehen in lusdVorschauRubriken.js. -->
<script>
	import { AlertTriangle, ChevronRight, CircleCheck } from '@lucide/svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';
	import Button from '../ui/Button.svelte';
	import LusdUmbenennungen from './LusdUmbenennungen.svelte';
	import {
		modusInfo,
		rubriken,
		umbenennungenFormwert,
		vorausgewaehlteZeilen
	} from './lusdVorschauRubriken.js';

	/** @typedef {import('./lusdVorschauRubriken.js').LusdPreviewResult} LusdPreviewResult */

	/** @type {{ onImported?: (result: LusdPreviewResult) => void }} */
	let { onImported = () => {} } = $props();

	let selectedFile = $state(/** @type {File | null} */ (null));
	let stage = $state(/** @type {'upload' | 'preview' | 'done'} */ ('upload'));
	let previewLoading = $state(false);
	let importLoading = $state(false);
	let previewResult = $state(/** @type {LusdPreviewResult | null} */ (null));
	let importResult = $state(/** @type {LusdPreviewResult | null} */ (null));
	let errorMessage = $state(/** @type {string | null} */ (null));
	/** Vom Admin bestätigte Umbenennungs-Paare (Zeilennummern der Datei); wird in
	 *  place geändert — SvelteSet ist selbst reaktiv, ein $state-Umschlag wäre doppelt. */
	const gewaehltePaare = new SvelteSet();

	const activeResult = $derived(stage === 'done' ? importResult : previewResult);
	const info = $derived(modusInfo(activeResult?.modus));
	const summaryRows = $derived(activeResult ? rubriken(activeResult) : []);

	// Bezugsgröße ist der aktive DB-Bestand (kommt vom Server), NICHT die
	// CSV-Zeilenzahl — sonst rutscht ein Teilexport unter die Schwelle.
	const hasRiskyGraduates = $derived(
		!!previewResult &&
			previewResult.active_db_students > 0 &&
			previewResult.graduates.length / previewResult.active_db_students >= 0.3
	);
	/** Serverseitige 409-Bremse hat gegriffen — zweite, bewusste Bestätigung nötig. */
	let needsGraduateConfirm = $state(false);

	function handleFileChange(/** @type {Event} */ e) {
		const target = /** @type {HTMLInputElement} */ (e.target);
		if (target.files && target.files[0]) {
			selectedFile = target.files[0];
			errorMessage = null;
			previewResult = null;
			stage = 'upload';
		}
	}

	function resetFlow() {
		selectedFile = null;
		previewResult = null;
		importResult = null;
		errorMessage = null;
		needsGraduateConfirm = false;
		gewaehltePaare.clear();
		stage = 'upload';
	}

	/** Sendet die aktuell gewählte Datei an einen der beiden LUSD-Endpoints. @param {string} url */
	async function submitLusdFile(url, confirmGraduates = false) {
		const formData = new FormData();
		formData.append('csvFile', /** @type {File} */ (selectedFile));
		if (confirmGraduates) formData.append('confirm_graduates', 'true');
		const wahl = umbenennungenFormwert(previewResult?.umbenennungen ?? [], gewaehltePaare);
		if (wahl) formData.append('umbenennungen', wahl);

		const res = await apiFetch(url, { method: 'POST', body: formData });
		if (!res.ok) {
			const data = await res.json().catch(() => null);
			const err = new Error(data?.error || 'Fehler bei der Verarbeitung der LUSD-Datei.');
			/** @type {any} */ (err).status = res.status;
			throw err;
		}
		return res.json();
	}

	async function runPreview() {
		if (!selectedFile || previewLoading) return;
		previewLoading = true;
		errorMessage = null;
		try {
			previewResult = await submitLusdFile('/api/lusd/preview');
			gewaehltePaare.clear();
			for (const z of vorausgewaehlteZeilen(previewResult?.umbenennungen ?? []))
				gewaehltePaare.add(z);
			stage = 'preview';
		} catch (err) {
			// Nur das Inline-Banner unten — kein zusätzlicher Toast mit identischem Text.
			errorMessage = /** @type {any} */ (err).message || String(err);
		} finally {
			previewLoading = false;
		}
	}

	async function finalizeImport(confirmGraduates = false) {
		if (!selectedFile || importLoading) return;
		importLoading = true;
		errorMessage = null;
		try {
			importResult = await submitLusdFile('/api/lusd/import', confirmGraduates);
			stage = 'done';
			needsGraduateConfirm = false;
			toastStore.addToast('LUSD-Import erfolgreich übernommen.', 'success');
			onImported(/** @type {LusdPreviewResult} */ (importResult));
		} catch (err) {
			// 409 = serverseitige Massenabgang-Bremse: bewusste zweite Bestätigung anbieten.
			needsGraduateConfirm = /** @type {any} */ (err).status === 409;
			errorMessage = /** @type {any} */ (err).message || String(err);
		} finally {
			importLoading = false;
		}
	}
</script>

{#snippet diffSection(section)}
	<details class="group py-1">
		<summary
			class="flex items-center justify-between py-3 cursor-pointer select-none marker:content-none [&::-webkit-details-marker]:hidden"
		>
			<div class="min-w-0 flex items-center gap-2">
				<ChevronRight
					class="w-3 h-3 text-slate-400 shrink-0 transition-transform group-open:rotate-90"
					aria-hidden="true"
				/>
				<div class="min-w-0">
					<p class="text-sm font-bold text-slate-800">{section.label}</p>
					<p class="text-xs text-slate-500 mt-0.5">{section.hint}</p>
				</div>
			</div>
			<span class="text-lg font-black tabular-nums shrink-0 ml-4 {section.valueClass}"
				>{section.items.length}</span
			>
		</summary>
		<ul class="divide-y divide-slate-50 pb-2">
			{#each section.items as item (item.id)}
				<li class="py-2 pl-5 flex items-center justify-between gap-3 text-xs">
					<span class="font-semibold text-slate-700 truncate">{item.vorname} {item.nachname}</span
					><span class="text-slate-400 font-mono shrink-0"
						>{item.alte_klasse && item.neue_klasse
							? `${item.alte_klasse} → ${item.neue_klasse}`
							: item.neue_klasse || item.alte_klasse || '—'}</span
					>
				</li>
			{/each}
		</ul>
	</details>
{/snippet}

{#snippet ergebnis(abgeschlossen)}
	{#if activeResult}
		<p
			class="text-xs font-semibold rounded-xl p-3 {info.warn
				? 'bg-error-container text-on-error-container'
				: 'text-on-surface-variant'}"
		>
			{info.text}
		</p>
		<div class="divide-y divide-slate-100">
			<LusdUmbenennungen
				paare={activeResult.umbenennungen ?? []}
				gewaehlt={gewaehltePaare}
				{abgeschlossen}
			/>
			{#each summaryRows as section (section.key)}
				{@render diffSection(section)}
			{/each}
		</div>
	{/if}
{/snippet}

{#snippet spinner(text)}
	<span class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"
	></span>
	{text}
{/snippet}

<div class="w-full max-w-2xl space-y-8">
	<div>
		<h2 class="text-base font-bold text-slate-900">LUSD-Import</h2>
		<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-xl">
			Lade die LUSD-Exportdatei hoch, um die Änderungen zu prüfen, bevor sie verbindlich in die
			Datenbank übernommen werden. Kein Datensatz wird ohne Bestätigung überschrieben.
		</p>
	</div>

	{#if errorMessage}
		<div
			role="alert"
			class="p-4 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold flex items-center gap-2"
		>
			<AlertTriangle class="h-4 w-4" aria-hidden="true" /><span>{errorMessage}</span>
		</div>
	{/if}

	{#if stage === 'done'}
		<div
			class="p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-800 text-sm font-semibold flex items-center gap-2"
		>
			<CircleCheck class="h-4 w-4" aria-hidden="true" /><span
				>Import abgeschlossen — der Bestand ist aktuell.</span
			>
		</div>
		{@render ergebnis(true)}
		<Button onclick={resetFlow}>Weitere Datei importieren</Button>
	{:else}
		<label
			class="border-2 border-dashed border-slate-200 hover:border-blue-500/70 hover:bg-slate-50/40 transition-all rounded-2xl p-8 flex flex-col items-center justify-center gap-3 cursor-pointer text-center select-none"
		>
			<input
				type="file"
				accept=".csv,.xlsx"
				class="sr-only"
				onchange={handleFileChange}
				disabled={previewLoading || importLoading}
			/>
			<span class="text-3xl">📂</span>
			{#if selectedFile}
				<div class="space-y-1">
					<p class="text-xs font-bold text-slate-700 max-w-xs truncate">{selectedFile.name}</p>
					<p class="text-label-small text-slate-400">{(selectedFile.size / 1024).toFixed(1)} KB</p>
				</div>
			{:else}
				<div class="space-y-1">
					<p class="text-xs font-bold text-slate-600">
						LUSD-CSV oder -Excel auswählen oder reinziehen
					</p>
					<p class="text-label-small text-slate-400 font-medium">
						CSV (Komma/Semikolon) oder .xlsx · Pflichtspalten: <code>vorname, nachname, klasse</code
						>
						· Zuordnung über <code>lusd_id</code>, sonst <code>geburtsdatum</code> (empfohlen),
						sonst nur der Name · <code>eintritt</code> (Schuleintritt) hilft bei Umbenennungen
					</p>
				</div>
			{/if}
		</label>

		{#if stage === 'upload'}
			<div class="flex justify-end">
				<Button onclick={runPreview} disabled={!selectedFile || previewLoading}>
					{#if previewLoading}{@render spinner('Vorschau wird geladen…')}{:else}Vorschau laden{/if}
				</Button>
			</div>
		{/if}

		{#if stage === 'preview' && previewResult}
			<div class="space-y-4">
				<p class="text-xs text-slate-500">
					{previewResult.total_csv_records} Datensätze in der Datei · {previewResult.active_db_students}
					aktive Schüler im Bestand
					{#if previewResult.skipped_no_id > 0}
						· <span class="text-amber-700 font-semibold"
							>{previewResult.skipped_no_id} Zeilen ohne LUSD-ID werden übersprungen</span
						>
					{/if}
					{#if previewResult.dubletten_in_datei > 0}
						· <span class="text-on-surface-variant font-semibold"
							>{previewResult.dubletten_in_datei} doppelte Zeilen zusammengelegt (letzte gewinnt)</span
						>
					{/if}
				</p>
				{#if hasRiskyGraduates}
					<div
						class="p-4 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-semibold flex items-center gap-2"
					>
						<AlertTriangle class="h-4 w-4" aria-hidden="true" /><span
							>Auffällig viele Abgänger ({previewResult.graduates.length} von {previewResult.active_db_students}
							aktiven Schülern) — Datei vor dem Import genau prüfen. Der Import verlangt dafür eine zusätzliche
							Bestätigung.</span
						>
					</div>
				{/if}

				{@render ergebnis(false)}

				<div class="flex justify-end gap-3 pt-2">
					<Button variant="secondary" onclick={resetFlow}>Andere Datei wählen</Button>
					{#if needsGraduateConfirm}
						<Button
							variant="danger-solid"
							onclick={() => finalizeImport(true)}
							disabled={importLoading}
						>
							{#if importLoading}{@render spinner('Import wird übernommen…')}{:else}Massenabgang
								bestätigen &amp; endgültig importieren{/if}
						</Button>
					{:else}
						<Button onclick={() => finalizeImport(false)} disabled={importLoading}>
							{#if importLoading}{@render spinner('Import wird übernommen…')}{:else}Import
								finalisieren{/if}
						</Button>
					{/if}
				</div>
			</div>
		{/if}
	{/if}
</div>
