<script>
	let { student, index, selected, onSelect } = $props();
</script>

<!-- Ausgerichtete Spalten statt Fließtext: Klasse und Ausweisnummer standen vorher
     direkt hinter dem Namen und damit in jeder Zeile an anderer Stelle. Bei
     namensgleichen Schülern (Hoffmann/Hofmann) ist genau das Untereinander die
     Entscheidungshilfe. Zeilenhöhe 48px wie die Scanleiste — ein Raster, nicht zwei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="Schüler: {student.vorname} {student.nachname}, Klasse {student.klasse}, Ausweis {student.barcode_id}"
	tabindex="-1"
	class="grid grid-cols-[minmax(0,1fr)_5rem_11rem] items-center gap-4 px-4 h-12 cursor-pointer {selected
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
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}"
		>{student.klasse}</span
	>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}"
		>{student.barcode_id}</span
	>
</div>
