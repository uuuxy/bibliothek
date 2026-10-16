<script lang="ts">
	import { apiFetch } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import Button from './components/ui/Button.svelte';

	let files: FileList | null = $state(null);
	let isImporting = $state(false);
	let importErgebnis: { type: 'success' | 'error'; message: string } | null = $state(null);

	async function ladeHoch() {
		if (!files || files.length === 0) return;

		isImporting = true;
		importErgebnis = null;

		const formulardaten = new FormData();
		formulardaten.append('file', files[0]);

		try {
			const res = await apiFetch('/api/import/littera', {
				method: 'POST',
				body: formulardaten
			});

			const daten = await res.json();

			if (!res.ok) {
				throw new Error(daten.error || 'Upload fehlgeschlagen');
			}

			let successMsg = '';
			if (daten.type === 'xml') {
				successMsg = `XML-Import erfolgreich! ${daten.updated_titles_count} bestehende Titel wurden aktualisiert.`;
			} else if (daten.type === 'xlsx') {
				successMsg = `Excel-Import erfolgreich! ${daten.new_titles_count} neue Titel und ${daten.imported_copies_count} Exemplare wurden angelegt.`;
			} else {
				successMsg = `CSV-Import erfolgreich! ${daten.new_titles_count} neue Titel und ${daten.imported_copies_count} Exemplare wurden angelegt.`;
			}

			importErgebnis = {
				type: 'success',
				message: successMsg
			};
			files = null; // reset input
		} catch (err) {
			importErgebnis = {
				type: 'error',
				message: (err instanceof Error && err.message) || 'Ein unerwarteter Fehler ist aufgetreten.'
			};
		} finally {
			isImporting = false;
		}
	}
</script>

<!-- Der erste Baustein der Gruppe, deshalb ohne Trennlinie darüber; sonst gebaut wie
     BestandImport und ListenImport. -->
<div>
	<h4 class="mb-1 text-sm font-bold text-on-surface">Katalog-Import (Littera)</h4>
	<p class="mb-4 text-xs text-on-surface-variant">
		Lade hier die <strong>katalogisat.xml</strong> hoch (um bestehende Buch-Metadaten zu
		aktualisieren) oder eine <strong>CSV- bzw. XLSX-Datei</strong> (um neue Bücher/Exemplare per Bulk-Insert
		anzulegen).
	</p>

	<div class="flex items-center gap-4">
		<label class="relative {isImporting ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
			<input
				type="file"
				accept=".xml,.csv,.xlsx"
				bind:files
				disabled={isImporting}
				class="sr-only"
			/>
			<div
				class="px-5 py-2.5 bg-surface-container hover:bg-surface-container-high text-on-surface font-semibold text-sm rounded-xl transition-colors border border-outline-variant inline-block"
			>
				{files && files.length > 0 ? files[0].name : 'Datei auswählen...'}
			</div>
		</label>

		<Button
			size="lg"
			onclick={ladeHoch}
			disabled={isImporting || !files || files.length === 0}
			class="px-6"
		>
			{#if isImporting}
				<Ladekreis size="sm" farbe="aktuell" />
				<span>Importiere...</span>
			{:else}
				<span>Import starten</span>
			{/if}
		</Button>
	</div>

	{#if importErgebnis}
		<div
			class="mt-4 p-4 rounded-xl text-sm font-semibold {importErgebnis.type === 'error'
				? 'bg-error-container text-on-error-container'
				: 'bg-success-container text-on-success-container'}"
		>
			{importErgebnis.message}
		</div>
	{/if}
</div>
