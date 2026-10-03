<!--
  @component DataManagement
  Verwaltungszentrum für den Import und Export von Medien- und Katalogdaten: Katalog-Import
  (Littera), Bestands-Import (Kombi-CSV), Listenimport (ISBN + Stückzahl), Cover-Sync und
  CSV-Export. Jeder Import ist ein eigenes Widget, damit diese Datei unter der
  200-Zeilen-Regel bleibt.
-->
<script lang="ts">
	import { Download, Upload } from '@lucide/svelte';
	import LitteraImportWidget from '../../LitteraImportWidget.svelte';
	import Button from '../ui/Button.svelte';
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
{#snippet adminCard(title: string, description: string, Symbol: typeof Upload, contentSnippet: any)}
	<!-- Flach über die volle Breite: keine Kachel mit Schatten, nur ein Trennstrich zwischen
	     den Abschnitten, wie in jeder anderen Einstellungs-Kategorie. -->
	<div class="border-outline-variant space-y-6 border-b pb-8">
		<div class="flex items-start gap-4">
			<div class="bg-primary-container text-on-primary-container rounded-full p-3">
				<Symbol class="h-6 w-6" aria-hidden="true" />
			</div>
			<div>
				<h3 class="text-lg font-bold text-on-surface">{title}</h3>
				<p class="mt-1 max-w-lg text-xs leading-relaxed text-on-surface-variant">{description}</p>
			</div>
		</div>
		<div class="pt-2">
			{@render contentSnippet()}
		</div>
	</div>
{/snippet}

<!-- Umrandet statt gefüllt: Beides sind eigene Werkzeuge der Seite, kein Abschluss eines
     Ablaufs. Gefüllt ist hier nur der Import einer gewählten Datei. -->
{#snippet actionButton(label: string, Symbol: typeof Upload, onclick: () => void, laeuft: boolean)}
	<Button variant="secondary" size="lg" class="px-6" {onclick} disabled={laeuft}>
		{#if laeuft}
			<Ladekreis size="sm" farbe="aktuell" />
			<span>Bitte warten...</span>
		{:else}
			<Symbol class="h-4 w-4" aria-hidden="true" />
			<span>{label}</span>
		{/if}
	</Button>
{/snippet}

{#snippet importContent()}
	<div class="flex flex-col gap-8">
		<LitteraImportWidget />

		<BestandImportWidget />
		<ListenImportWidget />

		<div class="border-t border-outline-variant pt-6">
			<h4 class="mb-1 text-sm font-bold text-on-surface">Cover-Synchronisation</h4>
			<p class="mb-4 text-xs text-on-surface-variant">
				Laden Sie fehlende Buchcover im Hintergrund asynchron aus externen APIs herunter (z.B.
				Google Books, DNB).
			</p>

			{@render actionButton(
				'Fehlende Cover im Hintergrund laden',
				Upload,
				handleSyncCovers,
				isSyncingCovers
			)}

			{#if syncCoversResult}
				<div
					class="mt-4 rounded-xl p-4 text-sm font-semibold {syncCoversResult.type === 'error'
						? 'bg-error-container text-on-error-container'
						: 'bg-success-container text-on-success-container'}"
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
			{@render actionButton('Katalog als CSV herunterladen', Download, handleExport, isExporting)}
		</div>
		{#if exportError}
			<div class="rounded-xl bg-error-container p-4 text-sm font-semibold text-on-error-container">
				{exportError}
			</div>
		{/if}
	</div>
{/snippet}

<!-- Titel und Beitext kommen vom KategorieRahmen (KategorieDetail.svelte). -->
<div class="space-y-8">
	<div class="grid grid-cols-1 gap-8">
		{#if darfImport}
			{@render adminCard(
				'Daten importieren',
				'Aktualisieren Sie den Bestand via MAB2-XML oder legen Sie neue Titel und Exemplare via Excel/CSV an.',
				Upload,
				importContent
			)}
		{/if}
		{#if darfExport}
			{@render adminCard(
				'Daten exportieren',
				'Exportieren Sie den aktuellen Medien- und Buchbestand vollständig als CSV-Datei zur weiteren Bearbeitung oder Archivierung.',
				Download,
				exportContent
			)}
		{/if}
	</div>

	<OfflineSicherungenEinspielen />
</div>
