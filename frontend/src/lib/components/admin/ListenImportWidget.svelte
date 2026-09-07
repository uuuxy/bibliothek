<!--
  @component ListenImportWidget
  Listenimport: CSV oder Excel mit ISBN und Stückzahl → neue Titel samt Exemplaren.

  Der Weg existierte seit dem 21.06.2026 nur noch als Backend-Route (POST /api/books/import):
  Sein Knopf saß in der Titel-Verwaltung und ging beim Entschlacken der Toolbar verloren,
  ohne Ersatz (Befund-Register, 07.09.2026). Jetzt hier, neben den anderen Importen.
  Anders als Katalog- und Bestands-Import ERZEUGT er Exemplare mit frischen B-Nummern aus
  barcode_seq — für Sammelkäufe und Spenden, die nicht durchs Bestellwesen laufen. Die
  Exemplare stehen danach im Druck-Center unter „Fehlende Etiketten".
-->
<script lang="ts">
	import { importiereListe } from '../../../inventur/lib/admin_api.js';
	import Button from '../ui/Button.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';

	// Die Route verlangt edit_books (inventur/api_routen.go); die Karte darüber zeigt sich
	// schon bei manage_inventory. Ohne eigene Prüfung sähe jemand einen Knopf, der 403 liefert.
	const darf = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	let files: FileList | null = $state(null);
	let laeuft = $state(false);
	let ergebnis: { type: 'success' | 'error'; message: string } | null = $state(null);

	async function importieren() {
		if (!files || files.length === 0) return;
		laeuft = true;
		ergebnis = null;
		try {
			const r = await importiereListe(files[0]);
			const fehl = r.failed > 0 ? `, ${r.failed} fehlgeschlagen` : '';
			ergebnis = { type: 'success', message: `${r.imported} Titel importiert${fehl}.` };
			files = null;
		} catch (err) {
			ergebnis = {
				type: 'error',
				message: (err instanceof Error && err.message) || 'Import fehlgeschlagen.'
			};
		} finally {
			laeuft = false;
		}
	}
</script>

{#if darf}
	<div class="pt-6 border-t border-outline-variant">
		<h4 class="text-sm font-bold text-on-surface mb-1">Listenimport (ISBN + Stückzahl)</h4>
		<p class="text-xs text-on-surface-variant mb-4">
			CSV oder Excel mit den Spalten ISBN (Pflicht), Titel, Autor, Fach, Klasse und Bestand — ohne
			Kopfzeile werden die Spalten am Inhalt erkannt. Je Zeile entsteht ein Titel mit so vielen
			Exemplaren, wie im Bestand steht, jedes mit einer neuen B-Nummer; die Etiketten stehen danach
			im Druck-Center unter „Fehlende Etiketten". Für Bücher, die schon eine Nummer tragen, ist der
			Bestands-Import der richtige Weg.
		</p>

		<div class="flex items-center gap-4">
			<label class="relative {laeuft ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
				<input
					type="file"
					accept=".csv,.xlsx"
					bind:files
					disabled={laeuft}
					class="sr-only"
					data-testid="listenimport-datei"
				/>
				<div
					class="px-5 py-2.5 bg-surface-container hover:bg-surface-container-high text-on-surface font-semibold text-sm rounded-xl transition-colors border border-outline-variant inline-block"
				>
					{files && files.length > 0 ? files[0].name : 'CSV- oder Excel-Datei auswählen...'}
				</div>
			</label>

			<Button
				size="lg"
				onclick={importieren}
				disabled={laeuft || !files || files.length === 0}
				class="px-6"
			>
				{#if laeuft}
					<div
						class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
					></div>
					<span>Importiere Liste...</span>
				{:else}
					<span>Liste importieren</span>
				{/if}
			</Button>
		</div>

		{#if ergebnis}
			<div
				class="mt-4 p-4 rounded-xl text-sm font-semibold {ergebnis.type === 'error'
					? 'bg-error-container text-on-error-container'
					: 'bg-primary-container text-on-primary-container'}"
				data-testid="listenimport-ergebnis"
			>
				{ergebnis.message}
			</div>
		{/if}
	</div>
{/if}
