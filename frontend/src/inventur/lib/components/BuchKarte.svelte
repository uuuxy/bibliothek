<script>
    /**
     * @type {{
     *   book: {
     *     id: string,
     *     isbn: string,
     *     title: string,
     *     author: string,
     *     subject: string,
     *     gradeLevel: number,
     *     track: string,
     *     stock: number,
     *     coverUrl: string,
     *     lastCounted: string
     *   }
     * }}
     */
    let { book } = $props();

    /** @type {string[]} */
    let coverCandidates = $state([]);
    let currentCandidateIndex = $state(0);
    let coverSrc = $derived(coverCandidates[currentCandidateIndex] || "");
    let coverFailed = $state(false);
    let copied = $state(false);

    /**
     * @param {string} isbn
     */
    function copyIsbn(isbn) {
        if (!isbn) return;
        navigator.clipboard.writeText(isbn);
        copied = true;
        setTimeout(() => (copied = false), 2000);
    }

    const subjectColors = {
        Mathe: "bg-blue-500/10 border border-blue-500/20 text-blue-400",
        Deutsch: "bg-red-500/10 border border-red-500/20 text-red-400",
        Englisch: "bg-indigo-500/10 border border-indigo-500/20 text-indigo-400",
        Französisch: "bg-indigo-500/10 border border-indigo-500/20 text-indigo-400",
        Geographie: "bg-emerald-500/10 border border-emerald-500/20 text-emerald-400",
        Geschichte: "bg-amber-500/10 border border-amber-500/20 text-amber-400",
        Biologie: "bg-green-500/10 border border-green-500/20 text-green-400",
        Chemie: "bg-yellow-500/10 border border-yellow-500/20 text-yellow-400",
        Physik: "bg-emerald-500/10 border border-emerald-500/20 text-emerald-400",
        Musik: "bg-pink-500/10 border border-pink-500/20 text-pink-400",
        Arbeitslehre: "bg-orange-500/10 border border-orange-500/20 text-orange-400",
        Politik: "bg-rose-500/10 border border-rose-500/20 text-rose-400",
        Informatik: "bg-cyan-500/10 border border-cyan-500/20 text-cyan-400",
        Latein: "bg-sky-500/10 border border-sky-500/20 text-sky-400",
        Spanisch: "bg-emerald-500/10 border border-emerald-500/20 text-emerald-400",
        "kath. Religion": "bg-violet-500/10 border border-violet-500/20 text-violet-400",
        "ev. Religion": "bg-violet-500/10 border border-violet-500/20 text-violet-400",
        Ethik: "bg-teal-500/10 border border-teal-500/20 text-teal-400",
    };

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSubjectColor(subject) {
        if (subject in subjectColors) {
            return subjectColors[/** @type {keyof typeof subjectColors} */ (subject)];
        }
        return "bg-zinc-800/40 border border-zinc-700/50 text-zinc-300";
    }

    /**
     * @param {number} stock
     * @returns {string}
     */
    function getStockDotColor(stock) {
        if (stock === 0)
            return "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]";
        if (stock < 5)
            return "bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.6)]";
        return "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]";
    }

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSubjectGradient(subject) {
        const clean = (subject || "").trim().toLowerCase();
        if (clean.includes("math")) {
            return "bg-linear-to-br from-blue-950 via-indigo-950 to-zinc-950 border-blue-500/20";
        }
        if (clean.includes("deu")) {
            return "bg-linear-to-br from-red-950 via-rose-950 to-zinc-950 border-red-500/20";
        }
        if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) {
            return "bg-linear-to-br from-violet-950 via-purple-955 to-zinc-955 border-purple-500/20";
        }
        if (clean.includes("bio") || clean.includes("che") || clean.includes("phy") || clean.includes("nat")) {
            return "bg-linear-to-br from-teal-955 via-emerald-955 to-zinc-955 border-teal-500/20";
        }
        if (clean.includes("ges") || clean.includes("pol") || clean.includes("geo") || clean.includes("erd") || clean.includes("soz")) {
            return "bg-linear-to-br from-amber-955 via-orange-955 to-zinc-955 border-amber-500/20";
        }
        if (clean.includes("mus") || clean.includes("kun")) {
            return "bg-linear-to-br from-pink-955 via-fuchsia-955 to-zinc-955 border-pink-500/20";
        }
        if (clean.includes("inf")) {
            return "bg-linear-to-br from-slate-900 via-zinc-950 to-black border-emerald-500/20";
        }
        return "bg-linear-to-br from-zinc-800 via-zinc-900 to-zinc-950 border-zinc-700/20";
    }

    /**
     * @param {string} subject
     * @returns {string}
     */
    function getSpineGradient(subject) {
        const clean = (subject || "").trim().toLowerCase();
        if (clean.includes("math")) return "from-blue-400 to-indigo-500";
        if (clean.includes("deu")) return "from-red-400 to-rose-500";
        if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) return "from-violet-400 to-fuchsia-500";
        if (clean.includes("bio") || clean.includes("che") || clean.includes("phy") || clean.includes("nat")) return "from-teal-400 to-emerald-500";
        if (clean.includes("ges") || clean.includes("pol") || clean.includes("geo") || clean.includes("erd") || clean.includes("soz")) return "from-amber-400 to-orange-500";
        if (clean.includes("mus") || clean.includes("kun")) return "from-pink-400 to-fuchsia-500";
        if (clean.includes("inf")) return "from-emerald-400 to-teal-500";
        return "from-zinc-500 to-zinc-650";
    }

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

    /**
     * @param {string} dateString
     * @returns {string|null}
     */
    function formatDate(dateString) {
        if (!dateString) return null;
        try {
            const date = new Date(dateString);
            if (isNaN(date.getTime())) return null;
            return new Intl.DateTimeFormat("de-DE", {
                day: "2-digit",
                month: "2-digit",
                year: "numeric",
            }).format(date);
        } catch {
            return null;
        }
    }
</script>

<article
    class="bg-zinc-900/40 rounded-3xl border border-zinc-800/50 backdrop-blur-xl flex flex-col h-full group overflow-hidden hover:border-emerald-500/20 transition-all duration-300 shadow-2xl"
>
    {#if coverSrc && !coverFailed}
        <div
            class="w-full h-56 rounded-t-2xl overflow-hidden bg-zinc-950/20 flex items-center justify-center border-b border-zinc-800/40 relative"
        >
            <img
                src={coverSrc}
                alt={`Cover von ${book.title}`}
                loading="lazy"
                class="object-contain h-full w-full p-3 transition-all duration-500 group-hover:scale-105"
                onerror={onCoverError}
                onload={onCoverLoad}
            />
        </div>
    {:else}
        <!-- CSS Styled Book Cover Mockup -->
        <div
            class="w-full h-56 rounded-t-2xl overflow-hidden {getSubjectGradient(book.subject)} flex flex-col justify-between p-5 relative border-b border-zinc-800/50 shadow-inner"
        >
            <!-- Styled Book Spine / Accent Bar -->
            <div class="absolute left-0 top-0 bottom-0 w-3 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-md"></div>
            
            <div class="pl-4 pr-1 pt-1 text-left">
                <span class="text-[9px] uppercase tracking-widest text-emerald-400 font-extrabold font-mono">{book.subject}</span>
                <h4 class="text-sm font-extrabold text-zinc-100 leading-snug line-clamp-3 mt-1.5">{book.title}</h4>
            </div>
            
            <div class="pl-4 pr-1 pb-1 text-left">
                <p class="text-[10px] font-semibold text-zinc-400 truncate">{book.author || "Unbekannter Autor"}</p>
                <p class="text-[8px] font-mono text-zinc-650 mt-0.5">ISBN: {book.isbn || "-"}</p>
            </div>
        </div>
    {/if}

    <div class="grow p-5 pt-4 flex flex-col justify-between">
        <div>
            <h2
                class="text-base font-bold text-zinc-100 leading-snug mb-1 line-clamp-2"
                title={book.title}
            >
                {book.title}
            </h2>
            <button
                class="text-[11px] text-zinc-400 mb-4 font-mono tracking-wide group/isbn flex items-center gap-2 text-left transition-colors hover:text-emerald-400 cursor-pointer"
                onclick={() => copyIsbn(book.isbn)}
                title="ISBN kopieren"
                aria-label="ISBN kopieren"
            >
                <span>ISBN: {book.isbn || "-"}</span>
                {#if book.isbn}
                    {#if copied}
                        <span class="text-emerald-400 text-[10px] font-sans font-bold"
                            >Kopiert!</span
                        >
                    {:else}
                        <svg
                            class="w-3.5 h-3.5 text-zinc-600 opacity-0 group-hover/isbn:opacity-100 transition-opacity"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                            ><path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                            ></path></svg
                        >
                    {/if}
                {/if}
            </button>

            <div class="flex flex-wrap gap-1.5 mb-4">
                <span
                    class="{getSubjectColor(
                        book.subject,
                    )} text-[10px] font-bold px-2 py-0.5 rounded-md"
                >
                    {book.subject}
                </span>
                <span
                    class="bg-zinc-850/50 border border-zinc-800 text-zinc-300 text-[10px] font-bold px-2 py-0.5 rounded-md"
                >
                    Klasse {book.gradeLevel}
                </span>
                {#if book.track}
                    <span
                        class="bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 text-[10px] font-bold px-2 py-0.5 rounded-md"
                    >
                        {book.track}
                    </span>
                {/if}
            </div>
        </div>

        <div class="space-y-4">
            <div
                class="inline-flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-xl bg-zinc-950/20 border border-zinc-850/40 text-[10px] text-zinc-400 font-medium"
            >
                <svg
                    class="w-3.5 h-3.5 text-zinc-500"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                    ></path>
                </svg>
                <span>
                    Zuletzt geprüft: {formatDate(book.lastCounted) || "Unbekannt"}
                </span>
            </div>

            <div
                class="pt-3 border-t border-zinc-800/40 flex justify-between items-center"
            >
                <span class="text-xs font-semibold text-zinc-400">Bestand</span>
                <div class="flex items-center gap-2">
                    <span
                        class="w-2 h-2 rounded-full {getStockDotColor(book.stock)}"
                    ></span>
                    <span class="text-lg font-extrabold text-zinc-100 font-mono">{book.stock}</span>
                </div>
            </div>
        </div>
    </div>
</article>
