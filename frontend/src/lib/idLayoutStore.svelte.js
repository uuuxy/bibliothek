/**
 * Shared Svelte 5 module-level state for the Student-ID card layout.
 *
 * Two-sided (duplex) layout for Datacard CR80 / ISO 7810 ID-1 card printers.
 * Survives tab switches inside the SPA without re-mount.
 *
 * Each side has:
 *   elements  – ordered array of card elements
 *   theme     – Tailwind background/text/border classes for the card surface
 *
 * Element fields:
 *   id        string   unique within the side
 *   type      string   'text'|'image'|'photo'|'barcode'|'name'|'details'|'validity'
 *   content   string   static label text (type:'text') or base64 data-url (type:'image')
 *   x, y      number   position in mm from top-left corner
 *   scale     number   size multiplier (also drives photo/barcode intrinsic size)
 *   show      boolean  visibility toggle
 *   zIndex    number   stacking order
 *   fontSize  number   (text types) pt size before scale
 *   bold      boolean  (text types) bold font weight
 *   width     number   (image type) intrinsic width in mm at scale 1
 *   height    number   (image type) intrinsic height in mm at scale 1
 */

let _nextId = 100;
/** @returns {string} */
export function nextId() {
	return `el-${++_nextId}`;
}

/** @returns {any[]} */
export function defaultFrontElements() {
	return [
		{
			id: 'header',
			type: 'text',
			content: 'STÄDTISCHES GYMNASIUM MUSTERSTADT',
			x: 5,
			y: 4,
			scale: 1.0,
			show: true,
			zIndex: 1,
			bold: true,
			fontSize: 7.5
		},
		{
			id: 'address',
			type: 'text',
			content: 'Musterstraße 12, 12345 Musterstadt',
			x: 30,
			y: 8,
			scale: 0.8,
			show: true,
			zIndex: 1,
			bold: false,
			fontSize: 6.5
		},
		{
			id: 'logo',
			type: 'image',
			content: '',
			x: 68,
			y: 12,
			scale: 1.0,
			show: true,
			zIndex: 2,
			width: 12,
			height: 12
		},
		{ id: 'photo', type: 'photo', content: '', x: 5, y: 12, scale: 1.0, show: true, zIndex: 2 },
		{
			id: 'name',
			type: 'name',
			content: '',
			x: 30,
			y: 14,
			scale: 1.1,
			show: true,
			zIndex: 1,
			bold: true,
			fontSize: 9
		},
		{
			id: 'details',
			type: 'details',
			content: '',
			x: 30,
			y: 21,
			scale: 0.9,
			show: true,
			zIndex: 1,
			bold: false,
			fontSize: 7.5
		},
		{
			id: 'validity',
			type: 'validity',
			content: '',
			x: 30,
			y: 27,
			scale: 0.85,
			show: true,
			zIndex: 1,
			bold: false,
			fontSize: 7
		},
		{ id: 'barcode', type: 'barcode', content: '', x: 30, y: 34, scale: 0.8, show: true, zIndex: 1 }
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
			scale: 1.0,
			show: true,
			zIndex: 1,
			bold: true,
			fontSize: 7.5
		},
		{
			id: 'back-info',
			type: 'text',
			content: 'Bitte bei Verlust abgeben an\nHauptschule · Bibliothek',
			x: 5,
			y: 18,
			scale: 1.0,
			show: true,
			zIndex: 1,
			bold: false,
			fontSize: 6.5
		},
		{
			id: 'back-sponsor-label',
			type: 'text',
			content: 'Mit freundlicher Unterstützung von:',
			x: 5,
			y: 35,
			scale: 1.0,
			show: true,
			zIndex: 1,
			bold: false,
			fontSize: 6
		},
		{
			id: 'back-sponsor-logo',
			type: 'image',
			content: '',
			x: 32,
			y: 40,
			scale: 1.0,
			show: true,
			zIndex: 2,
			width: 20,
			height: 10
		}
	];
}

export const idStore = $state({
	/** @type {"code39" | "qr"} */
	barcodeType: 'code39',

	/** @type {"card" | "a4"} Scheckkarte (ID-1) is the mandatory default */
	printMode: /** @type {"card"} */ ('card'),

	cardTheme: 'bg-white text-black border-slate-200',

	layout: {
		header: { x: 5, y: 4, scale: 1.0, show: true, text: 'STÄDTISCHES GYMNASIUM MUSTERSTADT' },
		address: { x: 30, y: 8, scale: 0.8, show: true, text: 'Musterstraße 12, 12345 Musterstadt' },
		logo: { x: 68, y: 12, scale: 1.0, show: true, url: '' },
		photo: { x: 5, y: 12, scale: 1.0, show: true },
		name: { x: 30, y: 14, scale: 1.1, show: true },
		details: { x: 30, y: 21, scale: 0.9, show: true },
		validity: { x: 30, y: 27, scale: 0.85, show: true },
		barcode: { x: 30, y: 34, scale: 0.8, show: true }
	},

	front: {
		elements: defaultFrontElements(),
		theme: 'bg-white text-black border-slate-200'
	},

	back: {
		elements: defaultBackElements(),
		theme: 'bg-slate-50 text-slate-900 border-slate-200'
	}
});
