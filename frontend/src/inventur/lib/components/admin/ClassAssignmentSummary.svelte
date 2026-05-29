<script>
    let {
        selectedClasses = [],
        selectedBookIds = new Set(),
        selectedBooksList = [],
        isSaving = false,
        isUpdate = false,
        onToggleBook = () => {},
        onsave = () => {},
    } = $props();

    function handleImageError(event) {
        const image = event.currentTarget;
        const fallback = image.dataset.fallback || "";
        const isFallback = image.dataset.isFallback === "1";
        const retryCount = Number(image.dataset.retryCount || "0");

        // Network hiccups are common for external cover hosts. Retry once first.
        if (retryCount < 1) {
            image.dataset.retryCount = String(retryCount + 1);
            const separator = image.src.includes("?") ? "&" : "?";
            image.src = `${image.src}${separator}retry=${Date.now()}`;
            return;
        }

        // Try exactly one deterministic fallback URL before showing placeholder.
        if (!isFallback && fallback && image.src !== fallback) {
            image.dataset.isFallback = "1";
            image.dataset.retryCount = "0";
            image.src = fallback;
            return;
        }

        image.style.display = "none";
        image.nextElementSibling.style.display = "flex";
    }

    function fallbackCover(isbn) {
        if (!isbn) return "";
        const cleaned = String(isbn).replace(/[^0-9Xx]/g, "");
        if (!cleaned) return "";
        return `https://covers.openlibrary.org/b/isbn/${cleaned}-M.jpg`;
    }
</script>

<div
    class="px-4 sm:px-6 py-4 sm:py-6 border-b border-surface-variant/20 flex items-center justify-between"
>
    <h3 class="text-xl font-bold text-gray-900">Auswahl</h3>
    <div
        class="bg-gray-100 px-3 py-1.5 rounded-full text-sm font-bold text-gray-800"
    >
        {selectedBookIds.size}
    </div>
</div>

<div class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-emerald-200 [&::-webkit-scrollbar-thumb]:rounded-full p-4 space-y-2">
    {#if selectedBooksList.length === 0}
        <div
            class="h-full flex flex-col items-center justify-center text-center p-8 opacity-40"
        >
            <svg
                width="48"
                height="48"
                class="text-gray-400 mb-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><rect width="18" height="18" x="3" y="3" rx="2" /><path
                    d="M3 9h18"
                /><path d="M9 21V9" /></svg
            >
            <p class="text-sm font-medium text-gray-500">
                Deine Auswahl ist noch leer
            </p>
        </div>
    {:else}
        {#each selectedBooksList as book (book.id)}
            {@const primaryCoverUrl = book.coverUrl || ""}
            {@const fallbackCoverUrl = fallbackCover(book.isbn)}
            {@const coverUrl = primaryCoverUrl || fallbackCoverUrl}
            <div
				class="flex items-center gap-3.5 hover:bg-emerald-50 p-2 rounded-xl transition-colors group"
            >
                <div
                    class="w-10 h-14 rounded overflow-hidden flex-shrink-0 bg-surface-container shadow-sm bg-white"
                >
                    {#if coverUrl}
                        <img
                            src={coverUrl}
                            data-fallback={fallbackCoverUrl}
                            data-is-fallback={primaryCoverUrl ? "0" : "1"}
                            data-retry-count="0"
                            alt=""
                            loading="eager"
                            decoding="async"
                            class="w-full h-full object-cover"
                            onerror={handleImageError}
                        />
                        <div
                            class="w-full h-full hidden items-center justify-center bg-gray-100 text-gray-300"
                        >
                            <svg
                                width="24"
                                height="24"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"
                                /></svg
                            >
                        </div>
                    {:else}
                        <div
                            class="w-full h-full flex items-center justify-center bg-gray-100 text-gray-300"
                        >
                            <svg
                                width="24"
                                height="24"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"
                                /></svg
                            >
                        </div>
                    {/if}
                </div>
                <p
                    class="font-medium text-gray-800 flex-grow truncate leading-tight"
                >
                    {book.title}
                </p>
                <button
                    onclick={() => onToggleBook(book.id)}
                    class="text-gray-400 hover:text-red-500 p-1 rounded-full transition-colors"
                    title="Buch entfernen"
                >
                    <svg
                        width="20"
                        height="20"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><line x1="18" y1="6" x2="6" y2="18"></line><line
                            x1="6"
                            y1="6"
                            x2="18"
                            y2="18"
                        ></line></svg
                    >
                </button>
            </div>
        {/each}
    {/if}
</div>

<footer
    class="p-4 sm:p-6 bg-white border-t border-surface-variant/20 flex flex-col gap-4"
>
    <button
        disabled={selectedClasses.length === 0 ||
            (!isUpdate && selectedBookIds.size === 0) ||
            isSaving}
        onclick={onsave}
        class="flex items-center justify-center w-full gap-2 p-5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-300 disabled:text-gray-500 text-white rounded-full font-bold text-base shadow-lg hover:shadow-emerald-200 transition-all tracking-wide"
    >
        <svg
            fill="none"
            height="24"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            viewBox="0 0 24 24"
            width="24"
            xmlns="http://www.w3.org/2000/svg"
            ><path
                d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"
            ></path><polyline points="17 21 17 13 7 13 7 21"
            ></polyline><polyline points="7 3 7 8 15 8"></polyline></svg
        >
        {isSaving ? "SPEICHERT..." : "AUSWAHL SPEICHERN"}
    </button>
</footer>
