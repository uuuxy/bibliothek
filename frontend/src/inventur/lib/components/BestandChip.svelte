<!-- @component BestandChip — der Bestand eines Titels als Chip auf dem Cover: kurz „27 / 28" mit
     Punkt, für Hilfstechnik und in der Sprechblase der ganze Satz („27 von 28 verfügbar").
     Katalog- und Klassenkachel tragen ihn gleich, deshalb steht die Regel hier und nicht in
     jeder Kachel. Die Worte kommen aus format.js, weil Theke, Portal und Klassensätze denselben
     Bestand in denselben Worten nennen. -->
<script>
	import StatusChip from '../../../lib/components/ui/StatusChip.svelte';
	import { bestandKurz, bestandLage, bestandSatz } from '../../../lib/utils/format.js';

	/** @type {{ gesamt?: number, verfuegbar?: number, imZulauf?: number }} */
	let { gesamt, verfuegbar, imZulauf } = $props();

	/** Grün, wenn alles im Regal steht, gelb bei einem Teil. Nichts im Regal ist grau, nicht
	 *  rot: Im Schuljahr ist fast jedes Lernmittel verliehen, und ein überall roter Katalog
	 *  sagt nichts mehr. */
	const PUNKT = { alle: 'erfolg', teil: 'warten', keins: 'aus' };

	const kurz = $derived(bestandKurz(gesamt, verfuegbar, imZulauf));
	const satz = $derived(bestandSatz(gesamt, verfuegbar, imZulauf));
</script>

<!-- Ohne Zahlen im Payload gibt es keinen Chip. -->
{#if kurz}
	<StatusChip
		text={kurz}
		punkt={PUNKT[bestandLage(gesamt, verfuegbar)]}
		vorlesen={kurz === satz ? '' : satz}
		tip={satz}
	/>
{/if}
