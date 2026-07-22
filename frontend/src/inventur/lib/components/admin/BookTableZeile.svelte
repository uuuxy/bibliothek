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
	 *     coverUrl: string,
	 *     lastCounted: string,
	 *     erweiterteEigenschaften?: { standort?: string }
	 *   },
	 *   index: number,
	 *   dragOverIndex: number|null,
	 *   isSelected: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onToggleSelect: (id: string) => void,
	 *   onDragStart: (event: any, index: number) => void,
	 *   onDragOver: (event: any, index: number) => void,
	 *   onDragLeave: (event: any, index: number) => void,
	 *   onDrop: (event: any, index: number) => void,
	 *   onDragEnd: (event: any) => void
	 * }}
	 */
	let {
		book,
		index,
		dragOverIndex,
		isSelected,
		onOpenDetail,
		onToggleSelect,
		onDragStart,
		onDragOver,
		onDragLeave,
		onDrop,
		onDragEnd
	} = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
	let coverFailed = $state(false);

	$effect(() => {
		const candidates = [];
		if (book?.coverUrl) {
			candidates.push(book.coverUrl);
		}
		if (book?.isbn) {
			const cleanIsbn = book.isbn.replace(/[- ]/g, '');
			candidates.push(
				`https://books.google.com/books/content?id=&vid=ISBN:${cleanIsbn}&printsec=frontcover&img=1&zoom=1`
			);
			candidates.push(`https://covers.openlibrary.org/b/isbn/${cleanIsbn}-S.jpg`);
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
</script>

<tr
	class="group transition-colors cursor-pointer border-b border-slate-100 text-slate-700 hover:bg-slate-50/50 {dragOverIndex ===
	index
		? 'border-t-2 border-blue-500 bg-blue-50/30'
		: ''}"
	draggable="true"
	ondragstart={(event) => onDragStart(event, index)}
	ondragover={(event) => onDragOver(event, index)}
	ondragleave={(event) => onDragLeave(event, index)}
	ondrop={(event) => onDrop(event, index)}
	ondragend={onDragEnd}
	onclick={() => onOpenDetail(book)}
>
	<td class="px-6 py-3" onclick={(event) => event.stopPropagation()}>
		<div class="flex items-center gap-2">
			<svg
				class="w-4 h-4 text-slate-350 cursor-grab active:cursor-grabbing hover:text-slate-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8h16M4 16h16" />
			</svg>
			<input
				type="checkbox"
				aria-label="Buch auswählen"
				class="rounded border-slate-200 bg-white text-blue-650 focus:ring-blue-500/20 cursor-pointer"
				checked={isSelected}
				onchange={() => onToggleSelect(book.id)}
			/>
		</div>
	</td>

	<td class="px-6 py-3">
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt="Cover"
				loading="lazy"
				class="w-12 aspect-3/4 object-cover rounded-md shadow-xs border border-slate-100"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		{:else}
			<div
				class="w-12 aspect-3/4 rounded-md shadow-xs flex items-center justify-center font-bold text-white bg-linear-to-br from-blue-500 to-indigo-650 text-sm border border-indigo-600/10"
			>
				{book.title ? book.title.charAt(0).toUpperCase() : '?'}
			</div>
		{/if}
	</td>

	<td class="px-6 py-3 font-semibold text-slate-900">
		{book.title}
		<div class="text-xs text-slate-450 font-normal">{book.author}</div>
	</td>

	<td class="px-6 py-3">
		{#if book.subject}
			<span
				class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-slate-100 text-slate-600"
			>
				{book.subject}
			</span>
		{:else}
			<span class="text-slate-350 text-xs">–</span>
		{/if}
	</td>
	<!-- Klasse 0 = nicht zugeordnet: „–" statt einer sinnlosen „Kl. 0". -->
	<td class="px-6 py-3 text-slate-600 text-sm">
		{#if book.gradeLevel}Kl. {book.gradeLevel}{:else}<span class="text-slate-350 text-xs">–</span>{/if}
	</td>

	<td class="px-6 py-3">
		{#if book.track}
			<span
				class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-cyan-50 text-cyan-700 border border-cyan-100"
			>
				{book.track}
			</span>
		{:else}
			<span class="text-slate-350 text-xs">-</span>
		{/if}
	</td>

	<td class="px-6 py-3">
		{#if book.erweiterteEigenschaften?.standort}
			<span
				class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-amber-50 text-amber-700 border border-amber-100"
			>
				{book.erweiterteEigenschaften.standort}
			</span>
		{:else}
			<span class="text-slate-350 text-xs">-</span>
		{/if}
	</td>

	<td class="px-6 py-3 text-right">
		{#if book.lastCounted}
			<span
				class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-50 border border-slate-100 text-slate-500 text-xs font-medium"
			>
				{new Date(book.lastCounted).toLocaleDateString('de-DE')}
			</span>
		{:else}
			<span class="text-slate-350 text-xs">-</span>
		{/if}
	</td>

	<td class="px-6 py-3 text-right">
		<span
			class="{book.gesamt < 5
				? 'bg-rose-50 border border-rose-100 text-rose-600'
				: 'bg-emerald-50 border border-emerald-100/50 text-emerald-700'} px-2.5 py-1 rounded-full text-xs font-bold"
		>
			{book.gesamt}
		</span>
	</td>

	<td class="px-6 py-3 text-right">
		<svg
			class="w-5 h-5 text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
		</svg>
	</td>
</tr>
