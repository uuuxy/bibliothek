<script>
    import KlassenBuchKachelStartseite from "$lib/components/KlassenBuchKachelStartseite.svelte";
    import { sortBooksBySubjectAndTitle } from "$lib/book_sorting.js";
    import { scrollCarousel, scrollHandler } from "$lib/carousel_utils.js";

    /**
     * @type {{
     *   filteredClasses: any[],
     *   getStockColor: (stock: number) => string,
     *   onBookClick: (book: any) => void
     * }}
     */
    let { filteredClasses, getStockColor, onBookClick } = $props();

    /**
     * @param {any[]} books
     */
    function sortBooks(books) {
        return [...books].sort(sortBooksBySubjectAndTitle);
    }
</script>

{#each filteredClasses as cls (cls.name)}
    <section class="bg-white rounded-2xl p-6 shadow-sm border border-slate-200 mb-6">
        <h2
            class="text-lg font-bold text-slate-900 mb-4 flex items-center gap-3"
        >
            {cls.name}
            <span
                class="bg-blue-50 border border-blue-200 text-blue-700 text-xs px-2.5 py-1 rounded-full font-bold"
                >{cls.books.length} Bücher</span
            >
        </h2>

        <div class="relative group/carousel carousel-wrapper" data-can-scroll-left="false" data-can-scroll-right="false">
            <!-- Left Navigation Button (FAB) -->
            <button
                class="btn-left absolute left-0 top-1/2 -translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white border border-slate-200 flex items-center justify-center text-slate-400 hover:text-blue-600 hover:border-blue-300 shadow-md transition-all duration-300 z-20 cursor-pointer"
                onclick={(e) => scrollCarousel(e, -1)}
                aria-label="Nach links scrollen"
            >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path></svg>
            </button>

            <!-- Carousel Container -->
            <div use:scrollHandler class="carousel-container flex overflow-x-auto gap-4 pb-4 pt-2 px-2 snap-x snap-mandatory hide-scrollbar scroll-smooth">
                {#each sortBooks(cls.books) as book (book.id)}
                    <KlassenBuchKachelStartseite {book} {getStockColor} onclick={() => onBookClick(book)} />
                {/each}
            </div>

            <!-- Right Navigation Button (FAB) -->
            <button
                class="btn-right absolute right-0 top-1/2 translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white border border-slate-200 flex items-center justify-center text-slate-400 hover:text-blue-600 hover:border-blue-300 shadow-md transition-all duration-300 z-20 cursor-pointer"
                onclick={(e) => scrollCarousel(e, 1)}
                aria-label="Nach rechts scrollen"
            >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
            </button>
        </div>
    </section>
{/each}
