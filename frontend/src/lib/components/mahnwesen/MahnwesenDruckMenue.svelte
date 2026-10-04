<!-- @component MahnwesenDruckMenue — die Listen des Mahnwesens hinter einem Drucker-Knopf.

     Die Mahnbriefe kommen aus der Auswahl (Kinder anhaken, „Mahnbriefe drucken"). Hier
     steht, was daneben gedruckt wird und keine Mahnung zählt: die Liste einer Klasse, die
     Übersichtsliste und die Seite selbst.

     M3 (Icon buttons, Guidelines): „Default icon buttons can open other elements, such as
     a menu"; der Tooltip beschreibt die Handlung, nicht das Symbol. Das Menü ist
     ui/Menue.svelte mit der Klassenwahl für die Mahnliste als Kopf. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import Menue from '../ui/Menue.svelte';
	import { Download, Printer } from '@lucide/svelte';

	const ETIKETT = 'Weitere Druck- und Export-Optionen';

	/** @type {import('../ui/menueGeometrie.js').Eintrag[]} */
	const eintraege = $derived([
		{
			id: 'uebersicht',
			text: 'Übersichtsliste (PDF)',
			icon: Download,
			disabled: mahnwesenStore.pdfLoading,
			trennerDavor: true,
			ueberschriftDavor: 'Weitere'
		},
		{ id: 'drucken', text: 'Diese Seite drucken', icon: Printer }
	]);

	/** @param {string} id */
	function waehle(id) {
		if (id === 'uebersicht') mahnwesenStore.downloadPDF();
		else if (id === 'drucken') window.print();
	}
</script>

<Menue etikett={ETIKETT} {eintraege} onwahl={waehle} breite={280}>
	{#snippet ausloeser({ offen, umschalten })}
		<Button
			variant="secondary"
			onclick={umschalten}
			aria-haspopup="menu"
			aria-expanded={offen}
			aria-label={ETIKETT}
			data-tip={ETIKETT}
			class="px-2 text-on-surface-variant"
		>
			<Printer class="h-4 w-4" aria-hidden="true" />
		</Button>
	{/snippet}
	{#snippet kopf()}
		<div class="mb-1.5 pt-1 text-label-small font-medium text-on-surface-variant">
			Mahnliste einer Klasse
		</div>
		<div class="flex items-center gap-2">
			<Select
				bind:value={mahnwesenStore.selectedKlasse}
				options={mahnwesenStore.versandKlassen.map((/** @type {any} */ k) => ({
					value: k.klasse,
					label: k.klasse
				}))}
				placeholder="Klasse wählen …"
				class="flex-1 min-w-0"
				aria-label="Klasse für die Mahnliste"
			/>
			<Button
				size="sm"
				onclick={() => mahnwesenStore.downloadKlassePDF(mahnwesenStore.selectedKlasse)}
				disabled={mahnwesenStore.klassePdfLoading || !mahnwesenStore.selectedKlasse}
				class="shrink-0"
			>
				PDF
			</Button>
		</div>
	{/snippet}
</Menue>
