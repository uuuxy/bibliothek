<script>
    let { formular = $bindable(), onCoverUpload } = $props();
    /** @type {HTMLInputElement|null} */
    let fileInput = $state(null);

    /** @param {Event} e */
    function handleFileChange(e) {
        onCoverUpload(e);
    }
</script>

<div class="flex flex-col items-center">
    <div
        class="w-32 h-44 bg-gray-100 rounded-lg shadow-md overflow-hidden mb-4 relative group"
    >
        {#if formular.coverUrl}
            <img
                src={formular.coverUrl}
                alt="Cover"
                class="w-full h-full object-cover"
                onerror={(/** @type {Event} */ e) => {
                    const target = /** @type {HTMLImageElement} */ (e.target);
                    if (target) {
                        target.onerror = null;
                        target.src = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 24 24" fill="none" stroke="%239ca3af" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>';
                    }
                }}
            />
        {:else}
            <div
                class="w-full h-full flex items-center justify-center text-gray-400"
            >
                Kein Bild
            </div>
        {/if}

        <!-- Overlay Upload Button -->
        {#if formular.id}
            <button
                class="absolute inset-0 bg-black/40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer border-none w-full h-full"
                onclick={() => fileInput?.click()}
                aria-label="Cover hochladen"
            >
                <svg
                    class="w-8 h-8 text-white"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
                    />
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                </svg>
            </button>
        {:else}
            <div
                class="absolute inset-0 flex items-center justify-center text-center p-2 text-xs text-gray-500"
            >
                Erst speichern, dann Bild hochladen
            </div>
        {/if}
    </div>
    {#if formular.id}
        <input
            id="cover-upload-drawer"
            type="file"
            hidden
            bind:this={fileInput}
            onchange={handleFileChange}
            accept="image/*"
        />
        <button
            class="text-sm text-emerald-600 font-medium hover:text-emerald-700"
            onclick={() => fileInput?.click()}
        >
            Cover ändern
        </button>
    {/if}
</div>

