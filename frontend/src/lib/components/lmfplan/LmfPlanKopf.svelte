<!-- @component LmfPlanKopf — die Kopfzeile des Planers: Art des Plans, die Aktionen und
     der eine Satz, der sagt, woran man gerade arbeitet (Entwurf, veröffentlichter Plan,
     Entwurf für den nächsten, oder der erste überhaupt). Gespeichert wird ein ENTWURF;
     „Veröffentlichen" (Migration 100) macht ihn für Portal und PDF des Kollegiums
     sichtbar und setzt beim Büchertausch die Fristen — vorher nimmt die Schulleitung
     ihn ab, dafür das PDF (Peter, 06.09.2026). Die Aktionen erscheinen nur, wenn ein
     Stand geladen ist — „Plan speichern" auf einem gescheiterten Laden würde den echten
     Plan durch den Entwurf ersetzen (ui/LadeFehler.svelte). -->
<script>
	import { Printer, Send, Trash2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import { ARTEN, artErklaerung, datumKurz } from '../../lmfplanDienst.js';

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

<div class="flex flex-wrap items-center justify-between gap-3">
	<Segmente
		etikett="Art des Plans"
		optionen={ARTEN.map((a) => ({ wert: a.wert, text: a.label }))}
		wert={art}
		onwahl={onart}
	/>
	{#if !ladeFehler}
		<div class="flex items-center gap-2">
			<Button variant="secondary" onclick={onpdf}>
				<Printer class="h-4 w-4" aria-hidden="true" />
				Als PDF
			</Button>
			<Button variant="secondary" onclick={onverwerfen} disabled={!stand?.plan || stand.vorbei}>
				<Trash2 class="h-4 w-4" aria-hidden="true" />
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

{#if !laedt && !ladeFehler}
	<p class="mt-3 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-plan-hinweis">
		{#if veroeffentlicht}
			Plan vom {datumKurz(stand.plan.erster_tag)}, veröffentlicht am {datumKurz(
				stand.plan.veroeffentlicht_am.slice(0, 10)
			)} — Änderungen gelten sofort nach „Plan speichern".
		{:else if laufend}
			Entwurf vom {datumKurz(stand.plan.erster_tag)} — nur hier sichtbar, nicht im Portal. „Als PDF" für
			die Abnahme durch die Schulleitung, dann „Veröffentlichen".
		{:else if stand?.plan && stand.vorbei}
			Der Plan vom {datumKurz(stand.plan.erster_tag)} ist vorbei. Dieser Entwurf übernimmt seine Reihenfolge
			— ersten Tag wählen, prüfen, speichern, veröffentlichen.
		{:else if art === 'ausgabe'}
			Noch kein Plan. Vorgeschlagen sind die Klassen der Eingangsjahrgänge, Jahrgang absteigend;
			alle anderen stehen unter „Nicht im Plan".
		{:else}
			Noch kein Plan. Die Reihenfolge folgt der Regel: Abschlussklassen zuerst, dann Jahrgang
			absteigend; die Oberstufe steht unter „Nicht im Plan".
		{/if}
		{artErklaerung(art, stand?.eingangsjahrgaenge)}
		{#if art === 'rueckgabe'}
			Mit dem Veröffentlichen wird der Termin einer Klasse die Frist ihrer Schulbücher.
		{/if}
	</p>
{/if}
