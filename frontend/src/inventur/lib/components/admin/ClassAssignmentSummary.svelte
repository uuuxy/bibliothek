<script>
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { coverSrc } from '../../../../lib/utils/coverSrc.js';
	let {
		selectedClasses = [],
		selectedBookIds = new Set(),
		selectedBooksList = [],
		isSaving = false,
		isUpdate = false,
		onToggleBook = () => {},
		onsave = () => {}
	} = $props();

	/**
	 * @param {Event} event
	 */
	function handleImageError(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		const fallback = image.dataset.fallback || '';
		const isFallback = image.dataset.isFallback === '1';
		const retryCount = Number(image.dataset.retryCount || '0');

		// Network hiccups are common for external cover hosts. Retry once first.
		if (retryCount < 1) {
			image.dataset.retryCount = String(retryCount + 1);
			const separator = image.src.includes('?') ? '&' : '?';
			image.src = `${image.src}${separator}retry=${Date.now()}`;
			return;
		}

		// Try exactly one deterministic fallback URL before showing placeholder.
		if (!isFallback && fallback && image.src !== fallback) {
			image.dataset.isFallback = '1';
			image.dataset.retryCount = '0';
			image.src = fallback;
			return;
		}

		image.style.display = 'none';
		const nextEl = /** @type {HTMLElement|null} */ (image.nextElementSibling);
		if (nextEl) nextEl.style.display = 'flex';
	}

	/**
	 * Ersatzcover über OpenLibrary — aber über den eigenen Proxy, nicht per Hotlink.
	 *
	 * Vorher stand hier die openlibrary.org-URL direkt im src. Damit meldete sich der
	 * Browser JEDER Lehrkraft bei jedem Öffnen dieser Ansicht bei einem Dritten und
	 * übermittelte dabei, welche ISBNs die Schule gerade einer Klasse zuteilt. Der
	 * Cover-Proxy (/api/images/cover) holt dasselbe Bild serverseitig, hat
	 * covers.openlibrary.org auf seiner Allowlist und legt das Ergebnis lokal ab —
	 * der Browser spricht nur noch mit dem eigenen Server.
	 *
	 * Das ist zugleich die Voraussetzung dafür, dass die Content-Security-Policy
	 * img-src ohne https: auskommt (siehe internal/middleware/security.go).
	 *
	 * @param {string|number|null|undefined} isbn
	 */
	function fallbackCover(isbn) {
		if (!isbn) return '';
		const cleaned = String(isbn).replace(/[^0-9Xx]/g, '');
		if (!cleaned) return '';
		return coverSrc(`https://covers.openlibrary.org/b/isbn/${cleaned}-M.jpg`, cleaned);
	}
</script>

<!-- Stand zweimal wortgleich im Markup (Ersatz hinter dem <img> und Buch ohne
     Cover). Der Aufrufer bestimmt nur noch die Sichtbarkeit. -->
{#snippet buchPlatzhalter(anzeige)}
	<div class="w-full h-full {anzeige} items-center justify-center bg-slate-100 text-slate-300">
		<svg
			width="24"
			height="24"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
			><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20" /></svg
		>
	</div>
{/snippet}

<div
	class="px-4 sm:px-6 py-4 sm:py-6 border-b border-surface-variant/20 flex items-center justify-between"
>
	<h3 class="text-xl font-bold text-slate-900">Auswahl</h3>
	<div class="bg-slate-100 px-3 py-1.5 rounded-full text-sm font-bold text-slate-800">
		{selectedBookIds.size}
	</div>
</div>

<div
	class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-emerald-200 [&::-webkit-scrollbar-thumb]:rounded-full p-4 space-y-2"
>
	{#if selectedBooksList.length === 0}
		<div class="h-full flex flex-col items-center justify-center text-center p-8 opacity-40">
			<svg
				width="48"
				height="48"
				class="text-slate-400 mb-4"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				><rect width="18" height="18" x="3" y="3" rx="2" /><path d="M3 9h18" /><path
					d="M9 21V9"
				/></svg
			>
			<p class="text-sm font-medium text-slate-500">Deine Auswahl ist noch leer</p>
		</div>
	{:else}
		{#each selectedBooksList as book (book.id)}
			{@const primaryCoverUrl = coverSrc(book.coverUrl, book.isbn)}
			{@const fallbackCoverUrl = fallbackCover(book.isbn)}
			{@const coverUrl = primaryCoverUrl || fallbackCoverUrl}
			<div
				class="flex items-center gap-3.5 hover:bg-emerald-50 p-2 rounded-xl transition-colors group"
			>
				<!-- Nur EIN bg-: Das zusätzliche bg-white war wirkungslos, weil
				     .bg-surface-container aus altlasten.css im Bundle dahinter landet
				     (gemessen: rgb(238,237,241)). Siehe docs/SECURITY.md. -->
				<div class="w-10 h-14 rounded overflow-hidden shrink-0 bg-surface-container shadow-sm">
					{#if coverUrl}
						<img
							src={coverUrl}
							data-fallback={fallbackCoverUrl}
							data-is-fallback={primaryCoverUrl ? '0' : '1'}
							data-retry-count="0"
							alt=""
							loading="eager"
							decoding="async"
							class="w-full h-full object-cover"
							onerror={handleImageError}
						/>
						{@render buchPlatzhalter('hidden')}
					{:else}
						{@render buchPlatzhalter('flex')}
					{/if}
				</div>
				<p class="font-medium text-slate-800 grow truncate leading-tight">
					{book.title}
				</p>
				<button
					onclick={() => onToggleBook(book.id)}
					class="text-slate-400 hover:text-red-500 p-1 rounded-full transition-colors focus-visible:ring-2 focus-visible:ring-blue-500 focus:outline-none"
					title="Buch entfernen"
					aria-label="Buch entfernen"
				>
					<svg
						aria-hidden="true"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"
						></line></svg
					>
				</button>
			</div>
		{/each}
	{/if}
</div>

<footer class="p-4 sm:p-6 bg-white border-t border-surface-variant/20 flex flex-col gap-4">
	<Button
		disabled={selectedClasses.length === 0 || (!isUpdate && selectedBookIds.size === 0) || isSaving}
		onclick={(e) => onsave(e)}
		class="h-auto w-full p-5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-slate-300 disabled:text-slate-500 disabled:opacity-100 text-base tracking-wide shadow-lg"
	>
		<svg
			fill="none"
			height="24"
			stroke="currentColor"
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
			viewBox="0 0 24 24"
			width="24"
			xmlns="http://www.w3.org/2000/svg"
			><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline
				points="17 21 17 13 7 13 7 21"
			></polyline><polyline points="7 3 7 8 15 8"></polyline></svg
		>
		{isSaving ? 'SPEICHERT...' : 'AUSWAHL SPEICHERN'}
	</Button>
</footer>
