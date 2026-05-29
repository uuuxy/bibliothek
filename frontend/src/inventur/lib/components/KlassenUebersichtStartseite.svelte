<script>
    import KlassenBuchKachelStartseite from "$lib/components/KlassenBuchKachelStartseite.svelte";
    import { sortBooksBySubjectAndTitle } from "$lib/book_sorting.js";
    import { scrollCarousel, scrollHandler } from "$lib/carousel_utils.js";

    /**
     * @type {{
     *   filteredClasses: any[],
     *   getStockColor: (stock: number) => string
     * }}
     */
    let { filteredClasses, getStockColor } = $props();

    /**
     * @param {any[]} books
     */
    function sortBooks(books) {
        return [...books].sort(sortBooksBySubjectAndTitle);
    }
</script>

{#each filteredClasses as cls (cls.name)}
    <section class="bg-zinc-900/40 rounded-3xl p-6 shadow-2xl border border-zinc-800/50 backdrop-blur-xl mb-8">
        <h2
            class="text-xl font-extrabold text-zinc-100 mb-4 flex items-center gap-3"
        >
            {cls.name}
            <span
                class="bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs px-2.5 py-1 rounded-full font-bold"
                >{cls.books.length} Bücher</span
            >
        </h2>

        <div class="relative group/carousel carousel-wrapper" data-can-scroll-left="false" data-can-scroll-right="false">
            <!-- Left Navigation Button (FAB) -->
            <button
                class="btn-left absolute left-0 top-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 rounded-full bg-zinc-950 border border-zinc-800 flex items-center justify-center text-zinc-400 hover:text-emerald-400 hover:border-emerald-500/30 hover:shadow-emerald-500/10 shadow-2xl transition-all duration-300 z-20 cursor-pointer"
                onclick={(e) => scrollCarousel(e, -1)}
                aria-label="Nach links scrollen"
            >
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path></svg>
            </button>

            <!-- Carousel Container -->
            <div use:scrollHandler class="carousel-container flex overflow-x-auto gap-4 pb-6 pt-2 px-2 snap-x snap-mandatory hide-scrollbar scroll-smooth">
                {#each sortBooks(cls.books) as book (book.id)}
                    <KlassenBuchKachelStartseite {book} {getStockColor} />
                {/each}
            </div>

            <!-- Right Navigation Button (FAB) -->
            <button
                class="btn-right absolute right-0 top-1/2 translate-x-1/2 -translate-y-1/2 w-12 h-12 rounded-full bg-zinc-950 border border-zinc-800 flex items-center justify-center text-zinc-400 hover:text-emerald-400 hover:border-emerald-500/30 hover:shadow-emerald-500/10 shadow-2xl transition-all duration-300 z-20 cursor-pointer"
                onclick={(e) => scrollCarousel(e, 1)}
                aria-label="Nach rechts scrollen"
            >
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
            </button>
        </div>
    </section>
{/each}

<style>
    @import "./KlassenUebersichtStartseite.css";
</style>
