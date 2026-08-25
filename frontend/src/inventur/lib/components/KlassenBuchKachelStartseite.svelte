<script>
	import { coverKandidaten } from '../../../lib/utils/coverSrc.js';
	import { getSubjectGradient, getSpineGradient } from '../bookHelpers.js';

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
	 *   onclick?: (event: Event) => void
	 * }}
	 */
	let { book, getStockColor, onclick } = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
	let coverFailed = $state(false);

	$effect(() => {
		const candidates = [];
		candidates.push(...coverKandidaten(book?.coverUrl, book?.isbn));
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

<!-- Anklickbare Kachel — ohne onclick (Nur-Lese im Lehrerportal) bewusst ohne Rolle,
     Fokus und Zeiger (tote Tür); das ignore: Rolle und tabindex hängen GEMEINSAM an
     onclick, statisch sieht der Compiler das nicht. w-40 stammte aus dem Karussell. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	class="group transition-all duration-300 bg-white rounded-xl p-2.5 border border-slate-200 shadow-sm flex flex-col justify-between {onclick
		? 'cursor-pointer hover:scale-[1.02] hover:-translate-y-1 hover:border-blue-300 hover:shadow-md'
		: ''}"
	role={onclick ? 'button' : undefined}
	tabindex={onclick ? 0 : undefined}
	{onclick}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onclick?.(e);
		}
	}}
>
	<div class="w-full aspect-2/3 rounded-xl overflow-hidden shadow-sm mb-3 relative bg-slate-50">
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
				class="w-full h-full flex flex-col justify-between p-3.5 relative shadow-inner {getSubjectGradient(
					book.subject
				)} border border-slate-200/30 rounded-xl"
			>
				<div
					class="absolute left-0 top-0 bottom-0 w-2 bg-linear-to-b {getSpineGradient(
						book.subject
					)} opacity-90 shadow-sm rounded-l-xl"
				></div>

				<div class="pl-1.5 pr-0.5 pt-0.5 text-left">
					<span class="text-[7px] text-white/80 font-extrabold">{book.subject}</span>
					<h4 class="text-[9px] font-extrabold text-white leading-snug line-clamp-4 mt-1">
						{book.title}
					</h4>
				</div>

				<div class="pl-1.5 pr-0.5 pb-0.5 text-left">
					<p class="text-[7px] font-semibold text-white/60 truncate">
						{book.author || 'Unbekannter Autor'}
					</p>
				</div>
			</div>
		{/if}

		<div class="absolute bottom-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<span
					class="bg-white/90 border border-slate-200 backdrop-blur-xs px-1.5 py-0.5 rounded text-[8px] font-medium text-slate-700 shadow-sm"
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
		<p class="text-label-small text-slate-400 mb-2 truncate">
			{book.isbn || '-'}
		</p>
		<div class="flex items-center gap-1.5">
			<span class="w-2 h-2 rounded-full {getStockColor(book.verfuegbar)}"></span>
			<span class="text-label-small font-bold text-slate-500"
				>{book.verfuegbar}{#if book.gesamt !== undefined}/{book.gesamt}{/if} Stück</span
			>
		</div>
	</div>
</div>
