<script>
	/**
	 * @file PropertiesPanel.svelte
	 * Eigenschaften des ausgewählten Ausweis-Elements. Was JEDES Element hat, steht
	 * hier: Lage, Größe, Ebene, Sichtbarkeit. Die artspezifischen Teile hängen an
	 * el.type — Schrift in PropertiesText, Bild-Upload weiter unten; photo und
	 * barcode bringen ihre Bedienung am Element selbst mit.
	 */
	import { idStore, bringForward, sendBackward, removeElement } from './idDesignerStore.svelte.js';
	import PropertiesText from './PropertiesText.svelte';
	import ZahlenFeld from './ZahlenFeld.svelte';
	import { MousePointer2, Trash2 } from '@lucide/svelte';

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
		el && ['header', 'address', 'name', 'validity', 'text'].includes(el.type)
	);
	const isImageType = $derived(el && (el.type === 'image' || el.type === 'logo'));
	const isDynamic = $derived(el && ['name', 'validity'].includes(el.type));

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
	class="w-full lg:w-80 bg-white border border-slate-100 p-5 rounded-xl shadow-xl space-y-5 shrink-0 text-left overflow-y-auto max-h-[80vh]"
>
	{#if !el}
		<div class="flex flex-col items-center justify-center py-12 text-center gap-3">
			<MousePointer2 class="w-10 h-10 text-slate-200" aria-hidden="true" />
			<span class="text-xs text-slate-400 font-medium">Element auf der Karte anklicken</span>
		</div>
	{:else}
		<div class="flex items-center justify-between">
			<h3 class="text-xs font-medium text-slate-600">{el.id}</h3>
			{#if !['header', 'address', 'logo', 'photo', 'name', 'validity', 'barcode'].includes(el.id)}
				<button
					onclick={handleDelete}
					class="text-slate-300 hover:text-rose-500 transition-colors p-1"
					title="Element löschen"
				>
					<Trash2 class="w-4 h-4" aria-hidden="true" />
				</button>
			{/if}
		</div>

		<!-- Visibility -->
		<div class="flex items-center justify-between">
			<span class="text-xs font-medium text-slate-500">Sichtbar</span>
			<label class="relative inline-flex items-center cursor-pointer select-none">
				<input type="checkbox" bind:checked={el.show} class="sr-only peer" />
				<div
					class="w-7 h-4 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-blue-600"
				></div>
			</label>
		</div>

		<!-- Position & Size -->
		<div class="space-y-2 pt-2 border-t border-slate-100">
			<span class="text-xs font-medium text-slate-500 block">Position &amp; Größe</span>
			<div class="grid grid-cols-2 gap-2">
				<ZahlenFeld
					label="X (mm)"
					value={el.x}
					min={0}
					max={80}
					step={0.5}
					onInput={(v) => (el.x = v)}
				/>
				<ZahlenFeld
					label="Y (mm)"
					value={el.y}
					min={0}
					max={50}
					step={0.5}
					onInput={(v) => (el.y = v)}
				/>
				<ZahlenFeld
					label="Breite (mm)"
					value={el.width}
					min={3}
					max={85}
					step={0.5}
					onInput={(v) => (el.width = v)}
				/>
				<ZahlenFeld
					label="Höhe (mm)"
					value={el.height}
					min={2}
					max={53}
					step={0.5}
					onInput={(v) => (el.height = v)}
				/>
			</div>
		</div>

		<!-- Z-Index -->
		<div class="flex items-center gap-2 pt-2 border-t border-slate-100">
			<span class="text-xs font-medium text-slate-500 flex-1">Ebene (z={el.zIndex})</span>
			<button
				onclick={() => bringForward(side, el.id)}
				class="px-2 py-1 text-label-small bg-slate-100 hover:bg-slate-200 rounded-lg font-bold transition-colors"
				title="Nach vorne">▲</button
			>
			<button
				onclick={() => sendBackward(side, el.id)}
				class="px-2 py-1 text-label-small bg-slate-100 hover:bg-slate-200 rounded-lg font-bold transition-colors"
				title="Nach hinten">▼</button
			>
		</div>

		{#if isTextType && el.style}
			<PropertiesText {el} istDynamisch={isDynamic} schriften={fontFamilies} />
		{/if}

		<!-- Image panel -->
		{#if isImageType}
			<div class="space-y-3 pt-2 border-t border-slate-100">
				<span class="text-xs font-medium text-slate-500 block">Bild</span>
				<input
					type="file"
					accept="image/*"
					onchange={handleImageUpload}
					class="w-full text-xs text-slate-500 file:mr-2 file:py-1 file:px-2 file:rounded-md file:border-0 file:text-label-small file:font-semibold file:bg-slate-100 file:text-slate-700 hover:file:bg-slate-200 cursor-pointer"
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
