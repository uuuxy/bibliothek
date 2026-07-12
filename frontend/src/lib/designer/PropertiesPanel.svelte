<script>
	/**
	 * @file PropertiesPanel.svelte
	 * Dynamic properties inspector for the selected canvas element.
	 *
	 * Runes in use:
	 *   $props()     — receives `selectedId` and `side`
	 *   $derived     — resolves the live element object from the store
	 *
	 * The panel adapts its controls based on `el.type`:
	 *   text / header / address / name / details / validity
	 *     → fontFamily, fontSize, color, textAlign, fontWeight, content (where editable)
	 *   image / logo
	 *     → file upload, proportional-lock toggle
	 *   photo
	 *     → read-only info (webcam trigger is on the canvas element itself)
	 *   barcode
	 *     → barcodeType selector
	 *   all
	 *     → x, y, width, height (numeric inputs), visibility toggle, z-index controls
	 */
	import { idStore, bringForward, sendBackward, removeElement } from './idDesignerStore.svelte.js';

	/** @type {{ selectedId: string|null, side: 'front'|'back' }} */
	const { selectedId, side } = $props();

	/** Currently selected element object (live reference — mutations are reactive). */
	const el = $derived(
		selectedId
			? ((side === 'front' ? idStore.front.elements : idStore.back.elements).find(
					(e) => e.id === selectedId
				) ?? null)
			: null
	);

	const isTextType = $derived(
		el && ['header', 'address', 'name', 'details', 'validity', 'text'].includes(el.type)
	);
	const isImageType = $derived(el && (el.type === 'image' || el.type === 'logo'));
	const isDynamic = $derived(el && ['name', 'details', 'validity'].includes(el.type));

	const fontFamilies = [
		{ label: 'System (Standard)', value: 'inherit' },
		{ label: 'Inter / Sans-Serif', value: 'Inter, system-ui, sans-serif' },
		{ label: 'Serif', value: 'Georgia, serif' },
		{ label: 'Monospace', value: 'ui-monospace, monospace' }
	];

	/** @param {Event} e */
	function handleImageUpload(e) {
		const files = /** @type {HTMLInputElement} */ (e.currentTarget).files;
		if (!files || !el) return;
		const file = files[0];
		if (!file) return;
		const reader = new FileReader();
		reader.onload = (ev) => {
			if (ev.target && typeof ev.target.result === 'string') el.content = ev.target.result;
		};
		reader.readAsDataURL(file);
	}

	function handleDelete() {
		if (!el) return;
		removeElement(side, el.id);
	}
</script>

<div
	class="w-full lg:w-80 bg-white border border-slate-100 p-5 rounded-2xl shadow-xl space-y-5 shrink-0 text-left overflow-y-auto max-h-[80vh]"
>
	{#if !el}
		<div class="flex flex-col items-center justify-center py-12 text-center gap-3">
			<svg
				class="w-10 h-10 text-slate-200"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="1"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5"
				/></svg
			>
			<span class="text-xs text-slate-400 font-medium">Element auf der Karte anklicken</span>
		</div>
	{:else}
		<div class="flex items-center justify-between">
			<h3 class="text-xs font-bold text-slate-600 uppercase tracking-wider">{el.id}</h3>
			{#if !['header', 'address', 'logo', 'photo', 'name', 'details', 'validity', 'barcode'].includes(el.id)}
				<button
					onclick={handleDelete}
					class="text-slate-300 hover:text-rose-500 transition-colors p-1"
					title="Element löschen"
				>
					<svg
						class="w-4 h-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
						/></svg
					>
				</button>
			{/if}
		</div>

		<!-- Visibility -->
		<div class="flex items-center justify-between">
			<span class="text-[10px] font-bold text-slate-450 uppercase">Sichtbar</span>
			<label class="relative inline-flex items-center cursor-pointer select-none">
				<input type="checkbox" bind:checked={el.show} class="sr-only peer" />
				<div
					class="w-7 h-4 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-blue-600"
				></div>
			</label>
		</div>

		<!-- Position & Size -->
		<div class="space-y-2 pt-2 border-t border-slate-100">
			<span class="text-[10px] font-bold text-slate-450 uppercase block">Position &amp; Größe</span>
			<div class="grid grid-cols-2 gap-2">
				{@render numInput('X (mm)', el.x, 0, 80, 0.5, (v) => {
					el.x = v;
				})}
				{@render numInput('Y (mm)', el.y, 0, 50, 0.5, (v) => {
					el.y = v;
				})}
				{@render numInput('Breite (mm)', el.width, 3, 85, 0.5, (v) => {
					el.width = v;
				})}
				{@render numInput('Höhe (mm)', el.height, 2, 53, 0.5, (v) => {
					el.height = v;
				})}
			</div>
		</div>

		<!-- Z-Index -->
		<div class="flex items-center gap-2 pt-2 border-t border-slate-100">
			<span class="text-[10px] font-bold text-slate-450 uppercase flex-1"
				>Ebene (z={el.zIndex})</span
			>
			<button
				onclick={() => bringForward(side, el.id)}
				class="px-2 py-1 text-[10px] bg-slate-100 hover:bg-slate-200 rounded-lg font-bold transition-colors"
				title="Nach vorne">▲</button
			>
			<button
				onclick={() => sendBackward(side, el.id)}
				class="px-2 py-1 text-[10px] bg-slate-100 hover:bg-slate-200 rounded-lg font-bold transition-colors"
				title="Nach hinten">▼</button
			>
		</div>

		<!-- Text style panel -->
		{#if isTextType && el.style}
			<div class="space-y-3 pt-2 border-t border-slate-100">
				<span class="text-[10px] font-bold text-slate-450 uppercase block">Textformatierung</span>

				{#if !isDynamic}
					<div class="space-y-1">
						<span class="text-[9px] text-slate-400 font-bold uppercase block">Inhalt</span>
						<input
							type="text"
							bind:value={el.content}
							class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
						/>
					</div>
				{/if}

				<div class="space-y-1">
					<span class="text-[9px] text-slate-400 font-bold uppercase block">Schriftart</span>
					<select
						bind:value={el.style.fontFamily}
						class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1.5 text-xs focus:outline-none"
					>
						{#each fontFamilies as ff, _i (_i)}
							<option value={ff.value}>{ff.label}</option>
						{/each}
					</select>
				</div>

				<div class="grid grid-cols-2 gap-2">
					{@render numInput('Größe (pt)', el.style.fontSize, 4, 20, 0.5, (v) => {
						el.style.fontSize = v;
					})}
					<div class="space-y-1">
						<span class="text-[9px] text-slate-400 font-bold uppercase block">Farbe</span>
						<input
							type="color"
							bind:value={el.style.color}
							class="w-full h-8 rounded-xl border border-slate-200 cursor-pointer bg-white px-1"
						/>
					</div>
				</div>

				<div class="grid grid-cols-3 gap-1">
					{@render alignBtn(el, 'left', '⬅')}
					{@render alignBtn(el, 'center', '↔')}
					{@render alignBtn(el, 'right', '➡')}
				</div>

				<label class="flex items-center gap-2 cursor-pointer">
					<input
						type="checkbox"
						checked={el.style.fontWeight === 'bold'}
						onchange={(e) => {
							el.style.fontWeight = /** @type {HTMLInputElement} */ (e.currentTarget).checked
								? 'bold'
								: 'normal';
						}}
						class="rounded border-slate-300 text-blue-600"
					/>
					<span class="text-xs text-slate-600 font-medium">Fett</span>
				</label>
			</div>
		{/if}

		<!-- Image panel -->
		{#if isImageType}
			<div class="space-y-3 pt-2 border-t border-slate-100">
				<span class="text-[10px] font-bold text-slate-450 uppercase block">Bild</span>
				<input
					type="file"
					accept="image/*"
					onchange={handleImageUpload}
					class="w-full text-xs text-slate-500 file:mr-2 file:py-1 file:px-2 file:rounded-md file:border-0 file:text-[10px] file:font-semibold file:bg-slate-100 file:text-slate-700 hover:file:bg-slate-200 cursor-pointer"
				/>
				<label class="flex items-center gap-2 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={el.proportional}
						class="rounded border-slate-300 text-blue-600"
					/>
					<span class="text-xs text-slate-600 font-medium">Proportionale Skalierung</span>
				</label>
			</div>
		{/if}
	{/if}
</div>

{#snippet numInput(label, value, min, max, step, onInput)}
	<div class="space-y-1">
		<span class="text-[9px] text-slate-400 font-bold uppercase block">{label}</span>
		<input
			type="number"
			{min}
			{max}
			{step}
			value={Math.round(value * 10) / 10}
			oninput={(e) =>
				onInput(parseFloat(/** @type {HTMLInputElement} */ (e.currentTarget).value) || 0)}
			class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
		/>
	</div>
{/snippet}

{#snippet alignBtn(el, align, icon)}
	<button
		onclick={() => {
			el.style.textAlign = align;
		}}
		class="py-1 rounded-lg text-sm transition-colors {el.style?.textAlign === align
			? 'bg-blue-600 text-white'
			: 'bg-slate-100 text-slate-500 hover:bg-slate-200'}"
		title={align}>{icon}</button
	>
{/snippet}
