<!--
  @component Datenverwaltung
  Verwaltungszentrum für den Import und Export von Medien- und Katalogdaten: Katalog-Import
  (Littera), Bestands-Import (Kombi-CSV), Listenimport (ISBN + Stückzahl), Cover-Sync und
  CSV-Export. Jeder Import ist ein eigenes Widget, damit diese Datei unter der
  200-Zeilen-Regel bleibt.
-->
<script>
	import LitteraImport from './LitteraImport.svelte';
	import Button from '../ui/Button.svelte';
	import Icon from '../ui/Icon.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import BestandImport from './BestandImport.svelte';
	import ListenImport from './ListenImport.svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { exportiereCSV } from '../../../inventur/lib/admin_api.js';
	import OfflineSicherungenEinspielen from './OfflineSicherungenEinspielen.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menue.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	const darfImport = $derived(hatRecht(authStore.currentUser, 'manage_inventory'));
	const darfExport = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	let isExporting = $state(false);

	async function exportiere() {
		isExporting = true;
		try {
			await exportiereCSV();
		} catch (err) {
			snackbars.addToast(
				(err instanceof Error && err.message) || 'Der Katalog ließ sich nicht herunterladen',
				'error'
			);
		} finally {
			isExporting = false;
		}
	}

	const COVER_NICHT_GESTARTET = 'Die Suche nach Covern ließ sich nicht starten';
	let isSyncingCovers = $state(false);

	async function gleicheCoverAb() {
		isSyncingCovers = true;
		try {
			const res = await apiFetch('/api/admin/sync-covers', { method: 'POST' });
			if (!res.ok) throw new Error(await leseApiFehler(res, COVER_NICHT_GESTARTET));
			const daten = await res.json();
			snackbars.addToast(
				daten.message || 'Fehlende Cover werden im Hintergrund geladen',
				'success'
			);
		} catch (err) {
			snackbars.addToast((err instanceof Error && err.message) || COVER_NICHT_GESTARTET, 'error');
		} finally {
			isSyncingCovers = false;
		}
	}
</script>

{#snippet adminCard(title, description, symbol, contentSnippet)}
	<!-- Flach über die volle Breite: keine Kachel mit Schatten, nur ein Trennstrich zwischen
	     den Abschnitten, wie in jeder anderen Einstellungs-Kategorie. -->
	<div class="border-outline-variant space-y-6 border-b pb-8">
		<div class="flex items-start gap-4">
			<div class="bg-primary-container text-on-primary-container rounded-full p-3">
				<Icon name={symbol} groesse={24} />
			</div>
			<div>
				<h3 class="text-title-medium text-on-surface">{title}</h3>
				<p class="mt-1 max-w-lg text-sm leading-relaxed text-on-surface-variant">{description}</p>
			</div>
		</div>
		<div class="pt-2">
			{@render contentSnippet()}
		</div>
	</div>
{/snippet}

<!-- Umrandet statt gefüllt: Beides sind eigene Werkzeuge der Seite, kein Abschluss eines
     Ablaufs. Gefüllt ist hier nur der Import einer gewählten Datei. -->
{#snippet actionButton(label, symbol, onclick, laeuft)}
	<Button variant="secondary" size="lg" class="px-6" {onclick} disabled={laeuft}>
		{#if laeuft}
			<Ladekreis size="sm" farbe="aktuell" />
			<span>Wird ausgeführt …</span>
		{:else}
			<Icon name={symbol} />
			<span>{label}</span>
		{/if}
	</Button>
{/snippet}

{#snippet importContent()}
	<div class="flex flex-col gap-8">
		<LitteraImport />

		<BestandImport />
		<ListenImport />

		<div class="border-t border-outline-variant pt-6">
			<h4 class="mb-1 text-title-small text-on-surface">Fehlende Cover</h4>
			<p class="mb-4 text-sm text-on-surface-variant">
				Sucht für Titel ohne Cover ein Bild, etwa bei Google Books oder der Deutschen
				Nationalbibliothek. Der Server arbeitet die Titel im Hintergrund ab.
			</p>

			{@render actionButton('Fehlende Cover laden', 'upload', gleicheCoverAb, isSyncingCovers)}
		</div>
	</div>
{/snippet}

{#snippet exportContent()}
	<div class="flex flex-col gap-4">
		<div>
			{@render actionButton('Katalog als CSV herunterladen', 'download', exportiere, isExporting)}
		</div>
	</div>
{/snippet}

<!-- Titel und Beitext kommen vom KategorieRahmen (KategorieDetail.svelte). -->
<div class="space-y-8">
	<div class="grid grid-cols-1 gap-8">
		{#if darfImport}
			{@render adminCard(
				'Daten importieren',
				'Übernimmt Titel und Exemplare aus Littera oder aus einer Tabelle und lädt fehlende Cover',
				'upload',
				importContent
			)}
		{/if}
		{#if darfExport}
			{@render adminCard(
				'Daten exportieren',
				'Lädt den ganzen Katalog als CSV-Datei herunter, zum Weiterbearbeiten in einer Tabelle oder zum Aufheben',
				'download',
				exportContent
			)}
		{/if}
	</div>

	<OfflineSicherungenEinspielen />
</div>
