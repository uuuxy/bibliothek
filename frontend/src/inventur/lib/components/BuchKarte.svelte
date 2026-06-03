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
     *     lastCounted: string,
     *     medientyp?: string
     *   },
     *   onclick?: () => void
     * }}
     */
    let { book, onclick } = $props();

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
        return "bg-slate-50 border border-slate-200 text-slate-600";
    }

    /**
     * @param {number} stock
     * @returns {string}
     */
    function getStockDotColor(stock) {
        if (stock === 0)
            return "bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.4)]";
        if (stock < 5)
            return "bg-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.4)]";
        return "bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.4)]";
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

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<article
    class="bg-white rounded-2xl border border-slate-200 flex flex-col h-full group overflow-hidden hover:border-blue-300 hover:shadow-md transition-all duration-300 shadow-sm cursor-pointer"
    onclick={onclick}
>
    {#if coverSrc && !coverFailed}
        <div
            class="w-full h-56 rounded-t-2xl overflow-hidden bg-slate-50 flex items-center justify-center border-b border-slate-100 relative"
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
            class="w-full h-56 rounded-t-2xl overflow-hidden {getSubjectGradient(book.subject)} flex flex-col justify-between p-5 relative border-b border-slate-100 shadow-inner"
        >
            <!-- Styled Book Spine / Accent Bar -->
            <div class="absolute left-0 top-0 bottom-0 w-3 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-md"></div>
            
            <div class="pl-4 pr-1 pt-1 text-left">
                <span class="text-[9px] uppercase tracking-widest text-white/80 font-extrabold">{book.subject}</span>
                <h4 class="text-sm font-extrabold text-white leading-snug line-clamp-3 mt-1.5">{book.title}</h4>
            </div>
            
            <div class="pl-4 pr-1 pt-1 text-left">
                <p class="text-[10px] font-semibold text-white/70 truncate">
                    {book.medientyp === 'DVD' ? (book.author ? 'Regisseur: ' + book.author : 'Unbekannter Regisseur') : (book.author || 'Unbekannter Autor')}
                </p>
                <p class="text-[8px] text-white/50 mt-0.5">{book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {book.isbn || "-"}</p>
            </div>
        </div>
    {/if}

    <div class="grow p-5 pt-4 flex flex-col justify-between">
        <div>
            <h2
                class="text-base font-bold text-slate-900 leading-snug mb-1 line-clamp-2"
                title={book.title}
            >
                {book.title}
            </h2>
            <button
                class="text-[11px] text-slate-400 mb-4 tracking-wide group/isbn flex items-center gap-2 text-left transition-colors hover:text-blue-600 cursor-pointer"
                onclick={(e) => { e.stopPropagation(); copyIsbn(book.isbn); }}
                title={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') + ' kopieren'}
                aria-label={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') + ' kopieren'}
            >
                <span>{book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {book.isbn || "-"}</span>
                {#if book.isbn}
                    {#if copied}
                        <span class="text-blue-600 text-[10px] font-sans font-bold"
                            >Kopiert!</span
                        >
                    {:else}
                        <svg
                            class="w-3.5 h-3.5 text-slate-300 opacity-0 group-hover/isbn:opacity-100 transition-opacity"
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
                    class="bg-slate-50 border border-slate-200 text-slate-600 text-[10px] font-bold px-2 py-0.5 rounded-md"
                >
                    Klasse {book.gradeLevel}
                </span>
                {#if book.track}
                    <span
                        class="bg-cyan-50 border border-cyan-200 text-cyan-700 text-[10px] font-bold px-2 py-0.5 rounded-md"
                    >
                        {book.track}
                    </span>
                {/if}
            </div>
        </div>

        <div class="space-y-4">
            <div
                class="inline-flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-lg bg-slate-50 border border-slate-100 text-[10px] text-slate-500 font-medium"
            >
                <svg
                    class="w-3.5 h-3.5 text-slate-400"
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
                class="pt-3 border-t border-slate-100 flex justify-between items-center"
            >
                <span class="text-xs font-semibold text-slate-400">Bestand</span>
                <div class="flex items-center gap-2">
                    <span
                        class="w-2 h-2 rounded-full {getStockDotColor(book.stock)}"
                    ></span>
                    <span class="text-lg font-extrabold text-slate-800">{book.stock}</span>
                </div>
            </div>
        </div>
    </div>
</article>
