<!--
  @component ListenImport
  Listenimport: CSV oder Excel mit ISBN und Stückzahl → neue Titel samt Exemplaren.

  Steht neben den anderen Importen (Route POST /api/books/import). Anders als Katalog- und
  Bestands-Import erzeugt er Exemplare mit frischen B-Nummern aus
  barcode_seq — für Sammelkäufe und Spenden, die nicht durchs Bestellwesen laufen. Die
  Exemplare stehen danach im Druck-Center unter „Fehlende Etiketten".
-->
<script>
	import { importiereListe } from '../../../inventur/lib/admin_api.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menue.js';
	import { nimmDatei } from '../../utils/dateiwahl.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	// Die Route verlangt edit_books (internal/api/routen_katalog.go); die Karte darüber zeigt sich
	// schon bei manage_inventory. Ohne eigene Prüfung sähe jemand einen Knopf, der 403 liefert.
	const darf = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	let datei = $state(/** @type {File | null} */ (null));
	let laeuft = $state(false);

	// Ein Schlüssel je Dateiauswahl, nicht je Klick: Bleibt die Auswahl nach einem
	// Netzfehler stehen, holt der nächste Klick das Ergebnis des ersten Laufs ab, statt
	// die Exemplare ein zweites Mal anzulegen. Eine neue Auswahl ist ein neuer Lauf.
	let schluessel = $state(crypto.randomUUID());

	async function importieren() {
		if (!datei) return;
		laeuft = true;
		try {
			const r = await importiereListe(datei, schluessel);
			const fehl = r.failed > 0 ? `, ${r.failed} fehlgeschlagen` : '';
			snackbars.addToast(`${r.imported} Titel importiert${fehl}.`, 'success');
			datei = null;
			schluessel = crypto.randomUUID();
		} catch (err) {
			snackbars.addToast(
				(err instanceof Error && err.message) || 'Import fehlgeschlagen.',
				'error'
			);
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
					onchange={(e) => {
						datei = nimmDatei(e);
						schluessel = crypto.randomUUID();
					}}
					disabled={laeuft}
					class="sr-only"
					data-testid="listenimport-datei"
				/>
				<div
					class="px-5 py-2.5 bg-surface-container hover:bg-surface-container-high text-on-surface font-semibold text-sm rounded-xl transition-colors border border-outline-variant inline-block"
				>
					{datei ? datei.name : 'CSV- oder Excel-Datei auswählen …'}
				</div>
			</label>

			<Button size="lg" onclick={importieren} disabled={laeuft || !datei} class="px-6">
				{#if laeuft}
					<Ladekreis size="sm" farbe="aktuell" />
					<span>Wird importiert …</span>
				{:else}
					<span>Liste importieren</span>
				{/if}
			</Button>
		</div>
	</div>
{/if}
