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

	let files = $state(/** @type {FileList | null} */ (null));
	let laeuft = $state(false);
	let ergebnis = $state(
		/** @type {{ type: 'success' | 'error', message: string } | null} */ (null)
	);

	async function importieren() {
		if (!files || files.length === 0) return;
		laeuft = true;
		ergebnis = null;
		const formulardaten = new FormData();
		formulardaten.append('file', files[0]);
		try {
			const res = await apiFetch('/api/admin/import-bestand', {
				method: 'POST',
				body: formulardaten
			});
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Bestands-Import fehlgeschlagen'));
			const daten = await res.json();
			ergebnis = {
				type: 'success',
				message: `Kombi-Import erfolgreich! ${daten.new_titles_count || 0} neue Titel und ${daten.imported_copies_count || 0} Exemplare wurden verarbeitet.`
			};
			files = null;
		} catch (err) {
			ergebnis = {
				type: 'error',
				message: (err instanceof Error && err.message) || 'Ein unerwarteter Fehler ist aufgetreten.'
			};
		} finally {
			laeuft = false;
		}
	}
</script>

<div class="pt-6 border-t border-outline-variant">
	<h4 class="text-sm font-bold text-on-surface mb-1">Finaler Bestands-Import (Kombi-CSV)</h4>
	<p class="text-xs text-on-surface-variant mb-4">
		Laden Sie die finale Semikolon-separierte CSV hoch (Spalten:
		Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand). Die Exemplare behalten die Nummer aus
		der Datei und gelten als etikettiert.
	</p>

	<div class="flex items-center gap-4">
		<label class="relative {laeuft ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
			<input
				type="file"
				accept=".csv"
				bind:files
				disabled={laeuft}
				class="sr-only"
				data-testid="bestandimport-datei"
			/>
			<div
				class="px-5 py-2.5 bg-surface-container hover:bg-surface-container-high text-on-surface font-semibold text-sm rounded-xl transition-colors border border-outline-variant inline-block"
			>
				{files && files.length > 0 ? files[0].name : 'CSV-Datei auswählen...'}
			</div>
		</label>

		<Button
			size="lg"
			onclick={importieren}
			disabled={laeuft || !files || files.length === 0}
			class="px-6"
		>
			{#if laeuft}
				<Ladekreis size="sm" farbe="aktuell" />
				<span>Importiere Bestand...</span>
			{:else}
				<span>Bestand importieren</span>
			{/if}
		</Button>
	</div>

	{#if ergebnis}
		<div
			role={ergebnis.type === 'error' ? 'alert' : 'status'}
			class="mt-4 p-4 rounded-xl text-sm font-semibold {ergebnis.type === 'error'
				? 'bg-error-container text-on-error-container'
				: 'bg-success-container text-on-success-container'}"
			data-testid="bestandimport-ergebnis"
		>
			{ergebnis.message}
		</div>
	{/if}
</div>
