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
	import Kaestchen from '../components/ui/Kaestchen.svelte';
	import Switch from '../components/ui/Switch.svelte';
	import { MousePointer2, Trash } from '@lucide/svelte';

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
		el && ['header', 'address', 'name', 'validity', 'dokumenttyp', 'text'].includes(el.type)
	);
	const isImageType = $derived(el && (el.type === 'image' || el.type === 'logo'));
	const isBoxType = $derived(el && el.type === 'box');
	// Dynamisch heisst: Der Inhalt kommt aus den Leserdaten, nicht aus dem Eingabefeld.
	// Beim Dokumenttyp ist es die Art — „Schülerausweis" oder „Lehrerausweis".
	const isDynamic = $derived(el && ['name', 'validity', 'dokumenttyp'].includes(el.type));

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
	class="w-full lg:w-80 bg-surface-container-lowest border border-outline-variant p-5 rounded-xl space-y-5 shrink-0 text-left overflow-y-auto max-h-[80vh]"
>
	{#if !el}
		<div class="flex flex-col items-center justify-center py-12 text-center gap-3">
			<MousePointer2 class="w-10 h-10 text-outline-variant" aria-hidden="true" />
			<span class="text-xs text-on-surface-variant font-medium"
				>Element auf der Karte anklicken</span
			>
		</div>
	{:else}
		<div class="flex items-center justify-between">
			<h3 class="text-base font-medium text-on-surface">{el.id}</h3>
			{#if !['header', 'address', 'logo', 'photo', 'name', 'validity', 'barcode'].includes(el.id)}
				<button
					onclick={handleDelete}
					class="icon-btn text-on-surface-variant"
					data-tip="Element löschen"
					aria-label="Element löschen"
				>
					<Trash class="w-4 h-4" aria-hidden="true" />
				</button>
			{/if}
		</div>

		<!-- Visibility -->
		<div class="flex items-center justify-between">
			<label for="designer-element-sichtbar" class="text-xs font-medium text-on-surface-variant"
				>Sichtbar</label
			>
			<Switch id="designer-element-sichtbar" bind:checked={el.show} />
		</div>

		<!-- Position & Size -->
		<div class="space-y-2 pt-2 border-t border-outline-variant">
			<span class="text-xs font-medium text-on-surface-variant block">Position &amp; Größe</span>
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
		<div class="flex items-center gap-2 pt-2 border-t border-outline-variant">
			<span class="text-xs font-medium text-on-surface-variant flex-1">Ebene (z={el.zIndex})</span>
			<button
				onclick={() => bringForward(side, el.id)}
				class="px-2 py-1 text-label-small bg-surface-container text-on-surface rounded-lg font-bold"
				title="Nach vorne"
				aria-label="Ebene nach vorne verschieben">▲</button
			>
			<button
				onclick={() => sendBackward(side, el.id)}
				class="px-2 py-1 text-label-small bg-surface-container text-on-surface rounded-lg font-bold"
				title="Nach hinten"
				aria-label="Ebene nach hinten verschieben">▼</button
			>
		</div>

		{#if isTextType && el.style}
			<PropertiesText {el} istDynamisch={isDynamic} schriften={fontFamilies} />
		{/if}

		<!-- Farbfläche: Füllfarbe und Eckenradius. Der Farbwähler trägt Label und Höhe von
		     Feld.svelte, damit er neben ZahlenFeld fluchtet. -->
		{#if isBoxType && el.style}
			<div class="space-y-3 pt-2 border-t border-outline-variant">
				<span class="text-xs font-medium text-on-surface-variant block">Fläche</span>
				<div class="grid grid-cols-2 gap-2">
					<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
						<span class="text-sm font-medium text-on-surface-variant">Farbe</span>
						<input
							type="color"
							bind:value={el.style.color}
							class="w-full h-9 rounded-xl border border-outline-variant cursor-pointer bg-surface-container-lowest px-1"
						/>
					</div>
					<ZahlenFeld
						label="Ecken (mm)"
						value={el.style.radius ?? 0}
						min={0}
						max={10}
						step={0.5}
						onInput={(v) => (el.style.radius = v)}
					/>
				</div>
			</div>
		{/if}

		<!-- Image panel -->
		{#if isImageType}
			<div class="space-y-3 pt-2 border-t border-outline-variant">
				<span class="text-xs font-medium text-on-surface-variant block">Bild</span>
				<input
					type="file"
					accept="image/*"
					onchange={handleImageUpload}
					class="w-full text-xs text-on-surface-variant file:mr-2 file:py-1 file:px-2 file:rounded-md file:border-0 file:text-label-small file:font-semibold file:bg-surface-container file:text-on-surface hover:file:bg-surface-container-highest cursor-pointer"
				/>
				<Kaestchen bind:checked={el.proportional} label="Proportionale Skalierung" />
			</div>
		{/if}
	{/if}
</div>
