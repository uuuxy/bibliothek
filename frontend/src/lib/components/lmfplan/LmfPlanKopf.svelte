<!-- @component LmfPlanKopf — die Kopfzeile des Planers: Art des Plans, die Aktionen und
     zu einem gespeicherten Plan der eine Satz, ob das Kollegium ihn sieht (Entwurf oder
     veröffentlicht). Gespeichert wird ein Entwurf;
     „Veröffentlichen" (Migration 100) macht ihn für Portal und PDF des Kollegiums
     sichtbar und setzt beim Büchertausch die Fristen. Vorher nimmt die Schulleitung
     ihn ab, dafür das PDF. Die Aktionen erscheinen nur, wenn ein
     Stand geladen ist — „Plan speichern" auf einem gescheiterten Laden würde den echten
     Plan durch den Entwurf ersetzen (ui/LadeFehler.svelte).

     Die Leiste haftet beim Scrollen oben (M3 Top App Bar: die Aktionen der Seite
     bleiben erreichbar, während der Inhalt darunter durchläuft) — bei 60 Zeilen lag
     „Plan speichern" sonst zwei Bildschirmhöhen über der letzten Änderung. Der
     Scroll-Container ist <main> (App.svelte), deshalb genügt sticky top-0. -->
<script>
	import { Printer, Send, Trash } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import { ARTEN, datumKurz } from './lmfplanDienst.js';

	/** @type {{ art: string, stand: any, laedt: boolean, ladeFehler: boolean, gueltig: boolean, speichert: boolean, onart: (a: string) => void, onpdf: () => void, onverwerfen: () => void, onspeichern: () => void, onveroeffentlichen: () => void }} */
	let {
		art,
		stand,
		laedt,
		ladeFehler,
		gueltig,
		speichert,
		onart,
		onpdf,
		onverwerfen,
		onspeichern,
		onveroeffentlichen
	} = $props();

	const laufend = $derived(Boolean(stand?.plan) && !stand.vorbei);
	const veroeffentlicht = $derived(laufend && Boolean(stand.plan.veroeffentlicht_am));
</script>

<div class="sticky top-0 z-10 -mt-6 bg-surface-container-lowest pt-6 pb-3">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<Segmente
			etikett="Art des Plans"
			optionen={ARTEN.map((a) => ({ wert: a.wert, text: a.label }))}
			wert={art}
			onwahl={onart}
		/>
		<!-- Während des Ladens keine Aktionen: Sonst schickte „Plan verwerfen" das DELETE an die
		     neue Art, entschiede seine Sperre aber am alten Plan, und der Bestätigungsdialog
		     zeigte dessen Datum. -->
		{#if !ladeFehler && !laedt}
			<div class="flex items-center gap-2">
				<Button variant="secondary" onclick={onpdf}>
					<Printer class="h-4 w-4" aria-hidden="true" />
					Als PDF
				</Button>
				<Button
					variant="secondary"
					onclick={onverwerfen}
					disabled={!stand?.plan || stand.vorbei || speichert}
				>
					<Trash class="h-4 w-4" aria-hidden="true" />
					Plan verwerfen
				</Button>
				{#if laufend && !veroeffentlicht}
					<Button variant="secondary" onclick={onveroeffentlichen} disabled={!gueltig || speichert}>
						<Send class="h-4 w-4" aria-hidden="true" />
						Veröffentlichen
					</Button>
				{/if}
				<Button onclick={onspeichern} disabled={!gueltig || speichert}>Plan speichern</Button>
			</div>
		{/if}
	</div>

	{#if !laedt && !ladeFehler && laufend}
		<!-- Nur dieser Satz zeigt, ob das Kollegium den Plan sieht; der Knopf „Veröffentlichen"
		     fehlt danach einfach. Beim Vorschlag steht keiner: Gespeichert ist nichts, und
		     wer Geändertes verlassen will, wird gefragt (verlassenSchutz). -->
		<p class="mt-3 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-plan-hinweis">
			{#if veroeffentlicht}
				Veröffentlicht am {datumKurz(stand.plan.veroeffentlicht_am.slice(0, 10))}.
			{:else}
				Entwurf, noch nicht veröffentlicht.
			{/if}
		</p>
	{/if}
</div>
