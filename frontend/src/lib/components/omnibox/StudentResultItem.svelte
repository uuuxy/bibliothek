<script>
	import { leserArtText, istKollegium } from '../../leserArt.js';

	let { student, index, selected, onSelect } = $props();

	// Die mittlere Spalte sagt, WER da steht: bei einem Schüler die Klasse, bei einem
	// Kollegen seine Art. Ohne das stünde ein Kollege — er hat keine Klasse — in der
	// Liste wie ein Schüler mit fehlender Angabe.
	const kollege = $derived(istKollegium(student));
	const mitte = $derived(kollege ? leserArtText(student.art) : student.klasse);

	// Ein Kollege aus der Selbstanmeldung hat noch keine Ausweisnummer. Das gehört
	// hingeschrieben: Eine leere Spalte sähe nach einem Anzeigefehler aus.
	const ausweis = $derived(student.barcode_id || 'ohne Ausweis');
</script>

<!-- Ausgerichtete Spalten statt Fließtext: Art/Klasse und Ausweisnummer standen vorher
     direkt hinter dem Namen und damit in jeder Zeile an anderer Stelle. Bei
     namensgleichen Lesern (Hoffmann/Hofmann) ist genau das Untereinander die
     Entscheidungshilfe. Zeilenhöhe 48px wie die Scanleiste — ein Raster, nicht zwei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="{leserArtText(student.art)}: {student.vorname} {student.nachname}{kollege
		? ''
		: `, Klasse ${student.klasse}`}, Ausweis {ausweis}"
	tabindex="-1"
	class="grid grid-cols-[minmax(0,1fr)_6rem_11rem_10rem] items-center gap-4 px-4 h-12 cursor-pointer {selected
		? 'bg-blue-50 text-blue-900'
		: 'text-slate-900 hover:bg-slate-50'}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	<span class="truncate font-medium">{student.vorname} {student.nachname}</span>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}">{mitte}</span>
	<span
		class="text-sm truncate {student.barcode_id ? '' : 'italic'} {selected
			? 'text-blue-700'
			: 'text-slate-600'}">{ausweis}</span
	>
	<!-- Leere vierte Spalte: Sie gehört den Büchern (dort steht der Bestand, seit
	     17.09.2026). Ohne sie wären die Spalten der beiden Gruppen um 10rem gegeneinander
	     versetzt — das Raster ist der Grund, warum man hier untereinander lesen kann. -->
	<span></span>
</div>
