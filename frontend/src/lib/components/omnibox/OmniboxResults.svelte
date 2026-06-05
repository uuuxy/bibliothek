<script>
  import StudentResultItem from "./StudentResultItem.svelte";
  import BookResultItem from "./BookResultItem.svelte";

  let { unifiedSearchResults, selectedDropdownIndex, onSelect } = $props();
</script>

<div id="omnibox-dropdown" role="listbox" aria-label="Suchergebnisse" class="absolute top-full left-0 right-0 mt-4 bg-white/80 backdrop-blur-2xl border border-white/60 shadow-[0_12px_40px_rgb(0,0,0,0.12)] rounded-2xl z-50 overflow-hidden flex flex-col max-h-[60vh] animate-slide-up">
  <div class="overflow-y-auto overscroll-contain flex-1 p-3 space-y-4">
    {#if unifiedSearchResults.students.length > 0}
      <div>
        <div class="px-3 pb-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider">Schüler ({unifiedSearchResults.students.length})</div>
        <div class="space-y-1">
          {#each unifiedSearchResults.students as student, i}
            <StudentResultItem 
              {student} 
              index={i} 
              selected={selectedDropdownIndex === i} 
              {onSelect} 
            />
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
            <BookResultItem 
              {book} 
              {index} 
              selected={selectedDropdownIndex === index} 
              {onSelect} 
            />
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>
