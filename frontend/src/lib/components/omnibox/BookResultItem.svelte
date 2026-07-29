<script>
	let { book, index, selected, onSelect } = $props();
</script>

<!-- Wie StudentResultItem: nackte Datenzeile. Das Cover behält seine Fläche, aber
     weder Rahmen noch Schatten noch Radius — es ist eine Abbildung, kein Objekt. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="Buch: {book.titel} von {book.autor}, ISBN {book.isbn || 'Keine ISBN'}"
	tabindex="-1"
	class="flex items-center gap-3 px-3 py-2 cursor-pointer {selected
		? 'bg-indigo-600 text-white'
		: 'text-slate-700 hover:bg-slate-100'}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	{#if book.cover_url}
		<img src={book.cover_url} class="w-8 h-11 shrink-0 object-cover" alt="Cover von {book.titel}" />
	{:else}
		<div class="w-8 h-11 shrink-0" aria-hidden="true"></div>
	{/if}
	<span class="font-medium line-clamp-1 {selected ? 'text-white' : 'text-slate-900'}">
		{book.titel}
	</span>
	<span class="text-xs line-clamp-1 {selected ? 'text-indigo-100' : 'text-slate-500'}">
		{book.autor} · {book.isbn || 'Keine ISBN'}
	</span>
</div>
