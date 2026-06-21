<!--
  @component DataManagement
  Verwaltungszentrum für den Import und Export von Medien- und Katalogdaten.
  Ermöglicht den Littera XML/CSV/XLSX-Import sowie den vollständigen CSV-Katalog-Export.
-->
<script lang="ts">
	import LitteraImportWidget from "../../LitteraImportWidget.svelte";
	import { exportiereCSV } from "../../../inventur/lib/admin_api.js";

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
	<LitteraImportWidget />
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

<div class="space-y-8 max-w-4xl">
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
</div>
