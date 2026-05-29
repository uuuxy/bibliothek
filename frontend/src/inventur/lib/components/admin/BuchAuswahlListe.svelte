<script>
    import { SvelteSet } from "svelte/reactivity";
    import { sortBooksBySubjectAndTitle } from "$lib/book_sorting.js";

    let { allBooks, selectedBookIds = $bindable() } = $props();
    let searchQuery = $state("");

    let filteredBooks = $derived(
        [...allBooks].filter(
            (b) =>
                b.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                b.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
                (b.gradeLevel && b.gradeLevel.toString().includes(searchQuery.toLowerCase())) ||
                (b.track && b.track.toLowerCase().includes(searchQuery.toLowerCase())),
        ).sort(sortBooksBySubjectAndTitle)
    );

    /**
     * @param {string|number} id
     */
    function toggleBook(id) {
        if (selectedBookIds.has(id)) {
            selectedBookIds.delete(id);
        } else {
            selectedBookIds.add(id);
        }
        selectedBookIds = new SvelteSet(selectedBookIds);
    }
</script>

<div class="mb-4 flex-1 flex flex-col min-h-[300px]">
    <label
        for="bookSearchInput"
        class="block text-sm font-medium text-gray-700 mb-2"
        >Bücher auswählen ({selectedBookIds.size} ausgewählt)</label
    >
    <input
        id="bookSearchInput"
        type="text"
        bind:value={searchQuery}
        class="w-full px-4 py-2 border border-gray-300 rounded-lg mb-4 text-sm shrink-0"
        placeholder="Bücher suchen..."
    />

    <div
        class="grid grid-cols-1 sm:grid-cols-2 gap-3 overflow-y-auto p-1 flex-1"
    >
        {#each filteredBooks as book (book.id)}
            <button
                onclick={() => toggleBook(book.id)}
                aria-pressed={selectedBookIds.has(book.id)}
                class="flex items-center gap-3 p-3 rounded-xl border text-left transition-all {selectedBookIds.has(
                    book.id,
                )
                    ? 'border-emerald-500 bg-emerald-50 ring-1 ring-emerald-500'
                    : 'border-gray-200 hover:border-emerald-300 hover:bg-gray-50'}"
            >
                <div
                    class="w-5 h-5 rounded border flex items-center justify-center shrink-0 {selectedBookIds.has(
                        book.id,
                    )
                        ? 'bg-emerald-600 border-emerald-600'
                        : 'border-gray-300 bg-white'}"
                >
                    {#if selectedBookIds.has(book.id)}
                        <svg
                            class="w-3.5 h-3.5 text-white"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            ><path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="3"
                                d="M5 13l4 4L19 7"
                            /></svg
                        >
                    {/if}
                </div>
                
                <div class="w-10 h-14 bg-gray-100 rounded shrink-0 flex items-center justify-center overflow-hidden border border-gray-200">
                    {#if book.coverUrl || book.isbn}
                        <img 
                            src={book.coverUrl || `https://covers.openlibrary.org/b/isbn/${book.isbn}-S.jpg`} 
                            alt="Cover" 
                            loading="lazy"
                            class="w-full h-full object-cover"
                            onerror={(/** @type {Event} */ e) => {
                                const target = /** @type {HTMLImageElement} */ (e.target);
                                if (target) target.style.display = 'none';
                            }}
                        />
                    {:else}
                        <svg class="w-5 h-5 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                        </svg>
                    {/if}
                </div>

                <div class="min-w-0 flex-1">
                    <div class="font-medium text-sm text-gray-800 line-clamp-2 leading-tight">
                        {book.title}
                    </div>
                    <div class="text-xs text-gray-500 mt-1 flex flex-wrap gap-1 items-center">
                        <span class="bg-gray-100 px-1.5 py-0.5 rounded">{book.subject}</span>
                        {#if book.gradeLevel}
                            <span class="bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded">Kl. {book.gradeLevel}</span>
                        {/if}
                        {#if book.track}
                            <span class="bg-emerald-50 text-emerald-600 px-1.5 py-0.5 rounded">{book.track}</span>
                        {/if}
                    </div>
                </div>
            </button>
        {/each}
    </div>
</div>

