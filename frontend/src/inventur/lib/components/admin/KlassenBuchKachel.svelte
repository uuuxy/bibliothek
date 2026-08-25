<script>
	import { coverKandidaten } from '../../../../lib/utils/coverSrc.js';
	import { getSubjectGradient, getSpineGradient } from '../../bookHelpers.js';
	import { Pencil } from '@lucide/svelte';

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
	 *   onEdit?: (book: any) => void,
	 *   bearbeitbar?: boolean
	 * }}
	 * bearbeitbar=false: reine Ansicht (Kollegiums-Portal, 25.08.2026). Ohne den Schalter
	 * wäre jede Kachel dort ein Knopf „Bearbeiten", hinter dem für die Rolle ein 403 liegt.
	 */
	let { book, onEdit = undefined, bearbeitbar = true } = $props();

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

<!-- Zwei Hüllen, ein Inhalt: Die Rolle „button" muss für svelte-check STATISCH am Element
     stehen (a11y_no_noninteractive_tabindex) — ein Ausdruck role={…} sähe für den Prüfer
     wie ein Div mit tabindex aus. Der Inhalt selbst liegt einmal im Snippet. -->
{#snippet inhalt()}
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

		<!-- Hover Overlay -->
		{#if bearbeitbar}
			<div
				class="absolute inset-0 bg-blue-600/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-20 flex items-center justify-center backdrop-blur-[1px]"
			>
				<div
					class="bg-blue-600 text-white font-bold text-xs px-3 py-1.5 rounded-full shadow-lg flex items-center gap-1.5 transform translate-y-4 group-hover:translate-y-0 transition-transform duration-300"
				>
					<Pencil class="w-4 h-4" aria-hidden="true" />
					<span>Bearbeiten</span>
				</div>
			</div>
		{/if}

		<div class="absolute bottom-2 right-2 flex flex-col gap-1 items-end z-10">
			{#if book.track}
				<div
					class="bg-white/90 border border-slate-200 backdrop-blur-xs px-1.5 py-0.5 rounded text-[8px] font-medium text-slate-700 shadow-sm"
				>
					{book.track}
				</div>
			{/if}
		</div>
	</div>

	<h3
		class="text-xs font-bold text-slate-800 line-clamp-2 leading-tight {bearbeitbar
			? 'group-hover:text-blue-600'
			: ''} transition-colors px-1"
	>
		{book.title}
	</h3>
{/snippet}

{#if bearbeitbar}
	<div
		class="group flex cursor-pointer flex-col justify-between rounded-xl border border-slate-200 bg-white p-2.5 shadow-sm transition-all hover:-translate-y-1 hover:border-blue-300 hover:shadow-md"
		onclick={handleEditClick}
		role="button"
		tabindex="0"
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				handleEditClick(e);
			}
		}}
	>
		{@render inhalt()}
	</div>
{:else}
	<div
		class="group flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-2.5 shadow-sm"
	>
		{@render inhalt()}
	</div>
{/if}
