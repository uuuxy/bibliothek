<script>
	/**
	 * @file Toolbar.svelte
	 * Top control bar for the canvas ID-card designer.
	 *
	 * Runes in use:
	 *   $props()   — receives `zoom`, `onZoom`, `side`, `onSide`,
	 *                `printMode`, `onPrintMode`, `printSide`, `onPrintSide`,
	 *                `onPrint`, `barcodeType`, `onBarcodeType`
	 *
	 * Element additions (text, multi-image) are handled directly here via
	 * `addTextElement` and `addImageElements` from the shared store.
	 */
	import {
		idStore,
		addTextElement,
		addImageElements,
		addBoxElement
	} from './idDesignerStore.svelte.js';
	import { WALDGRUEN_THEME } from './ausweisVorlagen.js';
	import Button from '../components/ui/Button.svelte';
	import ToolbarAuswahl from './ToolbarAuswahl.svelte';
	import ToolbarDruck from './ToolbarDruck.svelte';

	/** @type {HTMLInputElement | undefined} */
	let bildUploadEl = $state();

	/**
	 * @type {{
	 *   zoom: number, onZoom: (v: number) => void,
	 *   side: 'front'|'back', onSide: (s: 'front'|'back') => void,
	 *   printMode: 'card'|'etikett', onPrintMode: (m: 'card'|'etikett') => void,
	 *   onPrint: () => void,
	 *   barcodeType: 'code39'|'qr', onBarcodeType: (t: 'code39'|'qr') => void,
	 *   onVorlage: (kennung: string) => void,
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
		barcodeType,
		onBarcodeType,
		onVorlage,
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
		},
		// Dunkler Verlauf der Vorlage „Waldgrün" — steht mit in der Liste, damit das
		// Hintergrund-Dropdown nach dem Anwenden der Vorlage eine Auswahl anzeigt.
		{ value: WALDGRUEN_THEME, name: 'Waldgrün' }
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
	<ToolbarDruck {printMode} {onPrintMode} {side} {onPrint} />

	<ToolbarAuswahl
		{barcodeType}
		{onBarcodeType}
		{currentTheme}
		{themes}
		{setTheme}
		{onVorlage}
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

		<Button variant="secondary" size="sm" onclick={() => addBoxElement(side)}>+ Fläche</Button>

		<!-- Multi-image upload: die eigentliche Auswahl-Fläche ist die geteilte Button-
		     Komponente (dieselbe Pillenform/Höhe wie "+ Text" daneben, statt eines eigens
		     gestylten <label>, das bei jeder Änderung an Button.svelte hätte auseinanderlaufen
		     können); das <input type="file"> bleibt unsichtbar und wird nur programmatisch
		     ausgelöst. -->
		<Button variant="secondary" size="sm" onclick={() => bildUploadEl?.click()}>+ Bild(er)</Button>
		<input
			bind:this={bildUploadEl}
			type="file"
			accept="image/*"
			multiple
			class="sr-only"
			onchange={handleMultiImageUpload}
		/>

		{#if previewStudent}
			<!-- Ausdrücklich als Muster benannt: Hier stand früher ein echter Schüler aus
			     der gewählten Klasse, und man konnte meinen, dieser Bildschirm drucke ihn.
			     Gedruckt wird in der Schülerdatei. -->
			<span class="text-sm text-slate-500 font-medium ml-auto">
				Musterkarte: {previewStudent.vorname}
				{previewStudent.nachname}
			</span>
		{/if}
	</div>
</div>

{#snippet toggleGroup(options, active, onChange)}
	<div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200/40 text-sm shrink-0">
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
