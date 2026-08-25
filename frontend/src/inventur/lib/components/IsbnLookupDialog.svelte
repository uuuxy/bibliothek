<script>
	import Button from '../../../lib/components/ui/Button.svelte';
	import Select from '../../../lib/components/ui/Select.svelte';
	import Feld from '../../../lib/components/ui/Feld.svelte';
	// Alias: coverSrc ist in dieser Komponente bereits der Name des Anzeige-Zustands.
	import { coverSrc as proxyCover } from '../../../lib/utils/coverSrc.js';

	const klassenstufen = [5, 6, 7, 8, 9, 10].map((g) => ({ value: g, label: String(g) }));
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
		// Über den eigenen Proxy statt per Hotlink — siehe utils/coverSrc.js.
		return isbn ? proxyCover(`https://covers.openlibrary.org/b/isbn/${isbn}-L.jpg`, isbn) : '';
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
		if (
			!subject ||
			Number.isNaN(gradeNum) ||
			gradeNum < 1 ||
			Number.isNaN(stockNum) ||
			stockNum < 0
		)
			return;
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
	<div
		class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4"
		role="dialog"
		aria-modal="true"
	>
		<div
			class="w-full max-w-xl rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800"
		>
			<h3 class="text-lg font-bold text-slate-900">ISBN bestätigt</h3>
			<div class="mt-4 grid gap-4 sm:grid-cols-[120px,1fr]">
				<div
					class="h-36 overflow-hidden rounded-2xl border border-slate-200 bg-slate-50 flex items-center justify-center relative"
				>
					{#if coverSrc}
						<img
							src={coverSrc}
							alt={data.title}
							class="h-full w-full object-cover"
							onerror={onCoverError}
							onload={onCoverLoad}
						/>
					{:else}
						<div class="grid h-full place-items-center text-xs text-slate-500 font-semibold">
							Kein Cover
						</div>
					{/if}
				</div>
				<div>
					<p class="font-bold text-slate-900">{data.title || 'Unbekannter Titel'}</p>
					<p class="text-sm text-slate-500 mt-0.5">{data.author || 'Unbekannter Autor'}</p>
					<p class="mt-2 text-xs text-slate-400">ISBN: {data.isbn}</p>
				</div>
			</div>

			<div class="mt-5 grid gap-3 sm:grid-cols-2">
				<Feld id="isbn-fach" label="Fach" bind:value={subject} />
				<!-- Gleiche drei Subgrid-Zeilen wie das Feld daneben, sonst sitzt das
				     Auswahlfeld eine Zeile tiefer als das Fach. -->
				<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
					<label for="isbn-klassenstufe" class="text-sm font-medium text-on-surface-variant"
						>Klassenstufe</label
					>
					<Select
						id="isbn-klassenstufe"
						bind:value={grade}
						options={klassenstufen}
						placeholder="Klasse wählen"
					/>
				</div>
				<Feld id="isbn-bestand" label="Bestand" type="number" min="0" bind:value={stock} />
			</div>

			<div class="mt-6 flex justify-end gap-3">
				<Button variant="secondary" size="lg" onclick={onCancel} disabled={busy} class="px-5">
					Abbrechen
				</Button>
				<Button size="lg" onclick={save} disabled={busy} class="px-5">Speichern</Button>
			</div>
		</div>
	</div>
{/if}
