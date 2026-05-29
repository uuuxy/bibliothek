<script>
    import KlassenBuchKachel from "$lib/components/admin/KlassenBuchKachel.svelte";
    import { sortBooksBySubjectAndTitle } from "$lib/book_sorting.js";
    import { scrollCarousel, scrollHandler } from "$lib/carousel_utils.js";

    /**
     * @type {{
     *   group: {
     *     className: string,
     *     books: any[]
     *   },
     *   onEdit: () => void,
     *   onDelete: () => void
     * }}
     */
    let { group, onEdit, onDelete } = $props();

    let sortedBooks = $derived([...group.books].sort(sortBooksBySubjectAndTitle));
</script>

<div class="class-group">
    <div class="flex justify-between items-center mb-4 px-2">
        <h2 class="text-2xl font-bold text-zinc-100 flex items-center gap-2 font-sans">
            <span
                class="bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 px-3 py-1 rounded-lg text-sm font-semibold"
                >{group.books.length} Bücher</span
            >
            {group.className}
        </h2>
        <div class="flex gap-2">
            <button
                onclick={onEdit}
                class="bg-emerald-500/10 border border-emerald-500/20 hover:bg-emerald-500/20 text-emerald-400 font-bold px-4 py-2 rounded-lg transition-colors shadow-sm flex items-center gap-2 cursor-pointer"
                title="Klasse bearbeiten"
                aria-label="Klasse bearbeiten"
            >
                <svg
                    class="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    ><path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                    /></svg>
                Bücher verwalten
            </button>
            <button
                onclick={onDelete}
                class="text-red-400 hover:text-red-300 hover:bg-red-500/10 p-2 rounded-lg transition-colors cursor-pointer"
                title="Klasse löschen"
                aria-label="Klasse löschen"
            >
                <svg
                    class="w-5 h-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    ><path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    /></svg
                >
            </button>
        </div>
    </div>

    <!-- Horizontal Scroll Container (Netflix Style) -->
    <div class="relative group/admin-carousel carousel-wrapper" data-can-scroll-left="false" data-can-scroll-right="false">
        
        <!-- Left Gradient -->
        <div class="absolute left-0 top-0 bottom-6 w-24 bg-linear-to-r from-zinc-950 to-transparent pointer-events-none z-30 opacity-0 group-data-[can-scroll-left=true]/admin-carousel:opacity-100 transition-opacity duration-300 rounded-l-2xl"></div>

        <button
            class="btn-left absolute left-0 top-1/2 -translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-zinc-950 border border-zinc-800 flex items-center justify-center text-zinc-400 hover:text-emerald-400 hover:border-emerald-500/30 hover:shadow-emerald-500/10 shadow-2xl transition-all duration-300 z-40 cursor-pointer"
            onclick={(e) => scrollCarousel(e, -1)}
            aria-label="Nach links scrollen"
        >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path></svg>
        </button>

        <div use:scrollHandler class="carousel-container flex overflow-x-auto gap-5 pb-6 px-2 snap-x hide-scrollbar scroll-smooth">
            {#each sortedBooks as book (book.id)}
                <KlassenBuchKachel {book} onEdit={onEdit} />
            {/each}
        </div>

        <!-- Right Gradient -->
        <div class="absolute right-0 top-0 bottom-6 w-32 bg-linear-to-l from-zinc-950 to-transparent pointer-events-none z-30 opacity-0 group-data-[can-scroll-right=true]/admin-carousel:opacity-100 transition-opacity duration-300 rounded-r-2xl"></div>

        <button
            class="btn-right absolute right-0 top-1/2 translate-x-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-zinc-950 border border-zinc-800 flex items-center justify-center text-zinc-400 hover:text-emerald-400 hover:border-emerald-500/30 hover:shadow-emerald-500/10 shadow-2xl transition-all duration-300 z-40 cursor-pointer"
            onclick={(e) => scrollCarousel(e, 1)}
            aria-label="Nach rechts scrollen"
        >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
        </button>
    </div>
</div>


