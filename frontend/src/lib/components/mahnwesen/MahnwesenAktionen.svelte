<!--
  @component MahnwesenAktionen
  Die Knopfzeile des Mahnwesens. Sie steht in der Werkzeugzeile unter der Suchpille
  (MahnwesenSuchleiste), rechts neben dem Klassenfilter, so wie „Neuer Schüler" in
  SchuelerdateiLeiste: Über den Reitern schöbe sie die Suchpille als einzige Seite nach
  unten.

  Bei einer Auswahl uebernimmt sie den Auswahl-Modus (wie Gmail/Drive): Nur noch die auf
  die Markierung bezogenen Aktionen sind sichtbar.

  Den Mahnlauf-Dialog rendert nicht diese Komponente, sondern Mahnwesen.svelte auf
  oberster Ebene: Ein Overlay hat in einem Flex-Container mit `print:hidden` nichts
  verloren.
-->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { baueMahnlisteDruckHtml } from '../../utils/mahnlisteDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from '../../utils/listenDruck.js';
	import { FileText, Mail, Printer, RefreshCw, X } from '@lucide/svelte';

	/**
	 * onBescheid bekommt die ID des einen markierten Schülers. Als Prop, weil den Dialog
	 * Mahnwesen.svelte auf oberster Ebene rendert — ein Overlay hat in dieser Flex-Zeile
	 * mit `print:hidden` nichts verloren (dieselbe Regel wie beim Mahnlauf-Dialog).
	 * @type {{ onMahnlauf: () => void, onBescheid: (schuelerId: string) => void, darfBescheid: boolean, darfMahnen: boolean }}
	 */
	let { onMahnlauf, onBescheid, darfBescheid, darfMahnen } = $props();

	// Der Bescheid ist ein Einzelfall, kein Massenlauf: Jeder Betrag ist eine
	// Ermessensentscheidung, und jede Referenznummer wird unwiderruflich verbraucht.
	// Deshalb erscheint der Knopf nur bei GENAU EINER Markierung.
	const einzelnMarkiert = $derived(
		mahnwesenStore.gewaehlteKennungen.size === 1 ? [...mahnwesenStore.gewaehlteKennungen][0] : ''
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
		const html = baueMahnlisteDruckHtml(mahnwesenStore.gefilterteSchueler, {
			ansicht: mahnwesenStore.stufenFilter,
			klasse: mahnwesenStore.gewaehlteKlasse,
			suche: mahnwesenStore.suche
		});
		if (!druckeDokument(html)) snackbars.addToast(FENSTER_BLOCKIERT, 'warning');
	}
</script>

{#if mahnwesenStore.gewaehlteKennungen.size > 0}
	<Button
		variant="secondary"
		onclick={mahnwesenStore.waehleKeine}
		aria-label="Auswahl aufheben"
		data-tip="Auswahl aufheben"
		class="icon-btn px-2 text-on-surface-variant"
	>
		<X class="h-4 w-4" aria-hidden="true" />
	</Button>
	<span class="text-sm font-semibold text-on-surface"
		>{mahnwesenStore.gewaehlteKennungen.size} ausgewählt</span
	>
	<!-- Der Druck zählt die Mahnung: nur mit dem Recht der Route dahinter (create_orders). -->
	{#if darfMahnen}
		<Button onclick={mahnwesenStore.druckeGewaehlteMahnungen} disabled={mahnwesenStore.drucktPdf}>
			{#if mahnwesenStore.drucktPdf}
				<Ladekreis size="sm" farbe="aktuell" />
			{:else}
				<Printer class="h-4 w-4" aria-hidden="true" />
			{/if}
			Mahnbriefe drucken
		</Button>
	{/if}
	{#if darfBescheid && einzelnMarkiert}
		<!-- Getönt, nicht gefüllt: In diesem Bereich ist „Mahnbriefe drucken" die eine
		     gefüllte Aktion (M3). -->
		<Button variant="secondary" onclick={() => onBescheid(einzelnMarkiert)}>
			<FileText class="h-4 w-4" aria-hidden="true" />
			Schadensersatz-Bescheid
		</Button>
	{/if}
{:else}
	<!-- Wie in Verwaltungsprotokoll und Logbuch: Symbol und Wort. -->
	<Button variant="secondary" onclick={mahnwesenStore.lade}>
		<RefreshCw class="h-4 w-4" aria-hidden="true" />
		Aktualisieren
	</Button>

	<Button
		variant="secondary"
		onclick={listeDrucken}
		disabled={mahnwesenStore.gefilterteSchueler.length === 0}
	>
		<Printer class="h-4 w-4" aria-hidden="true" />
		Liste drucken
	</Button>

	<!-- „Alle anmahnen" ist die einzige E-Mail-Aktion, nur hier steht das Umschlag-Symbol.
	     Nur mit dem Recht der Route dahinter (create_orders, entschieden in Mahnwesen.svelte).
	     „Liste drucken" bleibt: Das Blatt entsteht im Browser und zählt keine Mahnung. Solange
	     ein Mahnlauf läuft, startet kein zweiter (sammelversand.svelte.js). -->
	{#if darfMahnen && countAlle > 0}
		<Button
			variant="danger"
			onclick={() => onMahnlauf()}
			disabled={mahnwesenStore.mahnlauf.laeuft}
			aria-label="Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden"
			class="shrink-0"
		>
			{#if mahnwesenStore.mahnlauf.laeuft}
				<Ladekreis size="sm" farbe="aktuell" etikett="Mahnlauf läuft" />
			{:else}
				<Mail class="h-4 w-4" aria-hidden="true" />
			{/if}
			Alle anmahnen
		</Button>
	{/if}
{/if}
