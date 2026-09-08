<!-- @component MahnwesenDruckMenue — Split-Button „Mahnbriefe" mit Geltungsbereich.

     Eine Primäraktion (Eltern-Briefe drucken) plus ein Menü für alles, was denselben
     Vorgang mit anderem Umfang meint. Ersetzt die vier früher verstreuten PDF-Wege.
     Drucker- und Dokument-Symbole — kein Umschlag, denn hier wird nichts gemailt.

     Seit 07.09.2026 auf ui/Menue.svelte: der Split-Button als `ausloeser`, das
     Auswahlfeld „Ganze Klasse" als `kopf`. Vorher ein Eigenbau in Paletten-Farben ohne
     Pfeiltasten und mit handgeschriebenen SVG-Pfaden (Register 06.09.). -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import Menue from '../ui/Menue.svelte';
	import { ChevronDown, Download, FileText, Printer } from '@lucide/svelte';

	/** @type {import('../ui/menueGeometrie.js').Eintrag[]} */
	const eintraege = $derived([
		{
			id: 'eltern',
			text: 'Alle überfälligen',
			icon: FileText,
			disabled: mahnwesenStore.elternPdfLoading
		},
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
		if (id === 'eltern') mahnwesenStore.downloadElternPDF();
		else if (id === 'uebersicht') mahnwesenStore.downloadPDF();
		else if (id === 'drucken') window.print();
	}
</script>

<Menue etikett="Weitere Druck- und Export-Optionen" {eintraege} onwahl={waehle} breite={280}>
	{#snippet ausloeser({ offen, umschalten })}
		<div class="inline-flex rounded-md shadow-sm">
			<Button
				onclick={mahnwesenStore.downloadElternPDF}
				disabled={mahnwesenStore.elternPdfLoading}
				class="rounded-r-none"
			>
				{#if mahnwesenStore.elternPdfLoading}
					<Ladekreis size="sm" farbe="aktuell" />
				{:else}
					<FileText class="h-4 w-4" aria-hidden="true" />
				{/if}
				Mahnbriefe
			</Button>
			<Button
				onclick={umschalten}
				aria-haspopup="menu"
				aria-expanded={offen}
				aria-label="Weitere Druck- und Export-Optionen"
				data-tip="Weitere Druck- und Export-Optionen"
				class="rounded-l-none border-l-white/25 px-2"
			>
				<ChevronDown
					class="h-3.5 w-3.5 transition-transform {offen ? 'rotate-180' : ''}"
					aria-hidden="true"
				/>
			</Button>
		</div>
	{/snippet}
	{#snippet kopf()}
		<div class="pt-1 pb-1 text-label-small font-medium text-on-surface-variant">
			Mahnbriefe an Eltern
		</div>
		<div class="text-label-small text-on-surface-variant mb-1.5">Ganze Klasse</div>
		<div class="flex items-center gap-2">
			<Select
				bind:value={mahnwesenStore.selectedKlasse}
				options={mahnwesenStore.klassen.map((/** @type {any} */ k) => ({
					value: k.klasse,
					label: k.klasse
				}))}
				placeholder="Klasse wählen …"
				class="flex-1 min-w-0"
				aria-label="Klasse für den Sammel-Mahnlauf"
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
