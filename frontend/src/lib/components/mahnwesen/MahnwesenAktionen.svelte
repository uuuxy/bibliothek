<!--
  @component MahnwesenAktionen
  Die Knopfzeile des Mahnwesens. Sie steht in der Werkzeugzeile UNTER der Suchpille
  (MahnwesenSuchleiste), rechts neben dem Klassenfilter — so wie „Neuer Schueler" in
  StudentDirectoryToolbar.

  Bis zum 04.09.2026 stand sie GANZ OBEN im `aktionen`-Slot von PageShell, weil sie dort
  einmal neben dem Seitentitel gestanden hatte. Den Seitentitel hat 68c4810 am 08.08.2026
  abgeschafft, die Knopfzeile blieb — als einzige von sechzehn Routen — allein oben stehen
  und schob die Suchpille um 84 px nach unten. Den Slot gibt es seitdem nicht mehr.

  Bei einer Auswahl uebernimmt sie den Auswahl-Modus (wie Gmail/Drive): Nur noch die auf
  die Markierung bezogenen Aktionen sind sichtbar.

  Den Mahnlauf-Dialog rendert NICHT diese Komponente, sondern Mahnwesen.svelte auf
  oberster Ebene: Ein Overlay hat in einem Flex-Container mit `print:hidden` nichts
  verloren.
-->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { baueMahnlisteDruckHtml } from '../../utils/mahnlisteDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from '../../utils/listenDruck.js';
	import { FileText, Mail, Printer, RefreshCw, X } from '@lucide/svelte';

	/**
	 * onBescheid bekommt die ID des einen markierten Schülers. Als Prop, weil den Dialog
	 * Mahnwesen.svelte auf oberster Ebene rendert — ein Overlay hat in dieser Flex-Zeile
	 * mit `print:hidden` nichts verloren (dieselbe Regel wie beim Mahnlauf-Dialog).
	 * @type {{ onMahnlauf: () => void, onBescheid: (schuelerId: string) => void, darfBescheid: boolean, darfMahnlauf: boolean }}
	 */
	let { onMahnlauf, onBescheid, darfBescheid, darfMahnlauf } = $props();

	// Der Bescheid ist ein Einzelfall, kein Massenlauf: Jeder Betrag ist eine
	// Ermessensentscheidung, und jede Referenznummer wird unwiderruflich verbraucht.
	// Deshalb erscheint der Knopf nur bei GENAU EINER Markierung.
	const einzelnMarkiert = $derived(
		mahnwesenStore.selectedIds.size === 1 ? [...mahnwesenStore.selectedIds][0] : ''
	);

	// Wen „Alle anmahnen" erreicht: die Kinder der Klassen, nicht die Ehemaligen.
	let countAlle = $derived(
		mahnwesenStore.versandKlassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.length,
			0
		)
	);

	// Druckt die Liste, wie Reiter, Klassenfilter und Suche sie gerade zeigen. Das Blatt
	// entsteht im Browser aus den geladenen Zeilen und zählt keine Mahnung.
	function listeDrucken() {
		const html = baueMahnlisteDruckHtml(mahnwesenStore.filteredSchueler, {
			ansicht: mahnwesenStore.activeFilter,
			klasse: mahnwesenStore.selectedKlasse,
			suche: mahnwesenStore.searchQuery
		});
		if (!druckeDokument(html)) toastStore.addToast(FENSTER_BLOCKIERT, 'warning');
	}
</script>

{#if mahnwesenStore.selectedIds.size > 0}
	<Button
		variant="secondary"
		onclick={mahnwesenStore.deselectAllSchueler}
		aria-label="Auswahl aufheben"
		title="Auswahl aufheben"
		class="px-2 text-on-surface-variant"
	>
		<X class="h-4 w-4" aria-hidden="true" />
	</Button>
	<span class="text-sm font-semibold text-on-surface"
		>{mahnwesenStore.selectedIds.size} ausgewählt</span
	>
	<Button onclick={mahnwesenStore.printSelectedMahnungen} disabled={mahnwesenStore.pdfLoading}>
		{#if mahnwesenStore.pdfLoading}
			<Ladekreis size="sm" farbe="aktuell" />
		{:else}
			<Printer class="h-4 w-4" aria-hidden="true" />
		{/if}
		Mahnbriefe drucken
	</Button>
	{#if darfBescheid && einzelnMarkiert}
		<!-- Getönt, nicht gefüllt: In diesem Bereich ist „Mahnbriefe drucken" die eine
		     gefüllte Aktion (M3). -->
		<Button variant="secondary" onclick={() => onBescheid(einzelnMarkiert)}>
			<FileText class="h-4 w-4" aria-hidden="true" />
			Schadensersatz-Bescheid
		</Button>
	{/if}
{:else}
	<Button
		variant="secondary"
		onclick={mahnwesenStore.fetchData}
		aria-label="Daten neu laden"
		data-tip="Daten neu laden"
		title="Neu laden"
		class="px-2 text-on-surface-variant"
	>
		<RefreshCw class="h-4 w-4" aria-hidden="true" />
	</Button>

	<Button
		variant="secondary"
		onclick={listeDrucken}
		disabled={mahnwesenStore.filteredSchueler.length === 0}
	>
		<Printer class="h-4 w-4" aria-hidden="true" />
		Liste drucken
	</Button>

	<!-- „Alle anmahnen" ist die einzige E-Mail-Aktion, nur hier steht das Umschlag-Symbol.
	     Nur mit dem Recht der Route dahinter (create_orders, entschieden in Mahnwesen.svelte);
	     Drucken bleibt — das hängt wie die Seite an view_students, Papier ist der Notweg. -->
	{#if darfMahnlauf && countAlle > 0}
		<Button
			variant="danger"
			onclick={() => onMahnlauf()}
			aria-label="Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden"
			class="shrink-0"
		>
			<Mail class="h-4 w-4" aria-hidden="true" />
			Alle anmahnen
		</Button>
	{/if}
{/if}
