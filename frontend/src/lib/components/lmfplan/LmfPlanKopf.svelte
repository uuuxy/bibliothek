<!-- @component LmfPlanKopf — die Kopfzeile des Planers: Art des Plans, die Aktionen und
     der eine Satz, der sagt, woran man gerade arbeitet (laufender Plan, Entwurf für den
     nächsten, oder der erste überhaupt). Die Aktionen erscheinen nur, wenn ein Stand
     geladen ist — „Plan speichern" auf einem gescheiterten Laden würde den echten Plan
     durch den Entwurf ersetzen (ui/LadeFehler.svelte). -->
<script>
	import { Printer, Trash2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import { ARTEN, datumKurz } from '../../lmfplanDienst.js';

	/** @type {{ art: string, stand: any, laedt: boolean, ladeFehler: boolean, gueltig: boolean, speichert: boolean, onart: (a: string) => void, onpdf: () => void, onverwerfen: () => void, onspeichern: () => void }} */
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
		onspeichern
	} = $props();
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
			<Button onclick={onspeichern} disabled={!gueltig || speichert}>Plan speichern</Button>
		</div>
	{/if}
</div>

{#if !laedt && !ladeFehler}
	<p class="mt-3 text-sm text-on-surface-variant" data-testid="lmf-plan-hinweis">
		{#if stand?.plan && !stand.vorbei}
			Plan vom {datumKurz(stand.plan.erster_tag)} — Änderungen gelten nach „Plan speichern".
		{:else if stand?.plan && stand.vorbei}
			Der Plan vom {datumKurz(stand.plan.erster_tag)} ist vorbei. Dieser Entwurf übernimmt seine Reihenfolge
			— ersten Tag wählen, prüfen, speichern.
		{:else}
			Noch kein Plan. Die Reihenfolge folgt der Regel: Abschlussklassen zuerst, dann Jahrgang
			absteigend; die Oberstufe steht unter „Nicht im Plan".
		{/if}
		{#if art === 'rueckgabe'}
			Der Rückgabe-Termin einer Klasse wird die Frist ihrer Schulbücher.
		{/if}
	</p>
{/if}
