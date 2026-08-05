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

// Platzhalter für Schulname/-adresse, solange niemand die echten Schul-Stammdaten
// (Einstellungen → Allgemein) hinterlegt hat oder der Ausweis-Designer sie noch nicht
// laden konnte. wendeSchulstammdatenAn() ersetzt GENAU diesen Text — nie einen davon
// abweichenden, von jemandem bereits selbst eingetragenen Text.
export const PLATZHALTER_SCHULNAME = 'STÄDTISCHES GYMNASIUM MUSTERSTADT';
export const PLATZHALTER_ADRESSE = 'Musterstraße 12, 12345 Musterstadt';

/** @returns {any[]} */
export function defaultFrontElements() {
	return [
		{
			id: 'header',
			type: 'header',
			content: PLATZHALTER_SCHULNAME,
			x: 5,
			y: 4,
			width: 75,
			height: 7,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(8, '#1e293b', 'center', 'bold')
		},
		{
			id: 'address',
			type: 'address',
			content: PLATZHALTER_ADRESSE,
			x: 30,
			y: 8,
			width: 50,
			height: 7,
			zIndex: 1,
			show: true,
			proportional: false,
			style: textStyle(7, '#475569', 'left', 'normal')
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
			style: textStyle(10, '#0f172a', 'left', 'bold')
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
			style: textStyle(8, '#475569', 'left', 'normal')
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
			style: textStyle(7.5, '#475569', 'left', 'normal')
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
			style: textStyle(8, '#1e293b', 'left', 'bold')
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
			style: textStyle(7.5, '#475569', 'left', 'normal')
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
			style: textStyle(6.5, '#475569', 'left', 'normal')
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

// Muss exakt einem themes-Wert in Toolbar.svelte entsprechen, sonst zeigt das
// Hintergrund-Dropdown keine Auswahl an. Als Konstante, damit Store-Initialisierung
// und resetDesign() nicht auseinanderlaufen können.
export const FRONT_THEME_DEFAULT = 'bg-white text-black border-slate-200';
export const BACK_THEME_DEFAULT = 'bg-slate-100 text-slate-900 border-slate-300';

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
		theme: FRONT_THEME_DEFAULT
	},
	back: {
		elements: defaultBackElements(),
		theme: BACK_THEME_DEFAULT
	}
});

/**
 * Setzt beide Seiten auf die Standardwerte zurück.
 *
 * Nötig, weil die Defaults oben ausschließlich beim Erststart greifen: Sobald einmal ein
 * Design zentral gespeichert wurde, überschreibt applyDesign() sie vollständig. Ohne
 * diesen Weg käme eine Änderung an den Standardwerten (etwa an den Schriftgrößen) auf
 * einer laufenden Installation nie an.
 *
 * Der Aufrufer muss die Elementauswahl zurücksetzen — die bisherigen IDs existieren
 * danach nicht mehr.
 */
export function resetDesign() {
	idStore.front.elements = defaultFrontElements();
	idStore.front.theme = FRONT_THEME_DEFAULT;
	idStore.back.elements = defaultBackElements();
	idStore.back.theme = BACK_THEME_DEFAULT;
}

/**
 * Ersetzt den Platzhalter-Schulnamen/-adresse auf der Vorderseite durch die echten
 * Schul-Stammdaten (Einstellungen → Allgemein), sobald sie geladen werden konnten.
 *
 * Läuft nach JEDEM Laden — nicht nur beim allerersten Start: Ein bereits zentral
 * gespeichertes Design (z. B. weil beim Go-Live niemand den Kopf angepasst hat, bevor
 * das Design das erste Mal gespeichert wurde) trägt den Platzhalter sonst für immer
 * weiter, weil applyDesign() ihn aus der DB lädt statt aus defaultFrontElements().
 *
 * Rührt NIE einen Wert an, der vom Platzhalter abweicht — wer den Kopf bereits selbst
 * beschriftet hat (auch mit demselben Wortlaut wie eine andere Schule), wird nicht
 * überschrieben. Leere Schul-Stammdaten (Einstellungen selbst noch nicht ausgefüllt)
 * lassen den Platzhalter bewusst stehen statt ihn durch eine leere Zeile zu ersetzen.
 *
 * @param {string} schuleName
 * @param {string} adresse
 */
export function wendeSchulstammdatenAn(schuleName, adresse) {
	const header = idStore.front.elements.find((e) => e.id === 'header');
	if (header && schuleName && header.content === PLATZHALTER_SCHULNAME) {
		header.content = schuleName;
	}
	const address = idStore.front.elements.find((e) => e.id === 'address');
	if (address && adresse && address.content === PLATZHALTER_ADRESSE) {
		address.content = adresse;
	}
}

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
// Element management (frei hinzugefügte Elemente, beide Seiten)
// ---------------------------------------------------------------------------

/**
 * Fügt ein freies Textelement auf der angegebenen Seite hinzu.
 *
 * Die Seite ist ein Parameter (wie bei addImageElements): Die Funktion schrieb früher
 * fest auf die Rückseite, weshalb die Toolbar den „+ Text"-Button auf der Vorderseite
 * ausblenden musste. Gerendert wurden Text-Elemente aber schon immer auf beiden Seiten
 * (CardFace.svelte führt 'text' in der isText-Liste) — es fehlte nur der Weg, auf der
 * Vorderseite eines anzulegen.
 *
 * @param {'front'|'back'} side
 */
export function addTextElement(side) {
	const neu = {
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
		style: textStyle(7.5, '#1e293b', 'left', 'normal')
	};
	if (side === 'front') {
		idStore.front.elements = [...idStore.front.elements, neu];
	} else {
		idStore.back.elements = [...idStore.back.elements, neu];
	}
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
