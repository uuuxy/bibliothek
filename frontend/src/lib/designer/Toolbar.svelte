<script>
	import { Printer } from '@lucide/svelte';
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
	import Button from '../components/ui/Button.svelte';
	import ToolbarAuswahl from './ToolbarAuswahl.svelte';

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

	// Sichtbar unterscheidbare, druck-taugliche Töne. Die vorherigen Themes lagen bei
	// -50/40-Opazität (praktisch weiß) — die Auswahl war in der Vorschau nicht erkennbar.
	const themes = [
		{ value: 'bg-white text-black border-slate-200', name: 'Weiß' },
		{ value: 'bg-slate-100 text-slate-900 border-slate-300', name: 'Grau' },
		{
			value: 'bg-linear-to-tr from-emerald-100 to-teal-100 text-emerald-950 border-emerald-300',
			name: 'Smaragd'
		},
		{
			value: 'bg-linear-to-tr from-sky-100 to-indigo-100 text-indigo-950 border-sky-300',
			name: 'Blau'
		},
		{
			value: 'bg-linear-to-tr from-amber-100 to-orange-100 text-amber-950 border-amber-300',
			name: 'Bernstein'
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
		<Button onclick={onPrint} class="px-5">
			<Printer class="h-4 w-4" aria-hidden="true" />
			{side === 'back' ? 'Rückseiten drucken' : 'Vorderseiten drucken'}
		</Button>
	</div>

	<ToolbarAuswahl
		{classesList}
		{selectedKlasse}
		{onKlasse}
		{loadingStudents}
		{barcodeType}
		{onBarcodeType}
		{currentTheme}
		{themes}
		{setTheme}
		{zoom}
		{onZoom}
	/>

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

		<Button variant="secondary" size="sm" onclick={() => addTextElement(side)}>+ Text</Button>

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
