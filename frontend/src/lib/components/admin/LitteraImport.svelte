<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { nimmDatei } from '../../utils/dateiwahl.js';
	import { anzahlText } from '../../utils/format.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	let datei = $state(/** @type {File | null} */ (null));
	let isImporting = $state(false);

	async function ladeHoch() {
		if (!datei) return;

		isImporting = true;

		const formulardaten = new FormData();
		formulardaten.append('file', datei);

		try {
			const res = await apiFetch('/api/import/littera', {
				method: 'POST',
				body: formulardaten
			});

			if (!res.ok) throw new Error(await leseApiFehler(res, 'Import fehlgeschlagen'));
			const daten = await res.json();

			snackbars.addToast(
				daten.type === 'xml'
					? `${anzahlText(daten.updated_titles_count, 'vorhandener Titel', 'vorhandene Titel')} aktualisiert`
					: `${anzahlText(daten.new_titles_count, 'neuer Titel', 'neue Titel')} und ${anzahlText(daten.imported_copies_count, 'Exemplar', 'Exemplare')} angelegt`,
				'success'
			);
			datei = null;
		} catch (err) {
			snackbars.addToast(
				(err instanceof Error && err.message) || 'Ein unerwarteter Fehler ist aufgetreten',
				'error'
			);
		} finally {
			isImporting = false;
		}
	}
</script>

<!-- Der erste Baustein der Gruppe, deshalb ohne Trennlinie darüber; sonst gebaut wie
     BestandImport und ListenImport. -->
<div>
	<h4 class="mb-1 text-title-small text-on-surface">Katalog-Import (Littera)</h4>
	<p class="mb-4 text-sm text-on-surface-variant">
		Die Datei <strong>katalogisat.xml</strong> aus Littera ergänzt die Angaben vorhandener Titel.
		Eine
		<strong>CSV- oder Excel-Datei</strong> legt neue Titel und Exemplare an.
	</p>

	<div class="flex items-center gap-4">
		<label class="relative {isImporting ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
			<input
				type="file"
				accept=".xml,.csv,.xlsx"
				onchange={(e) => (datei = nimmDatei(e))}
				disabled={isImporting}
				class="sr-only"
			/>
			<div
				class="px-5 py-2.5 bg-surface-container hover:bg-surface-container-high text-on-surface text-body-medium-emphasized rounded-xl transition-colors border border-outline-variant inline-block"
			>
				{datei ? datei.name : 'Datei auswählen …'}
			</div>
		</label>

		<Button size="lg" onclick={ladeHoch} disabled={isImporting || !datei} class="px-6">
			{#if isImporting}
				<Ladekreis size="sm" farbe="aktuell" />
				<span>Wird importiert …</span>
			{:else}
				<span>Import starten</span>
			{/if}
		</Button>
	</div>
</div>
