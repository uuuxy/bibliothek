<script>
	import { getSubjectColor, getStockDotColor, formatDate } from '../bookHelpers.js';
	import BuchKarteCover from './BuchKarteCover.svelte';

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
	 *     medientyp?: string
	 *   },
	 *   onclick?: () => void,
	 *   onEditClick?: () => void
	 * }}
	 */
	let { book, onclick, onEditClick } = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
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
			const cleanIsbn = book.isbn.replace(/[- ]/g, '');
			candidates.push(
				`https://books.google.com/books/content?id=&vid=ISBN:${cleanIsbn}&printsec=frontcover&img=1&zoom=1`
			);
			candidates.push(`https://covers.openlibrary.org/b/isbn/${cleanIsbn}-L.jpg`);
		}
		coverCandidates = candidates;
		currentCandidateIndex = 0;
		coverFailed = candidates.length === 0;
	});
</script>

<article
	class="bg-white rounded-2xl border border-slate-200 flex flex-col h-full group overflow-hidden hover:border-blue-300 hover:shadow-md transition-all duration-300 shadow-sm cursor-pointer relative"
	{onclick}
>
	<!-- Quick-Edit Stift-Icon (sichtbar beim Hover) -->
	{#if onEditClick}
		<button
			class="absolute top-2 right-2 z-10 p-1.5 rounded-lg bg-white/80 backdrop-blur-sm border border-slate-200 text-slate-400 hover:text-blue-600 hover:border-blue-300 hover:bg-blue-50 opacity-0 group-hover:opacity-100 transition-all duration-200 shadow-sm cursor-pointer"
			onclick={(e) => {
				e.stopPropagation();
				onEditClick();
			}}
			title="Schnell bearbeiten"
			aria-label="Buch schnell bearbeiten"
		>
			<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
				/>
			</svg>
		</button>
	{/if}
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
		<BuchKarteCover
			subject={book.subject}
			title={book.title}
			author={book.author}
			medientyp={book.medientyp}
			isbn={book.isbn}
		/>
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
				onclick={(e) => {
					e.stopPropagation();
					copyIsbn(book.isbn);
				}}
				title={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') + ' kopieren'}
				aria-label={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') +
					' kopieren'}
			>
				<span
					>{book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {book.isbn ||
						'-'}</span
				>
				{#if book.isbn}
					{#if copied}
						<span class="text-blue-600 text-[10px] font-sans font-bold">Kopiert!</span>
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
				<span class="{getSubjectColor(book.subject)} text-[10px] font-bold px-2 py-0.5 rounded-md">
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
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
					></path></svg
				>
				<span>
					Zuletzt geprüft: {formatDate(book.lastCounted) || 'Unbekannt'}
				</span>
			</div>

			<div class="pt-3 border-t border-slate-100 flex justify-between items-center">
				<span class="text-xs font-semibold text-slate-400">Verfügbar</span>
				<div class="flex items-center gap-2">
					<span class="w-2 h-2 rounded-full {getStockDotColor(book.verfuegbar || 0)}"></span>
					<span class="text-lg font-extrabold text-slate-800">{book.verfuegbar || 0}</span>
					{#if book.gesamt !== undefined}
						<span class="text-xs text-slate-500 font-medium">/ {book.gesamt}</span>
					{/if}
				</div>
			</div>
		</div>
	</div>
</article>
