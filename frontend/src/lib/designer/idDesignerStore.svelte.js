/**
 * @file idDesignerStore.svelte.js
 * Shared Svelte 5 module-level reactive store for the canvas-based ID card designer.
 *
 * Two-sided (duplex) layout for ISO 7810 ID-1 cards (85.60 mm × 53.98 mm).
 * Survives tab switches inside the SPA without re-mount.
 *
 * --- Element Schema ---
 * id          string   unique within the side
 * type        string   'text'|'name'|'details'|'validity'|'header'|'address'|
 *                      'image'|'logo'|'photo'|'barcode'
 * content     string   static text or base64 data-url (image)
 * x, y        number   position in mm from top-left
 * width       number   bounding box width in mm
 * height      number   bounding box height in mm
 * zIndex      number   stacking order (higher = on top)
 * show        boolean  visibility toggle
 * proportional boolean lock aspect ratio during resize (images)
 * style       object   (text-type elements only)
 *   fontFamily  string CSS font-family value
 *   fontSize    number pt size
 *   color       string CSS color
 *   textAlign   string 'left'|'center'|'right'
 *   fontWeight  string 'normal'|'bold'
 */

/** Monotone counter for generating unique element IDs at runtime. */
let _nextId = 100;
/** @returns {string} */
export function nextId() {
	return `el-${++_nextId}`;
}

/** Default text style shared by all text-type elements. */
function textStyle(fontSize = 7, color = '#1e293b', textAlign = 'left', fontWeight = 'normal') {
	return { fontFamily: 'inherit', fontSize, color, textAlign, fontWeight };
}

/** @returns {any[]} */
export function defaultFrontElements() {
	return [
		{
			id: 'header',
			type: 'header',
			content: 'STÄDTISCHES GYMNASIUM MUSTERSTADT',
			x: 5,
			y: 4,
			width: 75,
			height: 7,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7.5, '#1e293b', 'center', 'bold')
		},
		{
			id: 'address',
			type: 'address',
			content: 'Musterstraße 12, 12345 Musterstadt',
			x: 30,
			y: 8,
			width: 50,
			height: 7,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(6.5, '#475569', 'left', 'normal')
		},
		{
			id: 'logo',
			type: 'logo',
			content: '',
			x: 68,
			y: 12,
			width: 12,
			height: 12,
			zIndex: 2,
			show: true,
			proportional: true
		},
		{
			id: 'photo',
			type: 'photo',
			content: '',
			x: 5,
			y: 12,
			width: 22,
			height: 28,
			zIndex: 2,
			show: true,
			proportional: true
		},
		{
			id: 'name',
			type: 'name',
			content: '',
			x: 30,
			y: 14,
			width: 50,
			height: 10,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(9, '#0f172a', 'left', 'bold')
		},
		{
			id: 'details',
			type: 'details',
			content: '',
			x: 30,
			y: 21,
			width: 50,
			height: 8,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7.5, '#475569', 'left', 'normal')
		},
		{
			id: 'validity',
			type: 'validity',
			content: '',
			x: 30,
			y: 27,
			width: 50,
			height: 6,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7, '#475569', 'left', 'normal')
		},
		{
			id: 'barcode',
			type: 'barcode',
			content: '',
			x: 30,
			y: 34,
			width: 30,
			height: 14,
			zIndex: 1,
			show: true,
			proportional: true
		}
	];
}

/** @returns {any[]} */
export function defaultBackElements() {
	return [
		{
			id: 'back-header',
			type: 'text',
			content: 'STÄDTISCHE SCHULBIBLIOTHEK',
			x: 5,
			y: 8,
			width: 75,
			height: 7,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7.5, '#1e293b', 'left', 'bold')
		},
		{
			id: 'back-info',
			type: 'text',
			content: 'Bitte bei Verlust abgeben an\nHauptschule · Bibliothek',
			x: 5,
			y: 18,
			width: 60,
			height: 12,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(6.5, '#475569', 'left', 'normal')
		},
		{
			id: 'back-sponsor-label',
			type: 'text',
			content: 'Mit freundlicher Unterstützung von:',
			x: 5,
			y: 35,
			width: 75,
			height: 6,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(6, '#475569', 'left', 'normal')
		},
		{
			id: 'back-sponsor-logo',
			type: 'image',
			content: '',
			x: 32,
			y: 40,
			width: 20,
			height: 10,
			zIndex: 2,
			show: true,
			proportional: true
		}
	];
}

/**
 * Central store — all fields are deeply reactive via Svelte 5 $state.
 * Access as `idStore.front.elements[i].x` etc. from any component.
 */
export const idStore = $state({
	/** @type {"code39" | "qr"} */
	barcodeType: 'code39',
	/** @type {"card" | "a4"} */
	printMode: 'card',

	front: {
		elements: defaultFrontElements(),
		theme: 'bg-white text-black border-slate-200'
	},
	back: {
		elements: defaultBackElements(),
		// Muss exakt einem themes-Wert in Toolbar.svelte entsprechen, sonst zeigt das
		// Hintergrund-Dropdown keine Auswahl an.
		theme: 'bg-slate-100 text-slate-900 border-slate-300'
	}
});

// ---------------------------------------------------------------------------
// Z-index helpers
// ---------------------------------------------------------------------------

/**
 * @param {'front'|'back'} side
 * @param {string} id
 */
export function bringForward(side, id) {
	const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
	const el = els.find((e) => e.id === id);
	if (!el) return;
	const maxZ = Math.max(...els.map((e) => e.zIndex));
	if (el.zIndex < maxZ) el.zIndex++;
}

/**
 * @param {'front'|'back'} side
 * @param {string} id
 */
export function sendBackward(side, id) {
	const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
	const el = els.find((e) => e.id === id);
	if (!el) return;
	const minZ = Math.min(...els.map((e) => e.zIndex));
	if (el.zIndex > minZ) el.zIndex--;
}

// ---------------------------------------------------------------------------
// Element management (back-side custom elements)
// ---------------------------------------------------------------------------

/** Add a new free-text element to the back side. */
export function addTextElement() {
	idStore.back.elements = [
		...idStore.back.elements,
		{
			id: nextId(),
			type: 'text',
			content: 'Neuer Text',
			x: 10,
			y: 10,
			width: 40,
			height: 8,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7, '#1e293b', 'left', 'normal')
		}
	];
}

/**
 * Add one or more image elements to the given side.
 * @param {'front'|'back'} side
 * @param {string[]} dataUrls - base64 data-URLs of the uploaded images
 */
export function addImageElements(side, dataUrls) {
	const newEls = dataUrls.map((url, i) => ({
		id: nextId(),
		type: 'image',
		content: url,
		x: 10 + i * 5,
		y: 10 + i * 5,
		width: 20,
		height: 10,
		zIndex: 2,
		show: true,
		proportional: true
	}));
	if (side === 'front') {
		idStore.front.elements = [...idStore.front.elements, ...newEls];
	} else {
		idStore.back.elements = [...idStore.back.elements, ...newEls];
	}
}

/**
 * @param {'front'|'back'} side
 * @param {string} id
 */
export function removeElement(side, id) {
	if (side === 'front') {
		idStore.front.elements = idStore.front.elements.filter((e) => e.id !== id);
	} else {
		idStore.back.elements = idStore.back.elements.filter((e) => e.id !== id);
	}
}

// ---------------------------------------------------------------------------
// Zentrale Persistenz (Backend) — damit alle vernetzten Arbeitsplätze denselben
// Ausweis-Stand sehen. serialize/apply kapseln die JSON-Form des Designs.
// ---------------------------------------------------------------------------

/** Liefert einen plainen Snapshot des gesamten Designs (für PUT /api/ausweis-layout). */
export function serializeDesign() {
	return $state.snapshot(idStore);
}

/**
 * Übernimmt ein vom Server geladenes Design in den Store. Defensiv: fehlende oder
 * ungültige Felder bleiben auf ihren Defaults (leeres {} = Erststart → Defaults).
 * @param {any} data
 */
export function applyDesign(data) {
	if (!data || typeof data !== 'object') return;
	if (data.barcodeType === 'qr' || data.barcodeType === 'code39') {
		idStore.barcodeType = data.barcodeType;
	}
	if (data.printMode === 'card' || data.printMode === 'a4') {
		idStore.printMode = data.printMode;
	}
	applySeite(idStore.front, data.front);
	applySeite(idStore.back, data.back);
}

/**
 * @param {{elements: any[], theme: string}} ziel
 * @param {any} quelle
 */
function applySeite(ziel, quelle) {
	if (!quelle || typeof quelle !== 'object') return;
	if (Array.isArray(quelle.elements)) {
		ziel.elements = quelle.elements;
	}
	if (typeof quelle.theme === 'string' && quelle.theme.trim() !== '') {
		ziel.theme = quelle.theme;
	}
}
