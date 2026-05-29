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
	 *   onEdit: (book: any) => void
	 * }}
	 */
	let { book, onEdit } = $props();

	/**
	 * @param {Event} event
	 */
	function handleEditClick(event) {
		event.stopPropagation();
		onEdit?.(book);
	}

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
			return "bg-linear-to-br from-blue-955 via-indigo-955 to-zinc-955 border-blue-500/20";
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
	class="snap-start shrink-0 w-40 group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-2xl rounded-2xl"
	onclick={handleEditClick}
	role="button"
	tabindex="0"
	onkeydown={(e) => e.key === 'Enter' && handleEditClick(e)}
>
	<div
		class="w-40 h-56 rounded-2xl overflow-hidden shadow-md mb-3 relative border border-zinc-800/50 bg-zinc-955/40 flex items-center justify-center"
	>
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt={book.title}
				loading="lazy"
				class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		{:else}
			<!-- Premium Small Book Cover Mockup -->
			<div
				class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(book.subject)} border border-zinc-850/30 rounded-2xl"
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

		<!-- Hover Overlay -->
		<div class="absolute inset-0 bg-emerald-955/45 opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-20 flex items-center justify-center backdrop-blur-[1px]">
			<div class="bg-emerald-500 text-zinc-950 font-semibold text-sm px-3 py-1.5 rounded-full shadow-lg flex items-center gap-1.5 transform translate-y-4 group-hover:translate-y-0 transition-transform duration-300">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"></path></svg>
				<span>Bearbeiten</span>
			</div>
		</div>

		<div class="absolute top-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<div
					class="bg-cyan-955/90 border border-cyan-900 backdrop-blur-xs px-2 py-1 rounded-md text-[8px] font-bold text-cyan-400 shadow-sm uppercase tracking-wider"
				>
					{book.track}
				</div>
			{/if}
		</div>
	</div>

	<h3
		class="text-xs font-semibold text-zinc-200 line-clamp-2 leading-tight group-hover:text-emerald-400 transition-colors px-1"
	>
		{book.title}
	</h3>
</div>
