<script>
    let {
        loginError = "",
        loading = false,
        onLogin,
    } = $props();
</script>

<div class="flex h-[80vh] w-full items-center justify-center p-8">
    <div class="w-full h-full bg-white rounded-3xl shadow-2xl border border-slate-100 flex flex-col items-center justify-center space-y-12">
        <div class="text-center">
            <h1 class="text-5xl font-extrabold text-indigo-900 tracking-tight">
                Inventur
            </h1>
            <p class="text-2xl text-slate-500 mt-6">
                Klicken Sie auf Starten, um den Bibliotheksbestand zu scannen und zu erfassen.
            </p>
        </div>

        <button
            type="button"
            onclick={onLogin}
            disabled={loading}
            aria-busy={loading}
            class="px-16 py-8 bg-indigo-600 hover:bg-indigo-700 text-white text-3xl font-bold rounded-2xl transition duration-200 shadow-2xl shadow-indigo-200 disabled:opacity-50 flex items-center justify-center gap-4 cursor-pointer"
        >
            {#if loading}
                <svg class="animate-spin h-8 w-8 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>Wird geladen...</span>
            {:else}
                <span>Inventur starten</span>
            {/if}
        </button>

        {#if loginError}
            <div class="flex items-center justify-center gap-3 text-rose-500 text-lg font-medium bg-rose-50 px-8 py-4 rounded-xl border border-rose-100" role="alert">
                <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{loginError}</span>
            </div>
        {/if}
    </div>
</div>
