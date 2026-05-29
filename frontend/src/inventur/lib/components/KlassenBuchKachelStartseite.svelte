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
	 *     coverUrl: string
	 *   },
	 *   getStockColor: (stock: number) => string
	 * }}
	 */
	let { book, getStockColor } = $props();

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

	/**
	 * @param {string} subject
	 * @returns {string}
	 */
	function getSubjectGradient(subject) {
		const clean = (subject || "").trim().toLowerCase();
		if (clean.includes("math")) {
			return "bg-linear-to-br from-blue-950 via-indigo-955 to-zinc-955 border-blue-500/20";
		}
		if (clean.includes("deu")) {
			return "bg-linear-to-br from-red-955 via-rose-955 to-zinc-955 border-red-500/20";
		}
		if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) {
			return "bg-linear-to-br from-violet-955 via-purple-955 to-zinc-955 border-purple-500/20";
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
</script>

<div
	class="snap-start shrink-0 w-40 group cursor-pointer transition-all duration-300 hover:scale-[1.02] hover:-translate-y-1 bg-zinc-900/40 rounded-3xl p-2.5 border border-zinc-800/50 hover:border-emerald-500/20 shadow-lg flex flex-col justify-between"
>
	<div
		class="w-full aspect-2/3 rounded-2xl overflow-hidden shadow-md mb-3 relative bg-zinc-950/40"
	>
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt={`Cover von ${book.title}`}
				loading="lazy"
				class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		{:else}
			<!-- Premium Small Book Cover Mockup -->
			<div
				class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(book.subject)} border border-zinc-800/30 rounded-2xl"
			>
				<div class="absolute left-0 top-0 bottom-0 w-2 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-sm rounded-l-2xl"></div>
				
				<div class="pl-1.5 pr-0.5 pt-0.5 text-left">
					<span class="text-[7px] uppercase tracking-widest text-emerald-400 font-extrabold font-mono">{book.subject}</span>
					<h4 class="text-[9px] font-extrabold text-zinc-100 leading-snug line-clamp-4 mt-1">{book.title}</h4>
				</div>
				
				<div class="pl-1.5 pr-0.5 pb-0.5 text-left">
					<p class="text-[7px] font-semibold text-zinc-450 truncate">{book.author || "Unbekannter Autor"}</p>
				</div>
			</div>
		{/if}

		<div class="absolute bottom-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<span
					class="bg-cyan-950/90 border border-cyan-900 backdrop-blur-xs px-1.5 py-0.5 rounded text-[8px] font-extrabold text-cyan-400 shadow-sm uppercase tracking-wider"
					>{book.track}</span
				>
			{/if}
		</div>
	</div>

	<div class="px-1.5 pb-1">
		<h3
			class="text-xs font-bold text-zinc-200 line-clamp-2 leading-tight group-hover:text-emerald-400 transition-colors mb-1"
			title={book.title}
		>
			{book.title}
		</h3>
		<p class="text-[9px] text-zinc-500 font-mono mb-2 truncate">
			{book.isbn || "-"}
		</p>
		<div class="flex items-center gap-1.5">
			<span class="w-2 h-2 rounded-full {getStockColor(book.stock)}"
			></span>
			<span class="text-[10px] font-bold text-zinc-400"
				>{book.stock} Stück</span
			>
		</div>
	</div>
</div>
