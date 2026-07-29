<script>
	let { book, index, selected, onSelect } = $props();
</script>

<!-- Gleiches Spaltenraster wie StudentResultItem, damit beide Gruppen im selben
     Dropdown auf einer Kante stehen. Und dieselbe Primärfarbe: Vorher war die
     Auswahl bei Schülern blau und bei Büchern indigo — zwei Akzente für dieselbe
     Interaktion in derselben Liste.
     Die Signatur steht vor dem Autor, weil sie beim Suchen im Regal die Frage
     beantwortet; das Cover ist raus, es hielt die Zeile auf und trug nichts bei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="Buch: {book.titel} von {book.autor}, Signatur {book.signatur || 'keine'}"
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
	<span class="truncate font-medium">{book.titel}</span>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}"
		>{book.signatur}</span
	>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}">{book.autor}</span>
</div>
