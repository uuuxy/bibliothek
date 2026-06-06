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
	 *     verfuegbar: number,
	 *     gesamt: number,
	 *     coverUrl: string
	 *   },
	 *   getStockColor: (stock: number) => string,
	 *   onclick?: () => void
	 * }}
	 */
	let { book, getStockColor, onclick } = $props();

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
			return "bg-linear-to-br from-blue-600 via-indigo-600 to-blue-700 border-blue-400/30";
		}
		if (clean.includes("deu")) {
			return "bg-linear-to-br from-red-600 via-rose-600 to-red-700 border-red-400/30";
		}
		if (clean.includes("eng") || clean.includes("fra") || clean.includes("spa") || clean.includes("lat") || clean.includes("spr")) {
			return "bg-linear-to-br from-violet-600 via-purple-600 to-violet-700 border-purple-400/30";
		}
		if (clean.includes("bio") || clean.includes("che") || clean.includes("phy") || clean.includes("nat")) {
			return "bg-linear-to-br from-teal-600 via-emerald-600 to-teal-700 border-teal-400/30";
		}
		if (clean.includes("ges") || clean.includes("pol") || clean.includes("geo") || clean.includes("erd") || clean.includes("soz")) {
			return "bg-linear-to-br from-amber-600 via-orange-600 to-amber-700 border-amber-400/30";
		}
		if (clean.includes("mus") || clean.includes("kun")) {
			return "bg-linear-to-br from-pink-600 via-fuchsia-600 to-pink-700 border-pink-400/30";
		}
		if (clean.includes("inf")) {
			return "bg-linear-to-br from-slate-600 via-slate-700 to-slate-800 border-emerald-400/30";
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

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="snap-start shrink-0 w-40 group cursor-pointer transition-all duration-300 hover:scale-[1.02] hover:-translate-y-1 bg-white rounded-2xl p-2.5 border border-slate-200 hover:border-blue-300 shadow-sm hover:shadow-md flex flex-col justify-between"
	onclick={onclick}
>
	<div
		class="w-full aspect-2/3 rounded-xl overflow-hidden shadow-sm mb-3 relative bg-slate-50"
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
				class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(book.subject)} border border-slate-200/30 rounded-xl"
			>
				<div class="absolute left-0 top-0 bottom-0 w-2 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-sm rounded-l-xl"></div>
				
				<div class="pl-1.5 pr-0.5 pt-0.5 text-left">
					<span class="text-[7px] uppercase tracking-widest text-white/80 font-extrabold">{book.subject}</span>
					<h4 class="text-[9px] font-extrabold text-white leading-snug line-clamp-4 mt-1">{book.title}</h4>
				</div>
				
				<div class="pl-1.5 pr-0.5 pb-0.5 text-left">
					<p class="text-[7px] font-semibold text-white/60 truncate">{book.author || "Unbekannter Autor"}</p>
				</div>
			</div>
		{/if}

		<div class="absolute bottom-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<span
					class="bg-white/90 border border-slate-200 backdrop-blur-xs px-1.5 py-0.5 rounded text-[8px] font-bold text-slate-700 shadow-sm uppercase tracking-wider"
					>{book.track}</span
				>
			{/if}
		</div>
	</div>

	<div class="px-1.5 pb-1">
		<h3
			class="text-xs font-bold text-slate-800 line-clamp-2 leading-tight group-hover:text-blue-600 transition-colors mb-1"
			title={book.title}
		>
			{book.title}
		</h3>
		<p class="text-[9px] text-slate-400 mb-2 truncate">
			{book.isbn || "-"}
		</p>
		<div class="flex items-center gap-1.5">
			<span class="w-2 h-2 rounded-full {getStockColor(book.verfuegbar)}"
			></span>
			<span class="text-[10px] font-bold text-slate-500"
				>{book.verfuegbar}{#if book.gesamt !== undefined}/{book.gesamt}{/if} Stück</span
			>
		</div>
	</div>
</div>
