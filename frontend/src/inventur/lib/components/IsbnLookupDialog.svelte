<script>
	/**
	 * @type {{
	 *   data: any,
	 *   busy?: boolean,
	 *   onCancel?: () => void,
	 *   onSave?: (savedBook: any) => void
	 * }}
	 */
	let { data = null, busy = false, onCancel = () => {}, onSave = () => {} } = $props();
	let subject = $state('');
	let grade = $state('');
	let stock = $state('');
	let coverSrc = $state('');
	let triedFallback = $state(false);

	/**
	 * @param {string} isbn
	 */
	function fallbackCover(isbn) {
		return isbn ? `https://covers.openlibrary.org/b/isbn/${isbn}-L.jpg` : '';
	}

	function onCoverError() {
		const fallback = fallbackCover(data?.isbn);
		if (!triedFallback && fallback && coverSrc !== fallback) {
			coverSrc = fallback;
			triedFallback = true;
			return;
		}
		coverSrc = '';
	}

	/**
	 * @param {Event} event
	 */
	function onCoverLoad(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		// OpenLibrary returns a 43-byte 1x1 pixel image when no cover is found
		if (image.naturalWidth < 20 || image.naturalHeight < 20) {
			onCoverError();
		}
	}

	$effect(() => {
		if (!data) return;
		subject = data.subject ?? 'Mathe';
		grade = data.grade ?? '7';
		stock = '';
		const fallback = fallbackCover(data.isbn);
		coverSrc = data.coverUrl || fallback;
		triedFallback = !data.coverUrl;
	});

	function save() {
		const gradeNum = Number.parseInt(grade, 10);
		const stockNum = Number.parseInt(stock, 10);
		if (!subject || Number.isNaN(gradeNum) || gradeNum < 1 || Number.isNaN(stockNum) || stockNum < 0) return;
		onSave({
			isbn: data.isbn,
			title: data.title,
			author: data.author,
			coverUrl: data.coverUrl,
			subject,
			gradeLevel: gradeNum,
			stock: stockNum
		});
	}
</script>

{#if data}
	<div class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4" role="dialog" aria-modal="true">
		<div class="w-full max-w-xl rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800">
			<h3 class="text-lg font-bold text-slate-900">ISBN bestätigt</h3>
			<div class="mt-4 grid gap-4 sm:grid-cols-[120px,1fr]">
				<div class="h-36 overflow-hidden rounded-2xl border border-slate-200 bg-slate-50 flex items-center justify-center relative">
					{#if coverSrc}
						<img src={coverSrc} alt={data.title} class="h-full w-full object-cover" onerror={onCoverError} onload={onCoverLoad} />
					{:else}
						<div class="grid h-full place-items-center text-xs text-slate-450 font-semibold">Kein Cover</div>
					{/if}
				</div>
				<div>
					<p class="font-bold text-slate-900">{data.title || 'Unbekannter Titel'}</p>
					<p class="text-sm text-slate-500 mt-0.5">{data.author || 'Unbekannter Autor'}</p>
					<p class="mt-2 text-xs text-slate-400">ISBN: {data.isbn}</p>
				</div>
			</div>

			<div class="mt-5 grid gap-3 sm:grid-cols-2">
				<label class="block text-xs font-semibold uppercase tracking-wider text-slate-400">Fach
					<input type="text" bind:value={subject} class="mt-1.5 w-full rounded-xl border border-slate-350 bg-white px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" />
				</label>
				<label class="block text-xs font-semibold uppercase tracking-wider text-slate-400">Klassenstufe
					<select bind:value={grade} class="mt-1.5 w-full rounded-xl border border-slate-350 bg-white px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all cursor-pointer">
						{#each [5,6,7,8,9,10] as g (g)}
							<option value={g}>{g}</option>
						{/each}
					</select>
				</label>
			</div>
			<label class="mt-4 block text-xs font-semibold uppercase tracking-wider text-slate-400">Bestand
				<input type="number" min="0" bind:value={stock} class="mt-1.5 w-full rounded-xl border border-slate-350 bg-white px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" />
			</label>

			<div class="mt-6 flex justify-end gap-3">
				<button onclick={onCancel} disabled={busy} class="rounded-xl bg-slate-100 px-5 py-2.5 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer">Abbrechen</button>
				<button onclick={save} disabled={busy} class="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-bold text-white hover:bg-blue-700 disabled:opacity-60 transition-colors cursor-pointer">Speichern</button>
			</div>
		</div>
	</div>
{/if}
