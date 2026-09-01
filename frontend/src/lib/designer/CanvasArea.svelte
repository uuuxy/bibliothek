<script>
	/**
	 * @file CanvasArea.svelte
	 * Interactive canvas for the ID card designer. Die Darstellung der einzelnen
	 * Elemente (inkl. Auswahl-Ring und Skalier-Griffe) liegt in CanvasElement.svelte;
	 * hier wohnt die Drag-/Resize-Mechanik, weil Fensterlistener und ihr Aufräumen an
	 * EINE Stelle gehören.
	 *
	 * Drag & Resize:
	 *   Both operations bind pointermove/pointerup on `window` to allow the pointer
	 *   to travel outside the card boundary. `activeDragCleanup` stores the teardown
	 *   function and is called by onDestroy to prevent listener leaks on unmount.
	 */
	import { onDestroy } from 'svelte';
	import { idStore } from './idDesignerStore.svelte.js';
	import CanvasElement from './CanvasElement.svelte';

	/** @type {{ side: 'front'|'back', selectedId: string|null, onSelect: (id: string|null)=>void, student: any, zoom: number, barcodeType: string }} */
	const { side, selectedId, onSelect, student, zoom, barcodeType } = $props();

	/** Elements for the active side, sorted ascending by zIndex so higher layers render on top. */
	const elements = $derived(
		(side === 'front' ? idStore.front.elements : idStore.back.elements)
			.filter((el) => el.show)
			.slice()
			.sort((a, b) => a.zIndex - b.zIndex)
	);

	const theme = $derived(side === 'front' ? idStore.front.theme : idStore.back.theme);

	/** @type {(() => void) | null} */
	let activeDragCleanup = null;
	// Called on unmount — removes any stale window listeners from an in-progress drag.
	onDestroy(() => activeDragCleanup?.());

	/** @type {HTMLDivElement | undefined} */
	let cardEl = $state(undefined);

	/** Convert screen-space pixel delta to card-space millimetres. */
	function pxToMm() {
		if (!cardEl) return 1;
		return 85.6 / cardEl.getBoundingClientRect().width;
	}

	/**
	 * Begin drag-to-move for an element.
	 * @param {PointerEvent} e
	 * @param {string} elId
	 */
	function startDrag(e, elId) {
		if (e.button !== 0) return;
		e.preventDefault();
		e.stopPropagation();

		const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
		const el = els.find((x) => x.id === elId);
		if (!el) return;

		onSelect(elId);
		const scale = pxToMm();
		const startX = e.clientX,
			startY = e.clientY;
		const ix = el.x,
			iy = el.y;

		/** @param {PointerEvent} mv */
		function onMove(mv) {
			const dx = (mv.clientX - startX) * scale;
			const dy = (mv.clientY - startY) * scale;
			el.x = Math.max(0, Math.min(85.6 - el.width, Math.round((ix + dx) * 10) / 10));
			el.y = Math.max(0, Math.min(53.98 - el.height, Math.round((iy + dy) * 10) / 10));
		}
		function onUp() {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			activeDragCleanup = null;
		}
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
		activeDragCleanup = () => {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
	}

	/**
	 * Begin resize from a corner handle.
	 * @param {PointerEvent} e
	 * @param {string} elId
	 * @param {'nw'|'ne'|'sw'|'se'} corner
	 */
	function startResize(e, elId, corner) {
		if (e.button !== 0) return;
		e.preventDefault();
		e.stopPropagation();

		const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
		const el = els.find((x) => x.id === elId);
		if (!el) return;

		const scale = pxToMm();
		const startX = e.clientX,
			startY = e.clientY;
		const ix = el.x,
			iy = el.y,
			iw = el.width,
			ih = el.height;
		const aspectRatio = iw / ih;

		/** @param {PointerEvent} mv */
		function onMove(mv) {
			let dx = (mv.clientX - startX) * scale;
			let dy = (mv.clientY - startY) * scale;

			if (corner === 'se') {
				const newW = Math.max(5, iw + dx);
				el.width = newW;
				el.height = el.proportional ? newW / aspectRatio : Math.max(3, ih + dy);
			} else if (corner === 'sw') {
				const newW = Math.max(5, iw - dx);
				el.x = Math.max(0, ix + (iw - newW));
				el.width = newW;
				el.height = el.proportional ? newW / aspectRatio : Math.max(3, ih + dy);
			} else if (corner === 'ne') {
				const newW = Math.max(5, iw + dx);
				const newH = Math.max(3, ih - dy);
				el.y = Math.max(0, iy + (ih - newH));
				el.width = newW;
				el.height = el.proportional ? newW / aspectRatio : newH;
			} else {
				// nw
				const newW = Math.max(5, iw - dx);
				const newH = el.proportional ? newW / aspectRatio : Math.max(3, ih - dy);
				el.x = Math.max(0, ix + (iw - newW));
				el.y = Math.max(0, iy + (ih - newH));
				el.width = newW;
				el.height = newH;
			}
		}
		function onUp() {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			activeDragCleanup = null;
		}
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
		activeDragCleanup = () => {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
	}

	/** Deselect on clicking empty canvas background. */
	function onCanvasClick() {
		onSelect(null);
	}
</script>

<div
	class="flex-1 flex flex-col items-center justify-center overflow-hidden bg-slate-100 border border-dashed border-slate-200 rounded-3xl min-h-120 relative p-6"
	role="presentation"
	onpointerdown={onCanvasClick}
>
	<div
		style="transform: scale({zoom /
			100}); transform-origin: center; transition: transform 0.05s ease-out;"
		class="shrink-0"
	>
		<div
			bind:this={cardEl}
			class="card-container shadow-2xl relative border border-slate-200 rounded-lg overflow-visible select-none"
			style="width: 85.6mm; height: 53.98mm; background: white;"
		>
			<div class="w-full h-full relative rounded-lg overflow-hidden {theme}">
				{#each elements as el (el.id)}
					<CanvasElement
						{el}
						isSelected={selectedId === el.id}
						{student}
						{barcodeType}
						onStartDrag={startDrag}
						onStartResize={startResize}
					/>
				{/each}
			</div>
		</div>
	</div>

	<span
		class="absolute bottom-4 left-1/2 -translate-x-1/2 text-xs text-slate-400 font-medium pointer-events-none"
	>
		{side === 'front' ? 'Vorderseite' : 'Rückseite'} · Drag &amp; Drop zum Verschieben · Ecken zum Skalieren
	</span>
</div>
