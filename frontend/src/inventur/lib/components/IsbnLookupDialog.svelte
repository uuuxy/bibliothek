<!-- @component IsbnLookupDialog — die per ISBN gefundenen Titeldaten bestätigen und mit
     Fach, Klassenstufe und Bestand anlegen. Seit 07.09.2026 auf Modal.svelte (Register
     05.09.). -->
<script>
	import Modal from '../../../lib/Modal.svelte';
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
		subject = data.subject ?? 'Mathematik';
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
	<Modal open={true} onclose={onCancel} size="xl" beschriftetDurch="isbn-titel">
		<div class="p-6 text-on-surface">
			<h3 id="isbn-titel" class="text-lg font-bold text-on-surface">ISBN bestätigt</h3>
			<div class="mt-4 grid gap-4 sm:grid-cols-[120px,1fr]">
				<div
					class="h-36 overflow-hidden rounded-2xl border border-outline-variant bg-surface-container flex items-center justify-center relative"
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
						<div
							class="grid h-full place-items-center text-xs text-on-surface-variant font-semibold"
						>
							Kein Cover
						</div>
					{/if}
				</div>
				<div>
					<p class="font-bold text-on-surface">{data.title || 'Unbekannter Titel'}</p>
					<p class="text-sm text-on-surface-variant mt-0.5">{data.author || 'Unbekannter Autor'}</p>
					<p class="mt-2 text-xs text-on-surface-variant">ISBN: {data.isbn}</p>
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
	</Modal>
{/if}
