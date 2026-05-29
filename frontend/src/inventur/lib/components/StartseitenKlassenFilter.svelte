<script>
    let { classSearchQuery = $bindable(), classTrackFilter = $bindable() } =
        $props();
</script>

<div class="flex flex-col sm:flex-row items-center gap-4 mb-6">
    <div class="relative w-full sm:w-72">
        <svg
            class="w-5 h-5 absolute left-4 top-1/2 -translate-y-1/2 text-slate-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
        </svg>
        <input
            id="class-search"
            type="text"
            placeholder="Klasse suchen..."
            bind:value={classSearchQuery}
            class="w-full bg-white shadow-sm border border-slate-100 text-slate-700 text-sm font-medium rounded-full focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 block pl-12 pr-6 py-3 outline-none transition-all"
        />
    </div>

    <div class="w-full sm:w-48">
        <select
            bind:value={classTrackFilter}
            class="w-full bg-white shadow-sm border border-slate-100 text-slate-700 text-sm font-medium rounded-full focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 block px-6 py-3 outline-none transition-all appearance-none cursor-pointer"
        >
            <option value="Alle Zweige">Alle Zweige</option>
            <option value="Gymnasium">Gymnasium</option>
            <option value="Realschule">Realschule</option>
            <option value="Hauptschule">Hauptschule</option>
            <option value="Förderstufe">Förderstufe</option>
        </select>
    </div>
</div>
