<script>
	import { leserArtText, istKollegium } from '../leser/leserArt.js';

	let { leser, index, gewaehlt, onSelect } = $props();

	// Die mittlere Spalte sagt, WER da steht: bei einem Schüler die Klasse, bei einem
	// Kollegen seine Art. Ohne das stünde ein Kollege — er hat keine Klasse — in der
	// Liste wie ein Schüler mit fehlender Angabe.
	const kollege = $derived(istKollegium(leser));
	const mitte = $derived(kollege ? leserArtText(leser.art) : leser.klasse);

	// Ein Kollege aus der Selbstanmeldung hat noch keine Ausweisnummer. Das gehört
	// hingeschrieben: Eine leere Spalte sähe nach einem Anzeigefehler aus.
	const ausweis = $derived(leser.barcode_id || 'ohne Ausweis');
</script>

<!-- Ausgerichtete Spalten statt Fließtext: Art/Klasse und Ausweisnummer standen vorher
     direkt hinter dem Namen und damit in jeder Zeile an anderer Stelle. Bei
     namensgleichen Lesern (Hoffmann/Hofmann) ist genau das Untereinander die
     Entscheidungshilfe. Zeilenhöhe 48px wie die Scanleiste — ein Raster, nicht zwei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={gewaehlt}
	aria-label="{leserArtText(leser.art)}: {leser.vorname} {leser.nachname}{kollege
		? ''
		: `, Klasse ${leser.klasse}`}, Ausweis {ausweis}"
	tabindex="-1"
	class="m3-state grid grid-cols-[minmax(0,1fr)_6rem_11rem_10rem] items-center gap-4 px-4 h-12 cursor-pointer text-on-surface {gewaehlt
		? 'bg-on-surface/8'
		: ''}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	<span class="truncate font-medium">{leser.vorname} {leser.nachname}</span>
	<span class="text-sm truncate text-on-surface-variant">{mitte}</span>
	<span class="text-sm truncate text-on-surface-variant {leser.barcode_id ? '' : 'italic'}"
		>{ausweis}</span
	>
	<!-- Leere vierte Spalte: Sie gehört den Büchern, dort steht der Bestand. Ohne sie wären
	     die Spalten der beiden Gruppen um 10rem gegeneinander versetzt; das Raster ist der
	     Grund, warum man hier untereinander lesen kann. -->
	<span></span>
</div>
