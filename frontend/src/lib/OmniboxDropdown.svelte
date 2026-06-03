<script>
  let { unifiedSearchResults, selectedDropdownIndex, selectDropdownItem } = $props();
</script>

<div id="omnibox-dropdown" role="listbox" aria-label="Suchergebnisse" class="absolute top-full left-0 right-0 mt-4 bg-white/80 backdrop-blur-2xl border border-white/60 shadow-[0_12px_40px_rgb(0,0,0,0.12)] rounded-2xl z-50 overflow-hidden flex flex-col max-h-[60vh] animate-slide-up">
  <div class="overflow-y-auto overscroll-contain flex-1 p-3 space-y-4">
    {#if unifiedSearchResults.students.length > 0}
      <div>
        <div class="px-3 pb-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider">Schüler ({unifiedSearchResults.students.length})</div>
        <div class="space-y-1">
          {#each unifiedSearchResults.students as student, i}
            <div id="dropdown-item-{i}"
                 role="option"
                 aria-selected={selectedDropdownIndex === i}
                 aria-label="Schüler: {student.vorname} {student.nachname}, Klasse {student.klasse}, Barcode {student.barcode_id}"
                 tabindex="-1"
                 class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selectedDropdownIndex === i ? 'bg-blue-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
                 onclick={() => selectDropdownItem(i)}
                 onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectDropdownItem(i); } }}>
              <div class="flex items-center space-x-3">
                <div class="w-10 h-10 rounded-full flex items-center justify-center font-bold {selectedDropdownIndex === i ? 'bg-white/20 text-white' : 'bg-blue-100 text-blue-700'}" aria-hidden="true">
                  {student.vorname.charAt(0)}{student.nachname.charAt(0)}
                </div>
                <div>
                  <div class="font-bold {selectedDropdownIndex === i ? 'text-white' : 'text-slate-900'}">{student.vorname} {student.nachname}</div>
                  <div class="text-xs {selectedDropdownIndex === i ? 'text-blue-100' : 'text-slate-500'}">
                    {student.klasse}
                    {#if student.geburtsdatum} · {new Date(student.geburtsdatum).toLocaleDateString('de-DE')}{/if}
                    · {student.barcode_id}
                  </div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
    {#if unifiedSearchResults.books.length > 0}
      <div>
        <div class="px-3 pb-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider">Bücher ({unifiedSearchResults.books.length})</div>
        <div class="space-y-1">
          {#each unifiedSearchResults.books as book, j}
            {@const index = j + unifiedSearchResults.students.length}
            <div id="dropdown-item-{index}"
                 role="option"
                 aria-selected={selectedDropdownIndex === index}
                 aria-label="Buch: {book.titel} von {book.autor}, ISBN {book.isbn || 'Keine ISBN'}"
                 tabindex="-1"
                 class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selectedDropdownIndex === index ? 'bg-indigo-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
                 onclick={() => selectDropdownItem(index)}
                 onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectDropdownItem(index); } }}>
              <div class="flex items-center space-x-4">
                {#if book.cover_url}
                  <img src={book.cover_url} class="w-10 h-14 object-cover rounded shadow-sm border {selectedDropdownIndex === index ? 'border-indigo-400' : 'border-slate-200'}" alt="Cover von {book.titel}" />
                {:else}
                  <div class="w-10 h-14 rounded shadow-sm flex items-center justify-center font-bold text-xs {selectedDropdownIndex === index ? 'bg-indigo-500 text-white border border-indigo-400' : 'bg-slate-100 text-slate-400 border border-slate-200'}" aria-hidden="true">
                    {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
                  </div>
                {/if}
                <div>
                  <div class="font-bold line-clamp-1 {selectedDropdownIndex === index ? 'text-white' : 'text-slate-900'}">{book.titel}</div>
                  <div class="text-xs line-clamp-1 {selectedDropdownIndex === index ? 'text-indigo-100' : 'text-slate-500'}">{book.autor} · {book.isbn || 'Keine ISBN'}</div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>
