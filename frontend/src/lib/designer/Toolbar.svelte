<script>
	/**
	 * @file Toolbar.svelte
	 * Top control bar for the canvas ID-card designer.
	 *
	 * Runes in use:
	 *   $props()   — receives `zoom`, `onZoom`, `side`, `onSide`,
	 *                `printMode`, `onPrintMode`, `printSide`, `onPrintSide`,
	 *                `onPrint`, `classesList`, `selectedKlasse`, `onKlasse`,
	 *                `barcodeType`, `onBarcodeType`, `loadingStudents`
	 *
	 * Element additions (text, multi-image) are handled directly here via
	 * `addTextElement` and `addImageElements` from the shared store.
	 */
	import { idStore, addTextElement, addImageElements } from './idDesignerStore.svelte.js';

	/**
	 * @type {{
	 *   zoom: number, onZoom: (v: number) => void,
	 *   side: 'front'|'back', onSide: (s: 'front'|'back') => void,
	 *   printMode: 'card'|'a4', onPrintMode: (m: 'card'|'a4') => void,
	 *   onPrint: () => void,
	 *   classesList: string[], selectedKlasse: string, onKlasse: (k: string) => void,
	 *   barcodeType: 'code39'|'qr', onBarcodeType: (t: 'code39'|'qr') => void,
	 *   loadingStudents: boolean,
	 *   previewStudent: any,
	 * }}
	 */
	const {
		zoom,
		onZoom,
		side,
		onSide,
		printMode,
		onPrintMode,
		onPrint,
		classesList,
		selectedKlasse,
		onKlasse,
		barcodeType,
		onBarcodeType,
		loadingStudents,
		previewStudent
	} = $props();

	const themes = [
		{ value: 'bg-white text-black border-slate-200', name: 'Standard Weiß' },
		{ value: 'bg-slate-50 text-slate-900 border-slate-200', name: 'Dezentes Grau' },
		{
			value: 'bg-linear-to-tr from-emerald-50/40 to-teal-50/40 text-zinc-900 border-emerald-100',
			name: 'Smaragd'
		},
		{
			value: 'bg-linear-to-tr from-blue-50/40 to-indigo-50/40 text-zinc-900 border-blue-100',
			name: 'Klassik'
		}
	];

	/** Current theme for the active side. */
	const currentTheme = $derived(side === 'front' ? idStore.front.theme : idStore.back.theme);

	/** @param {string} value */
	function setTheme(value) {
		if (side === 'front') idStore.front.theme = value;
		else idStore.back.theme = value;
	}

	/** Handle multi-file image upload → creates one element per file. */
	/** @param {Event} e */
	async function handleMultiImageUpload(e) {
		const files = /** @type {HTMLInputElement} */ (e.currentTarget).files;
		if (!files || files.length === 0) return;
		/** @type {string[]} */
		const dataUrls = await Promise.all(
			Array.from(files).map(
				(file) =>
					new Promise((resolve) => {
						const reader = new FileReader();
						reader.onload = (ev) => resolve(/** @type {string} */ (ev.target?.result ?? ''));
						reader.readAsDataURL(file);
					})
			)
		);
		addImageElements(side, dataUrls);
		// Reset input so the same files can be re-selected
		/** @type {HTMLInputElement} */ (e.currentTarget).value = '';
	}
</script>

<div class="w-full space-y-4 no-print">
	<!-- Row 1: Print controls -->
	<div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 pb-4">
		{@render toggleGroup(
			[
				{ value: 'card', label: 'Kartendrucker' },
				{ value: 'a4', label: 'A4-Bogen' }
			],
			printMode,
			(v) => onPrintMode(/** @type {'card'|'a4'} */ (v))
		)}
		<button
			onclick={onPrint}
			class="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold transition-all flex items-center gap-2 shadow-xs cursor-pointer text-xs"
		>
			🖨️ {side === 'back' ? 'Rückseiten drucken' : 'Vorderseiten drucken'}
		</button>
	</div>

	<!-- Row 2: Class / barcode selectors -->
	<div
		class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 bg-slate-50 border border-slate-100 rounded-2xl p-4"
	>
		<div class="space-y-1">
			<span class="text-[10px] uppercase font-bold text-slate-450">Klasse</span>
			{#if classesList.length > 0}
				<select
					value={selectedKlasse}
					onchange={(e) => onKlasse(/** @type {HTMLSelectElement} */ (e.currentTarget).value)}
					class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs focus:outline-none"
				>
					{#each classesList as kl, _i (_i)}
						<option value={kl}>Klasse {kl}</option>
					{/each}
				</select>
			{:else}
				<div class="text-xs text-slate-400 font-medium py-2">
					{loadingStudents ? 'Lade…' : 'Keine Klassen'}
				</div>
			{/if}
		</div>

		<div class="space-y-1">
			<span class="text-[10px] uppercase font-bold text-slate-450">Barcode-Typ</span>
			<select
				value={barcodeType}
				onchange={(e) =>
					onBarcodeType(
						/** @type {'code39'|'qr'} */ (/** @type {HTMLSelectElement} */ (e.currentTarget).value)
					)}
				class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs focus:outline-none"
			>
				<option value="code39">Code39 (1D)</option>
				<option value="qr">QR-Code (2D)</option>
			</select>
		</div>

		<div class="space-y-1">
			<span class="text-[10px] uppercase font-bold text-slate-450">Karten-Hintergrund</span>
			<select
				value={currentTheme}
				onchange={(e) => setTheme(/** @type {HTMLSelectElement} */ (e.currentTarget).value)}
				class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs focus:outline-none"
			>
				{#each themes as t, _i (_i)}
					<option value={t.value}>{t.name}</option>
				{/each}
			</select>
		</div>

		<div class="space-y-1">
			<span class="text-[10px] uppercase font-bold text-slate-450">Zoom</span>
			<div class="flex items-center gap-2">
				<input
					type="range"
					min="80"
					max="300"
					step="5"
					value={zoom}
					oninput={(e) => onZoom(parseInt(/** @type {HTMLInputElement} */ (e.currentTarget).value))}
					class="accent-blue-600 h-1 bg-slate-200 rounded-lg cursor-pointer flex-1"
				/>
				<span class="text-xs font-bold text-blue-600 w-10 text-right">{zoom}%</span>
			</div>
		</div>
	</div>

	<!-- Row 3: Side tab + Add-element buttons -->
	<div class="flex flex-wrap items-center gap-3">
		{@render toggleGroup(
			[
				{ value: 'front', label: '🪪 Vorderseite' },
				{ value: 'back', label: '↩ Rückseite' }
			],
			side,
			(v) => onSide(/** @type {'front'|'back'} */ (v))
		)}

		{#if side === 'back'}
			<button
				onclick={() => addTextElement()}
				class="px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-xs font-bold text-slate-700 transition-colors cursor-pointer"
			>
				+ Text
			</button>
		{/if}

		<!-- Multi-image upload -->
		<label
			class="px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-xs font-bold text-slate-700 transition-colors cursor-pointer"
		>
			+ Bild(er)
			<input
				type="file"
				accept="image/*"
				multiple
				class="sr-only"
				onchange={handleMultiImageUpload}
			/>
		</label>

		{#if previewStudent}
			<span class="text-xs text-slate-500 font-medium ml-auto">
				Vorschau: {previewStudent.vorname}
				{previewStudent.nachname} (Klasse {previewStudent.klasse})
			</span>
		{/if}
	</div>
</div>

{#snippet toggleGroup(options, active, onChange)}
	<div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200/40 text-xs shrink-0">
		{#each options as opt, _i (_i)}
			<button
				onclick={() => onChange(opt.value)}
				class="px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer {active === opt.value
					? 'bg-white text-slate-800 shadow-xs'
					: 'text-slate-500 hover:text-slate-700'}">{opt.label}</button
			>
		{/each}
	</div>
{/snippet}
