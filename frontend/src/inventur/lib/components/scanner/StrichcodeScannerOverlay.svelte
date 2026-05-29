<script>
    import { fade } from "svelte/transition";
    import KameraScanner from "$lib/components/scanner/KameraScanner.svelte";

    let { isScanning = $bindable(), onScan } = $props();
    let scanStatus = $state("");

    /**
     * @param {string} code
     */
    function handleScan(code) {
        onScan(code);
        isScanning = false;
    }
</script>

{#if isScanning}
    <div
        class="fixed inset-0 z-60 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4"
        transition:fade
    >
        <div
            class="bg-white p-6 rounded-2xl shadow-xl w-full max-w-md relative"
        >
            <button
                onclick={() => (isScanning = false)}
                class="absolute top-4 right-4 text-gray-500 hover:text-gray-800"
                aria-label="Scanner schließen"
            >
                <svg
                    class="w-6 h-6"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M6 18L18 6M6 6l12 12"
                    />
                </svg>
            </button>
            <h3 class="text-lg font-bold mb-4 text-center">ISBN scannen</h3>
            <KameraScanner
                onDecode={handleScan}
                onStatusChange={(/** @type {string} */ s) => (scanStatus = s)}
            />
            <p class="text-center text-sm text-gray-600 mt-2">{scanStatus}</p>
        </div>
    </div>
{/if}
