<script>
	/**
	 * @file CanvasFlaeche.svelte
	 * Interactive canvas for the ID card designer. Die Darstellung der einzelnen
	 * Elemente (inkl. Auswahl-Ring und Skalier-Griffe) liegt in CanvasElement.svelte;
	 * hier wohnt die Drag-/Resize-Mechanik, weil Fensterlistener und ihr Aufräumen an
	 * EINE Stelle gehören.
	 */
	import { onDestroy } from 'svelte';
	import { idStore } from './idDesignerStore.svelte.js';
	import CanvasElement from './CanvasElement.svelte';
	import { kartenStil } from './kartenFarben.js';
	import {
		KARTE_BREITE_MM,
		KARTE_HOEHE_MM,
		rasteEin,
		sichererAbstand,
		stehtImRand
	} from './kartenRand.js';
	import { hinweisZumRand, linienAbstand, linienStil } from './randLinie.js';
	import { skaliere } from './skalieren.js';

	/** @type {{ side: 'front'|'back', gewaehlteKennung: string|null, onSelect: (id: string|null)=>void, leser: any, zoom: number, barcodeType: string, gesperrt?: boolean }} */
	const { side, gewaehlteKennung, onSelect, leser, zoom, barcodeType, gesperrt = false } = $props();

	/** Elements for the active side, sorted ascending by zIndex so higher layers render on top. */
	const elements = $derived(
		(side === 'front' ? idStore.front.elements : idStore.back.elements)
			.filter((el) => el.show)
			.slice()
			.sort((a, b) => a.zIndex - b.zIndex)
	);

	const theme = $derived(side === 'front' ? idStore.front.theme : idStore.back.theme);

	// Das Element, das gerade gezogen wird, und ob eine seiner Kanten an der Linie eingerastet ist.
	/** @type {any} */
	let gezogen = $state(null);
	let eingerastet = $state(false);
	const zuNah = $derived(elements.filter(stehtImRand));
	const aktiv = $derived(eingerastet || (!!gezogen && stehtImRand(gezogen)));

	/** @type {(() => void) | null} */
	let ziehenAufraeumen = null;
	// Called on unmount — removes any stale window listeners from an in-progress drag.
	onDestroy(() => ziehenAufraeumen?.());

	/** @type {HTMLDivElement | undefined} */
	let karteEl = $state(undefined);

	/** Convert screen-space pixel delta to card-space millimetres. */
	function pxToMm() {
		if (!karteEl) return 1;
		return KARTE_BREITE_MM / karteEl.getBoundingClientRect().width;
	}

	/**
	 * Hängt die Listener einer Bewegung an window, damit der Zeiger die Karte verlassen darf.
	 * Loslassen nimmt sie ab, ebenso onDestroy mitten in der Bewegung (ziehenAufraeumen).
	 * @param {(mv: PointerEvent) => void} onMove
	 */
	function verfolge(onMove) {
		function onUp() {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			ziehenAufraeumen = null;
			gezogen = null;
			eingerastet = false;
		}
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
		ziehenAufraeumen = onUp;
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
			const lage = rasteEin({ ...el, x: ix + dx, y: iy + dy });
			el.x = Math.max(0, Math.min(KARTE_BREITE_MM - el.width, lage.x));
			el.y = Math.max(0, Math.min(KARTE_HOEHE_MM - el.height, lage.y));
			eingerastet = lage.eingerastet;
		}
		gezogen = el;
		verfolge(onMove);
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
		const start = { x: el.x, y: el.y, width: el.width, height: el.height };
		const proportional = !!el.proportional;

		verfolge((mv) => {
			const dx = (mv.clientX - startX) * scale;
			const dy = (mv.clientY - startY) * scale;
			Object.assign(el, skaliere(corner, { ...start, proportional }, dx, dy));
		});
	}
</script>

<div
	class="flex-1 flex flex-col items-center justify-center overflow-hidden bg-surface-container border border-dashed border-outline-variant rounded-3xl min-h-120 relative p-6"
	role="presentation"
	inert={gesperrt}
	onpointerdown={() => onSelect(null)}
>
	<div
		style="transform: scale({zoom /
			100}); transform-origin: center; transition: transform 0.05s ease-out;"
		class="shrink-0"
	>
		<!-- data-ausweis-vorschau: Ab hier ist alles die maßstäbliche Zeichnung der
		     Plastikkarte, kein Bedienelement — das Typografie-Gate misst darin nicht
		     (e2e/typo-rollen.spec.js). Die Schriftgrößen hier sind die des DRUCKS. -->
		<div
			bind:this={karteEl}
			data-ausweis-vorschau
			class="schema-hell card-container shadow-2xl relative border border-outline-variant rounded-lg overflow-visible select-none"
			style="width: 85.6mm; height: 53.98mm; background: white;"
		>
			<div class="w-full h-full relative rounded-lg overflow-hidden" style={kartenStil(theme)}>
				{#each elements as el (el.id)}
					<CanvasElement
						{el}
						gewaehlt={gewaehlteKennung === el.id}
						{leser}
						{barcodeType}
						onStartDrag={startDrag}
						onStartResize={startResize}
					/>
				{/each}
				<!-- Bis hierher druckt der Kartendrucker sicher (kartenRand.js); die Linie wird nicht
				     gedruckt. Beim Ziehen zeigt sie die Grenze des gezogenen Elements, beim Strichcode die seiner Striche. -->
				<div
					data-sicherheitsrand
					aria-hidden="true"
					class="pointer-events-none absolute z-50 border-dashed {linienStil({
						zuNah: zuNah.length > 0,
						eingerastet,
						aktiv
					})}"
					style="inset: {linienAbstand(gezogen ? sichererAbstand(gezogen) : null)};"
				></div>
			</div>
		</div>
	</div>

	<span
		class="absolute bottom-4 left-1/2 -translate-x-1/2 text-sm text-on-surface-variant font-medium pointer-events-none"
	>
		{side === 'front' ? 'Vorderseite' : 'Rückseite'}{gesperrt
			? ''
			: ' · Ziehen zum Verschieben · Ecken zum Skalieren'} · {hinweisZumRand(zuNah)}
	</span>
</div>
