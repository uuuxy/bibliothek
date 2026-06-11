<script>
  import BorrowedBooksList from "./BorrowedBooksList.svelte";

  /** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void }} */
  let { books = [], onReturnClick = undefined, onDamageClick = undefined } = $props();
</script>

<div class="w-full h-full pt-2">
  <div class="flex items-center justify-between pb-3 border-b border-slate-100 mb-6">
    <h3 class="text-base font-bold text-slate-500 uppercase tracking-wider">Entliehene Bücher ({books?.length || 0})</h3>
  </div>

  {#if !books || books.length === 0}
    <div class="py-16 flex flex-col items-center justify-center text-slate-500 space-y-3">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
      <span class="text-sm font-semibold text-slate-400">Aktuell keine Bücher entliehen.</span>
    </div>
  {:else}
    <div class="relative border-l-2 border-slate-100 pl-4 ml-2 py-1">
      <BorrowedBooksList {books} {onReturnClick} {onDamageClick} mode="loans" />
    </div>
  {/if}
</div>
