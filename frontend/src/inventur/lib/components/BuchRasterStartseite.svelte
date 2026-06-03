<script>
    import BuchKarte from "./BuchKarte.svelte";

    /**
     * @type {{
     *   filteredBooks: any[],
     *   onBookClick: (book: any) => void
     * }}
     */
    let { filteredBooks, onBookClick } = $props();
</script>

<div
    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
>
    {#each filteredBooks as book (book.id)}
        <BuchKarte {book} onclick={() => onBookClick(book)} />
    {/each}
</div>

