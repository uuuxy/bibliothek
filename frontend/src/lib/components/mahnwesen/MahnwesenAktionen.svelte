<!--
  @component MahnwesenAktionen
  Die rechte Aktionsleiste des Mahnwesens. Herausgeloest aus MahnwesenFilters, damit sie
  in den `aktionen`-Slot von PageShell passt — dort steht sie neben dem Seitentitel, so
  wie sie es vorher in der handgebauten Kopfzeile tat.

  Bei einer Auswahl uebernimmt sie den Auswahl-Modus (wie Gmail/Drive): Nur noch die auf
  die Markierung bezogenen Aktionen sind sichtbar.

  Den Mahnlauf-Dialog rendert NICHT diese Komponente, sondern Mahnwesen.svelte auf
  oberster Ebene: Ein Overlay hat in einem Flex-Container mit `print:hidden` nichts
  verloren.
-->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Button from '../ui/Button.svelte';
	import MahnwesenDruckMenue from './MahnwesenDruckMenue.svelte';
	import { Mail, Printer, RefreshCw, X } from '@lucide/svelte';

	/** @type {{ onMahnlauf: () => void }} */
	let { onMahnlauf } = $props();

	let countAlle = $derived(
		mahnwesenStore.klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.length,
			0
		)
	);
</script>

{#if mahnwesenStore.selectedIds.size > 0}
	<Button
		variant="secondary"
		onclick={mahnwesenStore.deselectAllSchueler}
		aria-label="Auswahl aufheben"
		title="Auswahl aufheben"
		class="px-2 text-slate-500 hover:text-slate-700"
	>
		<X class="h-4 w-4" aria-hidden="true" />
	</Button>
	<span class="text-sm font-semibold text-slate-700"
		>{mahnwesenStore.selectedIds.size} ausgewählt</span
	>
	<Button onclick={mahnwesenStore.printSelectedMahnungen} disabled={mahnwesenStore.pdfLoading}>
		{#if mahnwesenStore.pdfLoading}
			<div
				class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
			></div>
		{:else}
			<Printer class="h-4 w-4" aria-hidden="true" />
		{/if}
		Mahnbriefe drucken
	</Button>
{:else}
	<div class="flex items-center gap-1 bg-slate-100 p-1 rounded-xl">
		<button
			class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
			'datum'
				? 'bg-white text-slate-800 shadow-sm'
				: 'text-slate-500 hover:text-slate-700'}"
			onclick={() => {
				mahnwesenStore.mahnMode = 'datum';
				mahnwesenStore.fetchData();
			}}
		>
			Datum
		</button>
		<button
			class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
			'jahrgang'
				? 'bg-white text-slate-800 shadow-sm'
				: 'text-slate-500 hover:text-slate-700'}"
			onclick={() => {
				mahnwesenStore.mahnMode = 'jahrgang';
				mahnwesenStore.fetchData();
			}}
		>
			Jahrgang
		</button>
	</div>

	<Button
		variant="secondary"
		onclick={mahnwesenStore.fetchData}
		aria-label="Daten neu laden"
		data-tip="Daten neu laden"
		title="Neu laden"
		class="px-2 text-slate-500 hover:text-slate-700"
	>
		<RefreshCw class="h-4 w-4" aria-hidden="true" />
	</Button>

	<MahnwesenDruckMenue />

	<!-- „Alle anmahnen" ist die EINZIGE echte E-Mail-Aktion → nur hier das Umschlag-Icon.
	     Getönt statt gefüllt (M3: EIN gefüllter Knopf je Bereich — das ist „Mahnbriefe");
	     bis 24.08.2026 standen zwei gefüllte nebeneinander, blau und rot. -->
	{#if countAlle > 0}
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
