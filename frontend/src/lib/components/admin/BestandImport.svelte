<!--
  @component BestandImport
  Finaler Bestands-Import (Kombi-CSV): übernimmt vorhandene Exemplare samt ihrer Nummern.

  apiFetch statt fetch: setzt den CSRF-Token selbst und gibt Uploads fünf Minuten statt zehn
  Sekunden (frontend/src/lib/apiFetch.js).
-->
<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { nimmDatei } from '../../utils/dateiwahl.js';
	import { anzahlText } from '../../utils/format.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	let datei = $state(/** @type {File | null} */ (null));
	let dateiFeld = $state(/** @type {HTMLInputElement | undefined} */ (undefined));
	let laeuft = $state(false);

	async function importieren() {
		if (!datei) return;
		laeuft = true;
		const formulardaten = new FormData();
		formulardaten.append('file', datei);
		try {
			const res = await apiFetch('/api/admin/import-bestand', {
				method: 'POST',
				body: formulardaten
			});
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Bestands-Import fehlgeschlagen'));
			const daten = await res.json();
			snackbars.addToast(
				`Bestand übernommen: ${anzahlText(daten.new_titles_count, 'neuer Titel', 'neue Titel')}, ${anzahlText(daten.imported_copies_count, 'Exemplar', 'Exemplare')}`,
				'success'
			);
			datei = null;
		} catch (err) {
			snackbars.addToast(
				(err instanceof Error && err.message) || 'Ein unerwarteter Fehler ist aufgetreten',
				'error'
			);
		} finally {
			laeuft = false;
		}
	}
</script>

<div class="pt-6 border-t border-outline-variant">
	<h4 class="text-title-small text-on-surface mb-1">Bestand mit vorhandenen Nummern</h4>
	<p class="text-sm text-on-surface-variant mb-4">
		Für Bücher, die schon eine Nummer tragen: eine CSV-Datei mit Semikolon als Trenner und einer
		Kopfzeile mit den Spalten Titel, Autor, Verlag, ISBN, Jahr, Kategorie, Barcode und Zustand. Die
		Exemplare behalten ihre Nummer und gelten als etikettiert.
	</p>

	<div class="flex items-center gap-4">
		<input
			bind:this={dateiFeld}
			type="file"
			accept=".csv"
			onchange={(e) => (datei = nimmDatei(e))}
			disabled={laeuft}
			class="sr-only"
			tabindex="-1"
			data-testid="bestandimport-datei"
		/>
		<Button variant="secondary" onclick={() => dateiFeld?.click()} disabled={laeuft}>
			{datei ? datei.name : 'CSV-Datei auswählen …'}
		</Button>

		<Button size="lg" onclick={importieren} disabled={laeuft || !datei} class="px-6">
			{#if laeuft}
				<Ladekreis size="sm" farbe="aktuell" />
				<span>Wird importiert …</span>
			{:else}
				<span>Bestand importieren</span>
			{/if}
		</Button>
	</div>
</div>
