<script>
    let { viewMode = $bindable() } = $props();
</script>

<header class="flex flex-col items-center mb-8">
    <div
        class="bg-white p-1.5 rounded-full shadow-sm border border-slate-100 inline-flex relative"
        role="tablist"
        aria-label="Ansichtsmodus"
    >
        <div
            class="absolute top-1.5 bottom-1.5 w-1/3 bg-indigo-100 rounded-full transition-transform duration-300 ease-out"
            style="transform: translateX({viewMode === 'books'
                ? '0%'
                : viewMode === 'classes' ? '100%' : '200%'})"
        ></div>

        <button
            role="tab"
            aria-selected={viewMode === "books"}
            class="relative z-10 px-6 py-2.5 rounded-full text-sm font-bold transition-colors duration-300 {viewMode ===
            'books'
                ? 'text-indigo-700'
                : 'text-slate-500 hover:text-slate-700'}"
            onclick={() => (viewMode = "books")}
        >
            Buch-Ansicht
        </button>
        <button
            role="tab"
            aria-selected={viewMode === "classes"}
            class="relative z-10 px-6 py-2.5 rounded-full text-sm font-bold transition-colors duration-300 {viewMode ===
            'classes'
                ? 'text-indigo-700'
                : 'text-slate-500 hover:text-slate-700'}"
            onclick={() => (viewMode = "classes")}
        >
            Klassen-Ansicht
        </button>
        <button
            role="tab"
            aria-selected={viewMode === "real-classes"}
            class="relative z-10 px-6 py-2.5 rounded-full text-sm font-bold transition-colors duration-300 {viewMode ===
            'real-classes'
                ? 'text-indigo-700'
                : 'text-slate-500 hover:text-slate-700'}"
            onclick={() => (viewMode = "real-classes")}
        >
            Echte Klassen
        </button>
    </div>
</header>
