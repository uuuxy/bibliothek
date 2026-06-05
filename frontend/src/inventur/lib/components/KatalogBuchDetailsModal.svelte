<script>
    import { fade, fly } from "svelte/transition";

    let { book, onClose } = $props();

    /** @type {any[]} */
    let exemplare = $state([]);
    let wirdGeladen = $state(false);

    /** @type {string[]} */
    let coverCandidates = $state([]);
    let currentCandidateIndex = $state(0);
    let coverSrc = $derived(coverCandidates[currentCandidateIndex] || "");
    let coverFailed = $state(false);

    $effect(() => {
        const candidates = [];
        if (book?.coverUrl) {
            candidates.push(book.coverUrl);
        }
        if (book?.isbn) {
            const cleanIsbn = book.isbn.replace(/[- ]/g, "");
            candidates.push(`https://books.google.com/books/content?id=&vid=ISBN:${cleanIsbn}&printsec=frontcover&img=1&zoom=1`);
            candidates.push(`https://covers.openlibrary.org/b/isbn/${cleanIsbn}-L.jpg`);
        }
        coverCandidates = candidates;
        currentCandidateIndex = 0;
        coverFailed = candidates.length === 0;
    });

    function onCoverError() {
        if (currentCandidateIndex < coverCandidates.length - 1) {
            currentCandidateIndex++;
        } else {
            coverFailed = true;
        }
    }

    /**
     * @param {Event} event
     */
    function onCoverLoad(event) {
        const image = /** @type {HTMLImageElement} */ (event.currentTarget);
        if (image.naturalWidth < 10 || image.naturalHeight < 10) {
            onCoverError();
        }
    }

    async function ladeExemplare() {
        if (!book || !book.id) return;
        wirdGeladen = true;
        try {
            const res = await fetch(`/api/buecher/titel/${book.id}/exemplare`);
            if (res.ok) {
                exemplare = await res.json();
            } else {
                exemplare = [];
            }
        } catch (e) {
            console.error("Fehler beim Laden der Exemplare:", e);
            exemplare = [];
        } finally {
            wirdGeladen = false;
        }
    }

    $effect(() => {
        ladeExemplare();
    });

    const subjectColors = {
        Mathe: "bg-blue-50 border border-blue-200 text-blue-700",
        Deutsch: "bg-red-50 border border-red-200 text-red-700",
        Englisch: "bg-indigo-50 border border-indigo-200 text-indigo-700",
        Französisch: "bg-indigo-50 border border-indigo-200 text-indigo-700",
        Geographie: "bg-emerald-50 border border-emerald-200 text-emerald-700",
        Geschichte: "bg-amber-50 border border-amber-200 text-amber-700",
        Biologie: "bg-green-50 border border-green-200 text-green-700",
        Chemie: "bg-yellow-50 border border-yellow-200 text-yellow-700",
        Physik: "bg-emerald-50 border border-emerald-200 text-emerald-700",
        Musik: "bg-pink-50 border border-pink-200 text-pink-700",
        Arbeitslehre: "bg-orange-50 border border-orange-200 text-orange-700",
        Politik: "bg-rose-50 border border-rose-200 text-rose-700",
        Informatik: "bg-cyan-50 border border-cyan-200 text-cyan-700",
        Latein: "bg-sky-50 border border-sky-200 text-sky-700",
        Spanisch: "bg-emerald-50 border border-emerald-200 text-emerald-700",
        "kath. Religion": "bg-violet-50 border border-violet-200 text-violet-700",
        "ev. Religion": "bg-violet-50 border border-violet-200 text-violet-700",
        Ethik: "bg-teal-50 border border-teal-200 text-teal-700",
    };

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSubjectColor(subject) {
        if (subject in subjectColors) {
            return subjectColors[/** @type {keyof typeof subjectColors} */ (subject)];
        }
        return "bg-slate-50 border border-slate-200 text-slate-650";
    }

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSubjectGradient(subject) {
        const clean = (subject || "").trim().toLowerCase();
        if (clean.includes("math")) {
            return "bg-linear-to-br from-blue-600 via-indigo-600 to-blue-700 border-blue-500/30";
        }
        if (clean.includes("deu")) {
            return "bg-linear-to-br from-red-600 via-rose-600 to-red-700 border-red-500/30";
        }
        if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) {
            return "bg-linear-to-br from-violet-600 via-purple-600 to-violet-700 border-purple-500/30";
        }
        if (clean.includes("bio") || clean.includes("che") || clean.includes("phy") || clean.includes("nat")) {
            return "bg-linear-to-br from-teal-600 via-emerald-600 to-teal-700 border-teal-500/30";
        }
        if (clean.includes("ges") || clean.includes("pol") || clean.includes("geo") || clean.includes("erd") || clean.includes("soz")) {
            return "bg-linear-to-br from-amber-600 via-orange-600 to-amber-700 border-amber-500/30";
        }
        if (clean.includes("mus") || clean.includes("kun")) {
            return "bg-linear-to-br from-pink-600 via-fuchsia-600 to-pink-700 border-pink-500/30";
        }
        if (clean.includes("inf")) {
            return "bg-linear-to-br from-slate-600 via-slate-700 to-slate-800 border-emerald-500/30";
        }
        return "bg-linear-to-br from-slate-500 via-slate-600 to-slate-700 border-slate-400/30";
    }

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSpineGradient(subject) {
        const clean = (subject || "").trim().toLowerCase();
        if (clean.includes("math")) return "from-blue-300 to-indigo-400";
        if (clean.includes("deu")) return "from-red-300 to-rose-400";
        if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) return "from-violet-300 to-fuchsia-400";
        if (clean.includes("bio") || clean.includes("che") || clean.includes("phy") || clean.includes("nat")) return "from-teal-300 to-emerald-400";
        if (clean.includes("ges") || clean.includes("pol") || clean.includes("geo") || clean.includes("erd") || clean.includes("soz")) return "from-amber-300 to-orange-400";
        if (clean.includes("mus") || clean.includes("kun")) return "from-pink-300 to-fuchsia-400";
        if (clean.includes("inf")) return "from-emerald-300 to-teal-400";
        return "from-slate-400 to-slate-500";
    }
</script>

<!-- Backdrop -->
<button
    class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-45 transition-opacity border-none cursor-default w-full h-full block text-left"
    transition:fade={{ duration: 200 }}
    onclick={onClose}
    aria-label="Schließen"
></button>

<!-- Slide-over panel -->
<div
    class="fixed inset-y-0 right-0 w-full sm:max-w-2xl bg-white shadow-2xl z-50 overflow-y-auto flex flex-col border-l border-slate-100"
    transition:fly={{ x: 400, duration: 300 }}
>
    <!-- Header -->
    <div class="px-6 py-5 border-b border-slate-100 flex items-center justify-between sticky top-0 bg-white z-10">
        <h2 class="text-xl font-extrabold text-slate-800">Buchdetails</h2>
        <button
            onclick={onClose}
            class="p-2 hover:bg-slate-100 rounded-full text-slate-450 transition-colors cursor-pointer"
            aria-label="Schließen"
        >
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
        </button>
    </div>

    <!-- Content -->
    <div class="p-6 space-y-6 flex-1">
        <!-- Buch-Info-Card -->
        <div class="flex flex-col sm:flex-row gap-6 bg-slate-50/50 border border-slate-100 rounded-2xl p-5 shadow-xs">
            <!-- Cover -->
            <div class="w-32 aspect-2/3 shrink-0 rounded-xl overflow-hidden shadow-md bg-white border border-slate-200/55 flex items-center justify-center relative">
                {#if coverSrc && !coverFailed}
                    <img
                        src={coverSrc}
                        alt={`Cover von ${book.title}`}
                        class="w-full h-full object-cover"
                        onerror={onCoverError}
                        onload={onCoverLoad}
                    />
                {:else}
                    <div class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(book.subject)} rounded-xl">
                        <div class="absolute left-0 top-0 bottom-0 w-2.5 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-sm rounded-l-xl"></div>
                        <div class="pl-2 pt-1.5 text-left">
                            <span class="text-[7px] uppercase tracking-widest text-white/80 font-extrabold">{book.subject}</span>
                            <h4 class="text-[10px] font-extrabold text-white leading-snug line-clamp-4 mt-1">{book.title}</h4>
                        </div>
                        <div class="pl-2 pb-1 text-left">
                            <p class="text-[8px] font-semibold text-white/60 truncate">{book.author || "Unbekannter Autor"}</p>
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Metadata -->
            <div class="flex-1 space-y-3">
                <div class="flex flex-wrap gap-1.5">
                    <span class="{getSubjectColor(book.subject)} text-[10px] font-bold px-2 py-0.5 rounded-md">
                        {book.subject}
                    </span>
                    <span class="bg-slate-100 border border-slate-200 text-slate-600 text-[10px] font-bold px-2 py-0.5 rounded-md">
                        Klasse {book.gradeLevel}
                    </span>
                    {#if book.track}
                        <span class="bg-cyan-50 border border-cyan-200 text-cyan-700 text-[10px] font-bold px-2 py-0.5 rounded-md">
                            {book.track}
                        </span>
                    {/if}
                </div>

                <h3 class="text-xl font-extrabold text-slate-900 leading-snug">{book.title}</h3>
                
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-4 text-sm text-slate-600 pt-2">
                    <div>
                        <span class="text-slate-400 font-medium text-xs block uppercase tracking-wider mb-0.5">Autor / Regisseur</span>
                        <span class="font-semibold text-slate-800">{book.author || "Unbekannt"}</span>
                    </div>
                    <div>
                        <span class="text-slate-400 font-medium text-xs block uppercase tracking-wider mb-0.5">ISBN / EAN</span>
                        <span class="font-semibold text-slate-850">{book.isbn || "-"}</span>
                    </div>
                    <div>
                        <span class="text-slate-400 font-medium text-xs block uppercase tracking-wider mb-0.5">Bestand (Verfügbar/Gesamt)</span>
                        <span class="font-semibold text-slate-800">{book.verfuegbar ?? book.stock} / {book.gesamt ?? book.stock}</span>
                    </div>
                    <div>
                        <span class="text-slate-400 font-medium text-xs block uppercase tracking-wider mb-0.5">Standort / Regal</span>
                        {#if book.erweiterteEigenschaften?.standort}
                            <span class="font-bold text-amber-700 bg-amber-50 border border-amber-100 px-2 py-0.5 rounded-md text-xs inline-block">
                                {book.erweiterteEigenschaften.standort}
                            </span>
                        {:else}
                            <span class="text-slate-400 italic text-xs">Kein Standort definiert</span>
                        {/if}
                    </div>
                </div>
            </div>
        </div>

        <!-- Exemplare-Liste -->
        <div class="space-y-4 pt-2">
            <h3 class="text-lg font-bold text-slate-850">Vorhandene Exemplare</h3>

            {#if wirdGeladen}
                <div class="py-12 flex justify-center items-center">
                    <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
                </div>
            {:else if exemplare.length === 0}
                <div class="py-8 flex flex-col items-center justify-center text-slate-400 border border-dashed border-slate-200 rounded-2xl bg-slate-50/50 space-y-2 text-center p-4">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-slate-355" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                    </svg>
                    <span class="text-xs font-semibold">Keine physischen Exemplare mit Barcodes angelegt.</span>
                </div>
            {:else}
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {#each exemplare as exemplar}
                        <div class="p-4 rounded-xl bg-slate-50/50 border border-slate-100 flex flex-col justify-between hover:border-slate-200 transition-colors shadow-xs">
                            <div class="flex items-start justify-between">
                                <div class="space-y-1">
                                    <span class="text-xs font-bold text-blue-700 bg-blue-50 border border-blue-100/50 px-2 py-0.5 rounded">
                                        {exemplar.barcode_id}
                                    </span>
                                    <p class="text-xs text-slate-600 pt-2">
                                        <strong class="text-slate-400 font-medium">Zustand:</strong> {exemplar.zustand_notiz || 'Neuwertig'}
                                    </p>
                                </div>
                                <span class="text-[10px] font-bold px-2 py-0.5 rounded-full {exemplar.ist_ausleihbar ? 'bg-emerald-50 text-emerald-700 border border-emerald-100' : 'bg-rose-50 text-rose-700 border border-rose-100'}">
                                    {exemplar.ist_ausleihbar ? 'Ausleihbar' : 'Gesperrt'}
                                </span>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
</div>
