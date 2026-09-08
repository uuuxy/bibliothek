<!--
  @component DataManagement
  Verwaltungszentrum für den Import und Export von Medien- und Katalogdaten: Katalog-Import
  (Littera), Bestands-Import (Kombi-CSV), Listenimport (ISBN + Stückzahl), Cover-Sync und
  CSV-Export. Jeder Import ist ein eigenes Widget — seit dem 07.09.2026 auch der Bestand,
  damit diese Datei unter der 200-Zeilen-Regel bleibt.
-->
<script lang="ts">
	import LitteraImportWidget from '../../LitteraImportWidget.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import BestandImportWidget from './BestandImportWidget.svelte';
	import ListenImportWidget from './ListenImportWidget.svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { exportiereCSV } from '../../../inventur/lib/admin_api.js';
	import OfflineSicherungenEinspielen from './OfflineSicherungenEinspielen.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';

	const darfImport = $derived(hatRecht(authStore.currentUser, 'manage_inventory'));
	const darfExport = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	let isExporting = $state(false);
	let exportError = $state<string | null>(null);

	async function handleExport() {
		isExporting = true;
		exportError = null;
		try {
			await exportiereCSV();
		} catch (err) {
			exportError = (err instanceof Error && err.message) || 'Export fehlgeschlagen';
		} finally {
			isExporting = false;
		}
	}

	let isSyncingCovers = $state(false);
	let syncCoversResult: { type: 'success' | 'error'; message: string } | null = $state(null);

	async function handleSyncCovers() {
		isSyncingCovers = true;
		syncCoversResult = null;
		try {
			const res = await apiFetch('/api/admin/sync-covers', { method: 'POST' });
			const data = await res.json();
			if (!res.ok) throw new Error(data.error || 'Fehler beim Starten des Cover-Syncs');
			syncCoversResult = { type: 'success', message: data.message || 'Job gestartet.' };
		} catch (err) {
			syncCoversResult = {
				type: 'error',
				message: (err instanceof Error && err.message) || 'Job konnte nicht gestartet werden.'
			};
		} finally {
			isSyncingCovers = false;
		}
	}
</script>

<!-- contentSnippet bleibt `any`: der explizite Snippet-Import aus 'svelte' kollidiert hier mit
     dem ambienten Snippet-Typ, den svelte-check für Inline-Snippets verwendet („Two different
     types with this name exist") — die Typisierung erzeugte zwei neue Fehler statt Sicherheit. -->
<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
{#snippet adminCard(title: string, description: string, iconPath: string, contentSnippet: any)}
	<!-- Flach, edge-to-edge (Entscheidung f2320e1/e81ce75/95d5d33): keine Schatten-Kachel,
	     nur ein Trennstrich zwischen den Abschnitten — wie jede andere Einstellungs-Kategorie. -->
	<div class="border-outline-variant space-y-6 border-b pb-8">
		<div class="flex items-start gap-4">
			<div class="bg-primary-container text-on-primary-container rounded-full p-3">
				<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={iconPath} />
				</svg>
			</div>
			<div>
				<h3 class="text-lg font-bold text-slate-900">{title}</h3>
				<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">{description}</p>
			</div>
		</div>
		<div class="pt-2">
			{@render contentSnippet()}
		</div>
	</div>
{/snippet}

{#snippet actionButton(
	label: string,
	iconPath: string,
	onclick: () => void,
	disabled: boolean,
	loading: boolean
)}
	<button
		{onclick}
		{disabled}
		class="px-6 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-xl transition-all cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-sm flex items-center gap-2"
	>
		{#if loading}
			<Ladekreis size="sm" farbe="aktuell" />
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

		<BestandImportWidget />
		<ListenImportWidget />

		<div class="pt-6 border-t border-slate-100">
			<h4 class="text-sm font-bold text-slate-900 mb-1">Cover-Synchronisation</h4>
			<p class="text-xs text-slate-500 mb-4">
				Laden Sie fehlende Buchcover im Hintergrund asynchron aus externen APIs herunter (z.B.
				Google Books, DNB).
			</p>

			{@render actionButton(
				'Fehlende Cover im Hintergrund laden',
				'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12',
				handleSyncCovers,
				isSyncingCovers,
				isSyncingCovers
			)}

			{#if syncCoversResult}
				<div
					class="mt-4 p-4 rounded-xl text-sm font-semibold {syncCoversResult.type === 'error'
						? 'bg-rose-50 text-rose-600 border border-rose-100'
						: 'bg-emerald-50 text-emerald-700 border border-emerald-100'}"
				>
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
				'Katalog als CSV herunterladen',
				'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
				handleExport,
				isExporting,
				isExporting
			)}
		</div>
		{#if exportError}
			<div
				class="p-4 rounded-xl text-sm font-semibold bg-rose-50 text-rose-600 border border-rose-100"
			>
				{exportError}
			</div>
		{/if}
	</div>
{/snippet}

<!-- Titel und Beitext kommen vom KategorieRahmen (SystemSettings.svelte) — bis zum
     24.08.2026 stand „Datenverwaltung“ hier ein zweites Mal direkt darunter. -->
<div class="space-y-8">
	<div class="grid grid-cols-1 gap-8">
		{#if darfImport}
			{@render adminCard(
				'Daten importieren',
				'Aktualisieren Sie den Bestand via MAB2-XML oder legen Sie neue Titel und Exemplare via Excel/CSV an.',
				'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12',
				importContent
			)}
		{/if}
		{#if darfExport}
			{@render adminCard(
				'Daten exportieren',
				'Exportieren Sie den aktuellen Medien- und Buchbestand vollständig als CSV-Datei zur weiteren Bearbeitung oder Archivierung.',
				'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
				exportContent
			)}
		{/if}
	</div>

	<OfflineSicherungenEinspielen />
</div>
