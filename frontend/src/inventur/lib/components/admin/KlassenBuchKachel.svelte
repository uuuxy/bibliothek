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

<div
	class="snap-start shrink-0 w-40 group cursor-pointer transition-all duration-300 hover:scale-[1.02] hover:-translate-y-1 bg-white rounded-2xl p-2.5 border border-slate-200 hover:border-blue-300 shadow-sm hover:shadow-md flex flex-col justify-between animate-fade-in"
	onclick={handleEditClick}
	role="button"
	tabindex="0"
	onkeydown={(e) => e.key === 'Enter' && handleEditClick(e)}
>
	<div
		class="w-full aspect-2/3 rounded-xl overflow-hidden shadow-sm mb-3 relative bg-slate-50 border border-slate-100 flex items-center justify-center"
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
				class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(book.subject)} border border-slate-200/30 rounded-xl"
			>
				<div class="absolute left-0 top-0 bottom-0 w-2 bg-linear-to-b {getSpineGradient(book.subject)} opacity-90 shadow-sm rounded-l-xl"></div>
				
				<div class="pl-1.5 pr-0.5 pt-0.5 text-left">
					<span class="text-[7px] uppercase tracking-widest text-white/80 font-extrabold font-mono">{book.subject}</span>
					<h4 class="text-[9px] font-extrabold text-white leading-snug line-clamp-4 mt-1">{book.title}</h4>
				</div>
				
				<div class="pl-1.5 pr-0.5 pb-0.5 text-left">
					<p class="text-[7px] font-semibold text-white/60 truncate">{book.author || "Unbekannter Autor"}</p>
				</div>
			</div>
		{/if}

		<!-- Hover Overlay -->
		<div class="absolute inset-0 bg-blue-600/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-20 flex items-center justify-center backdrop-blur-[1px]">
			<div class="bg-blue-600 text-white font-bold text-xs px-3 py-1.5 rounded-full shadow-lg flex items-center gap-1.5 transform translate-y-4 group-hover:translate-y-0 transition-transform duration-300">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"></path></svg>
				<span>Bearbeiten</span>
			</div>
		</div>

		<div class="absolute bottom-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<div
					class="bg-white/90 border border-slate-200 backdrop-blur-xs px-1.5 py-0.5 rounded text-[8px] font-bold text-slate-700 shadow-sm uppercase tracking-wider"
				>
					{book.track}
				</div>
			{/if}
		</div>
	</div>

	<h3
		class="text-xs font-bold text-slate-800 line-clamp-2 leading-tight group-hover:text-blue-600 transition-colors px-1"
	>
		{book.title}
	</h3>
</div>
