<script>
  let { book, index, selected, onSelect } = $props();
</script>

<div id="dropdown-item-{index}"
     role="option"
     aria-selected={selected}
     aria-label="Buch: {book.titel} von {book.autor}, ISBN {book.isbn || 'Keine ISBN'}"
     tabindex="-1"
     class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selected ? 'bg-indigo-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
     onclick={() => onSelect(index)}
     onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(index); } }}>
  <div class="flex items-center space-x-4">
    {#if book.cover_url}
      <img src={book.cover_url} class="w-10 h-14 object-cover rounded shadow-sm border {selected ? 'border-indigo-400' : 'border-slate-200'}" alt="Cover von {book.titel}" />
    {:else}
      <div class="w-10 h-14 rounded shadow-sm flex items-center justify-center font-bold text-xs {selected ? 'bg-indigo-500 text-white border border-indigo-400' : 'bg-slate-100 text-slate-400 border border-slate-200'}" aria-hidden="true">
        {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
      </div>
    {/if}
    <div>
      <div class="font-bold line-clamp-1 {selected ? 'text-white' : 'text-slate-900'}">{book.titel}</div>
      <div class="text-xs line-clamp-1 {selected ? 'text-indigo-100' : 'text-slate-500'}">{book.autor} · {book.isbn || 'Keine ISBN'}</div>
    </div>
  </div>
</div>
