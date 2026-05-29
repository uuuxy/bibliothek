<script>
    import { toastState } from "$lib/store.svelte.js";
    import { fly, fade } from "svelte/transition";
</script>

{#if toastState.visible}
    <div
        class="fixed bottom-20 left-1/2 -translate-x-1/2 z-[9999] flex items-center shadow-xl rounded-full px-5 py-3
        {toastState.type === 'error'
            ? 'bg-red-100 text-red-900 border border-red-200'
            : 'bg-green-100 text-green-900 border border-green-200'}
        transition-colors"
        in:fly={{ y: 20, duration: 300, opacity: 0 }}
        out:fade={{ duration: 200 }}
        role="alert"
    >
        {#if toastState.type === "success"}
            <svg
                class="w-5 h-5 mr-3 text-green-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M5 13l4 4L19 7"
                />
            </svg>
        {:else}
            <svg
                class="w-5 h-5 mr-3 text-red-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
            </svg>
        {/if}
        <span class="font-medium">{toastState.message}</span>
    </div>
{/if}
