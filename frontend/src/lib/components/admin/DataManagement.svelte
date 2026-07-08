<!--
  @component DataManagement
  Verwaltungszentrum für den Import und Export von Medien- und Katalogdaten.
  Ermöglicht den Littera XML/CSV/XLSX-Import sowie den vollständigen CSV-Katalog-Export.
-->
<script lang="ts">
	import LitteraImportWidget from "../../LitteraImportWidget.svelte";
	import { exportiereCSV } from "../../../inventur/lib/admin_api.js";
	import PageContainer from "../layout/PageContainer.svelte";
	import LusdImportView from "../students/LusdImportView.svelte";
	import PromoteStudentsView from "../students/PromoteStudentsView.svelte";

	let isExporting = $state(false);
	let exportError = $state<string | null>(null);

	async function handleExport() {
		isExporting = true;
		exportError = null;
		try {
			await exportiereCSV();
		} catch (err: any) {
			exportError = err.message || "Export fehlgeschlagen";
		} finally {
			isExporting = false;
		}
	}

	let files: FileList | null = $state(null);
	let isImportingCsv = $state(false);
	let importCsvResult: { type: 'success' | 'error', message: string } | null = $state(null);

	let isSyncingCovers = $state(false);
	let syncCoversResult: { type: 'success' | 'error', message: string } | null = $state(null);

	async function handleSyncCovers() {
		isSyncingCovers = true;
		syncCoversResult = null;
		try {
			const token = typeof document !== 'undefined' ? document.cookie.split('; ').find(row => row.startsWith('csrf_token='))?.split('=')[1] : '';
			const res = await fetch('/api/admin/sync-covers', {
				method: 'POST',
				credentials: 'include',
				headers: token ? { 'X-CSRF-Token': decodeURIComponent(token) } : {}
			});
			const data = await res.json();
			if (!res.ok) throw new Error(data.error || 'Fehler beim Starten des Cover-Syncs');
			syncCoversResult = { type: 'success', message: data.message || 'Job gestartet.' };
		} catch (err: any) {
			syncCoversResult = { type: 'error', message: err.message || 'Job konnte nicht gestartet werden.' };
		} finally {
			isSyncingCovers = false;
		}
	}

	async function handleBestandUpload() {
		if (!files || files.length === 0) return;
		isImportingCsv = true;
		importCsvResult = null;

		const formData = new FormData();
		formData.append('file', files[0]);

		try {
			// Using native fetch to preserve automatic multipart/form-data with boundaries
			const token = typeof document !== 'undefined' ? document.cookie.split('; ').find(row => row.startsWith('csrf_token='))?.split('=')[1] : '';
			const res = await fetch('/api/admin/import-bestand', {
				method: 'POST',
				body: formData,
				credentials: 'include',
				headers: token ? { 'X-CSRF-Token': decodeURIComponent(token) } : {}
			});
			const data = await res.json();
			if (!res.ok) throw new Error(data.error || 'Bestands-Import fehlgeschlagen');

			importCsvResult = {
				type: 'success',
				message: `Kombi-Import erfolgreich! ${data.new_titles_count || 0} neue Titel und ${data.imported_copies_count || 0} Exemplare wurden verarbeitet.`
			};
			files = null;
		} catch (err: any) {
			importCsvResult = {
				type: 'error',
				message: err.message || 'Ein unerwarteter Fehler ist aufgetreten.'
			};
		} finally {
			isImportingCsv = false;
		}
	}
</script>

{#snippet adminCard(title: string, description: string, iconPath: string, contentSnippet: any)}
	<div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70 space-y-6">
		<div class="flex items-start gap-4">
			<div class="p-3 bg-blue-50 text-blue-600 rounded-2xl">
				<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={iconPath} />
				</svg>
			</div>
			<div>
				<h3 class="text-lg font-bold text-slate-900">{title}</h3>
				<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">{description}</p>
			</div>
		</div>
		<div class="pt-4 border-t border-slate-50">
			{@render contentSnippet()}
		</div>
	</div>
{/snippet}

{#snippet actionButton(label: string, iconPath: string, onclick: () => void, disabled: boolean, loading: boolean)}
	<button
		{onclick}
		{disabled}
		class="px-6 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-xl transition-all cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-sm flex items-center gap-2"
	>
		{#if loading}
			<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
			<span>Bitte warten...</span>
		{:else}
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={iconPath} />
			</svg>
			<span>{label}</span>
		{/if}
	</button>
{/snippet}

{#snippet importContent()}
	<div class="flex flex-col gap-8">
		<LitteraImportWidget />

		<div class="pt-6 border-t border-slate-100">
			<h4 class="text-sm font-bold text-slate-900 mb-1">Finaler Bestands-Import (Kombi-CSV)</h4>
			<p class="text-xs text-slate-500 mb-4">Laden Sie die finale Semikolon-separierte CSV hoch (Spalten: Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand).</p>
			
			<div class="flex items-center gap-4">
				<label class="relative {isImportingCsv ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
					<input type="file" accept=".csv" bind:files disabled={isImportingCsv} class="sr-only" />
					<div class="px-5 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-sm rounded-xl transition-colors border border-slate-200 inline-block">
						{files && files.length > 0 ? files[0].name : 'CSV-Datei auswählen...'}
					</div>
				</label>
				
				<button 
					onclick={handleBestandUpload} 
					disabled={isImportingCsv || !files || files.length === 0}
					class="px-6 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-sm rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
				>
					{#if isImportingCsv}
						<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
						<span>Importiere Bestand...</span>
					{:else}
						<span>Import Starten</span>
					{/if}
				</button>
			</div>

			{#if importCsvResult}
				<div class="mt-4 p-4 rounded-xl text-sm font-semibold {importCsvResult.type === 'error' ? 'bg-rose-50 text-rose-600 border border-rose-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
					{importCsvResult.message}
				</div>
			{/if}
		</div>

		<div class="pt-6 border-t border-slate-100">
			<h4 class="text-sm font-bold text-slate-900 mb-1">Cover-Synchronisation</h4>
			<p class="text-xs text-slate-500 mb-4">Laden Sie fehlende Buchcover im Hintergrund asynchron aus externen APIs herunter (z.B. Google Books, DNB).</p>
			
			{@render actionButton(
				"Fehlende Cover im Hintergrund laden",
				"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12",
				handleSyncCovers,
				isSyncingCovers,
				isSyncingCovers
			)}

			{#if syncCoversResult}
				<div class="mt-4 p-4 rounded-xl text-sm font-semibold {syncCoversResult.type === 'error' ? 'bg-rose-50 text-rose-600 border border-rose-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
					{syncCoversResult.message}
				</div>
			{/if}
		</div>
	</div>
{/snippet}

{#snippet exportContent()}
	<div class="flex flex-col gap-4">
		<div>
			{@render actionButton(
				"Katalog als CSV herunterladen",
				"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4",
				handleExport,
				isExporting,
				isExporting
			)}
		</div>
		{#if exportError}
			<div class="p-4 rounded-xl text-sm font-semibold bg-rose-50 text-rose-600 border border-rose-100">
				{exportError}
			</div>
		{/if}
	</div>
{/snippet}

<PageContainer>
	<div class="space-y-8">
		<div>
			<h2 class="text-xl font-bold text-slate-950">Datenverwaltung</h2>
			<p class="text-xs text-slate-500 mt-1">Hier können Sie den gesamten Medienbestand exportieren oder neue Daten importieren.</p>
		</div>

		<div class="grid grid-cols-1 gap-8">
			<!-- Import Card -->
			{@render adminCard(
				"Daten importieren",
				"Aktualisieren Sie den Bestand via MAB2-XML oder legen Sie neue Titel und Exemplare via Excel/CSV an.",
				"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12",
				importContent
			)}

			<!-- Export Card -->
			{@render adminCard(
				"Daten exportieren",
				"Exportieren Sie den aktuellen Medien- und Buchbestand vollständig als CSV-Datei zur weiteren Bearbeitung oder Archivierung.",
				"M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4",
				exportContent
			)}
		</div>

		<!-- Schuljahreswechsel: Schüler-Stammdaten (LUSD) + Versetzungs-Batch. Bewusst
		     AUßERHALB der Schatten-Kacheln oben, damit das eigene Edge-to-Edge-Listen-
		     Design beider Komponenten nicht in einer Kachel-in-Kachel verschachtelt wird. -->
		<div class="pt-8 border-t border-slate-200 space-y-10">
			<div>
				<h2 class="text-xl font-bold text-slate-950">Schuljahreswechsel & Import</h2>
				<p class="text-xs text-slate-500 mt-1">LUSD-Datenabgleich und automatische Klassen-Versetzung für das Ende des Schuljahres.</p>
			</div>

			<LusdImportView />

			<div class="pt-8 border-t border-slate-100">
				<PromoteStudentsView />
			</div>
		</div>
	</div>
</PageContainer>
