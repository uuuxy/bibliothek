<script>
	let { student, index, selected, onSelect } = $props();
</script>

<!-- Datenzeile, keine Karte: keine Radien, kein Schatten, keine Transition. Der
     Auswahlzustand wird allein über die Fläche getragen — ein zusätzlicher Schatten
     sagt nichts, was die Farbe nicht schon sagt. Bei Tastaturnavigation durch zehn
     Treffer ist jede Transition spürbare Latenz. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="Schüler: {student.vorname} {student.nachname}, Klasse {student.klasse}, Barcode {student.barcode_id}"
	tabindex="-1"
	class="flex items-baseline gap-3 px-3 py-2 cursor-pointer {selected
		? 'bg-blue-600 text-white'
		: 'text-slate-700 hover:bg-slate-100'}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	<span class="font-medium {selected ? 'text-white' : 'text-slate-900'}">
		{student.vorname}
		{student.nachname}
	</span>
	<span class="text-xs {selected ? 'text-blue-100' : 'text-slate-500'}">
		{student.klasse}
		{#if student.geburtsdatum}
			· {new Date(student.geburtsdatum).toLocaleDateString('de-DE')}{/if}
		· {student.barcode_id}
	</span>
</div>
