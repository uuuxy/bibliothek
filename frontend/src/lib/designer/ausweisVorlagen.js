import {
	idStore,
	PLATZHALTER_SCHULNAME,
	PLATZHALTER_ADRESSE,
	FRONT_THEME_DEFAULT
} from './idDesignerStore.svelte.js';
import { welleSvg } from './welleMotiv.js';

/**
 * @file ausweisVorlagen.js
 * Fertige Design-Vorlagen für den Ausweis-Designer: Ein Klick füllt beide Seiten mit
 * einem abgestimmten Layout, danach ist alles wie gewohnt frei editierbar.
 *
 * Flächen (Kopfband, Fußband, Seitenband, Streifen) sind 'box'-Elemente: ein Rechteck
 * mit Füllfarbe und optionalen runden Ecken, das Kartenseite/CanvasElement als schlichtes
 * div rendern. Als eingebettete SVG-Bilder wäre ihre Farbe nicht editierbar, und
 * object-contain ließe am Kartenrand weiße Streifen stehen. Einzig das Wellen-Motiv
 * (Reis-Welle) bleibt ein SVG-Bild: 40 Balken sind kein Rechteck (welleMotiv.js).
 *
 * Regeln, die alle Vorlagen einhalten:
 *   - Schulname/Adresse tragen die IDs 'header'/'address' mit den Platzhalter-Texten,
 *     damit wendeSchulstammdatenAn() sie wie beim Standard-Design ersetzt.
 *   - Das Logofeld bleibt leer (die Schule lädt ihre echte Logodatei hoch); leere
 *     Logofelder lässt der Druck einfach weg.
 *   - Keine Klassen-/Schuljahreszeile, der Ausweis gilt die ganze Schulzeit
 *     (Begründung am details-Filter in idDesignerStore.applySeite).
 *   - Der Barcode liegt immer auf heller Fläche: Kartenseite rendert die Nummer dunkel,
 *     auf dunklem Grund wäre sie unlesbar.
 *   - Text, Bilder und Barcode halten den sicheren Rand (kartenRand.js); nur Farbflächen
 *     und das Wellen-Motiv reichen bis an den Rand. Was außen steht, steht bündig an der
 *     Linie, der Barcode unten 0,3 mm darüber: Seine Striche brauchen 6,3 mm Abstand.
 */

/** ISO 7810 ID-1 — dieselben Maße wie Leinwand und Kartendruck. */
const KARTE_BREITE = 85.6;
const KARTE_HOEHE = 53.98;

/** Grün der Schul-Homepage; Kopfleiste dort ist schwarz. */
const GRUEN = '#76b82a';
const GRUEN_DUNKEL = '#4a7a16';
const SCHWARZ = '#101410';

/** Marine (Seitenband): Tiefblau mit Goldlinie. */
const MARINE = '#1d3557';
const GOLD = '#e9c46a';

/** Indigo (Kopfkarte): abgerundete Kopfkarte im M3-Stil. */
const INDIGO = '#3730a3';
const INDIGO_HELL = '#c7d2fe';
const INDIGO_AKZENT = '#4338ca';

/** Sommer-Streifen: warmes Streifen-Trio. */
const AMBER = '#f59e0b';
const ORANGE = '#ea580c';
const ROT = '#dc2626';

/** Kennung des dunklen Verlaufs in kartenFarben.js. */
export const WALDGRUEN_THEME = 'waldgruen';
const WEISS_THEME = FRONT_THEME_DEFAULT;

/** Für das Auswahl-Dropdown in der Werkzeugleiste. */
export const AUSWEIS_VORLAGEN = [
	{ value: 'blanko', label: 'Blanko (weiß)' },
	{ value: 'schwarz-gruen', label: 'Schwarz-Grün (Webauftritt)' },
	{ value: 'waldgruen', label: 'Waldgrün-Verlauf' },
	{ value: 'reis-welle', label: 'Reis-Welle' },
	{ value: 'marine', label: 'Marine (Seitenband)' },
	{ value: 'indigo-kopfkarte', label: 'Indigo (Kopfkarte)' },
	{ value: 'sonnenstreifen', label: 'Sommer-Streifen' }
];

const BESTIMMUNGEN =
	'Der Ausweis ist Eigentum der Schule und nicht übertragbar.\n' +
	'Verlust bitte sofort im Sekretariat oder in der Bibliothek melden.\n' +
	'Nur gültig mit Lichtbild und Unterschrift.';
const FUNDHINWEIS = 'Fundhinweis: Bitte im Sekretariat abgeben.';

/**
 * @param {number} fontSize pt
 * @param {string} color
 * @param {'normal'|'bold'} [fontWeight]
 * @param {'left'|'center'|'right'} [textAlign]
 */
function stil(fontSize, color, fontWeight = 'normal', textAlign = 'left') {
	return { fontFamily: 'inherit', fontSize, color, textAlign, fontWeight };
}

/** Lage und Größe eines Elements in mm: x, y, Breite, Höhe.
 *  @typedef {[x: number, y: number, width: number, height: number]} Rahmen */

/**
 * @param {string} type
 * @param {string} id
 * @param {string} content
 * @param {Rahmen} rahmen
 * @param {number} zIndex
 * @param {ReturnType<typeof stil>} [style]
 */
function el(type, id, content, [x, y, width, height], zIndex, style) {
	const proportional = type === 'photo' || type === 'logo' || type === 'barcode';
	return { id, type, content, x, y, width, height, zIndex, show: true, proportional, style };
}

/**
 * Farbfläche — Füllfarbe und Eckenradius sind im Eigenschaften-Panel editierbar.
 * @param {string} id
 * @param {string} farbe
 * @param {Rahmen} rahmen
 * @param {number} [zIndex]
 * @param {number} [radius] mm
 */
function box(id, farbe, [x, y, width, height], zIndex = 0, radius = 0) {
	return {
		id,
		type: 'box',
		content: '',
		x,
		y,
		width,
		height,
		zIndex,
		show: true,
		proportional: false,
		style: { color: farbe, radius }
	};
}

/**
 * Die drei festen Blöcke, die jede Vorlagen-Vorderseite gleich aufbaut.
 * @param {{akzent: string, text: string, nebentext: string}} farben
 * @param {number} [x]
 * @param {number} [yTitel]
 */
function personenBlock(farben, x = 30, yTitel = 16.5) {
	const breite = Math.round((82.6 - x) * 10) / 10; // bis an die Linie, auf 0,1 mm
	return [
		// Dokumenttyp statt fester Text: Die Aufschrift kommt aus der Art des Lesers
		// (leserArt.js). Die Großschreibung der Vorlagen ist damit eine Stilangabe.
		el('dokumenttyp', 'title', '', [x, yTitel, breite, 4], 1, {
			...stil(6, farben.akzent, 'bold'),
			textTransform: 'uppercase'
		}),
		el('name', 'name', '', [x, yTitel + 4.5, breite, 8], 1, stil(10, farben.text, 'bold')),
		el('validity', 'validity', '', [x, yTitel + 13, breite, 5], 1, stil(7.5, farben.nebentext))
	];
}

/**
 * Schulname + Adresse übereinander — der Standard-Kopf jeder Vorderseite. Die Adresse
 * gehört UNTER den Schulnamen (nicht ans Kartenende): So stand es im ursprünglichen
 * Entwurf, und so liest sich ein Briefkopf.
 * @param {number} x
 * @param {number} y
 * @param {number} breite
 * @param {{name: string, adresse: string}} farben
 * @param {number} [zeilen] Zeilen, die der Schulname belegen darf (schmale Spalte: 2)
 */
function schulkopf(x, y, breite, farben, zeilen = 1) {
	const groesse = zeilen > 1 ? 7 : 7.5;
	const hoehe = zeilen > 1 ? 7.5 : 5.5;
	return [
		el(
			'header',
			'header',
			PLATZHALTER_SCHULNAME,
			[x, y, breite, hoehe],
			1,
			stil(groesse, farben.name, 'bold')
		),
		el(
			'address',
			'address',
			PLATZHALTER_ADRESSE,
			[x, y + hoehe + 0.1, breite, 3.5],
			1,
			stil(5, farben.adresse)
		)
	];
}

/**
 * Unterschriftszeile für die Rückseiten (Linie als Farbfläche + Beschriftung).
 * @param {number} x
 * @param {number} y
 */
function unterschrift(x, y) {
	return [
		box('back-signatur-linie', '#3a4234', [x, y + 0.5, 32, 0.5], 1),
		el(
			'text',
			'back-signatur',
			'Unterschrift Schüler/in',
			[x, y + 1.7, 32, 3.5],
			1,
			stil(5.5, '#64748b')
		)
	];
}

/**
 * Standard-Rückseite: Bestimmungen, Unterschriftszeile, Fundhinweis — plus die
 * Deko-Flächen der jeweiligen Vorlage obendrüber.
 * @param {string} theme
 * @param {any[]} deko
 * @param {number} [yStart]
 */
function standardRueckseite(theme, deko = [], yStart = 11) {
	return {
		theme,
		elements: [
			...deko,
			el('text', 'back-header', 'Bestimmungen', [3, yStart, 79.6, 5], 1, stil(8, SCHWARZ, 'bold')),
			el('text', 'back-info', BESTIMMUNGEN, [3, yStart + 6.5, 79.6, 16], 1, stil(7, '#3a4234')),
			...unterschrift(3, 40),
			// Eine Zeile hoch: Text steht oben im Rahmen, und nur so endet er an der Linie.
			el('text', 'back-fund', FUNDHINWEIS, [3, 48.28, 79.6, 2.7], 1, stil(6, '#64748b'))
		]
	};
}

/** @returns {{front: {theme: string, elements: any[]}, back: {theme: string, elements: any[]}} | null} */
function blanko() {
	const farben = { akzent: '#475569', text: '#0f172a', nebentext: '#475569' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				...schulkopf(3, 3, 60, { name: '#111827', adresse: '#64748b' }),
				el('logo', 'logo', '', [72.6, 3, 10, 10], 2),
				el('photo', 'photo', '', [3, 16.5, 21, 25], 2),
				...personenBlock(farben, 27, 17),
				el('barcode', 'barcode', '', [27, 39.68, 30, 11], 1)
			]
		},
		back: standardRueckseite(WEISS_THEME, [], 3)
	};
}

/** @returns {ReturnType<typeof blanko>} */
function schwarzGruen() {
	const farben = { akzent: GRUEN_DUNKEL, text: SCHWARZ, nebentext: '#475569' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				box('kopfband', SCHWARZ, [0, 0, KARTE_BREITE, 13.4]),
				box('kopflinie', GRUEN, [0, 13.4, KARTE_BREITE, 1.1]),
				el('logo', 'logo', '', [3, 3, 9.5, 9.5], 2),
				...schulkopf(15.5, 3, 67.1, { name: '#ffffff', adresse: '#c9d6c2' }),
				el('photo', 'photo', '', [3, 17.5, 22, 27], 2),
				...personenBlock(farben, 29, 18),
				el('barcode', 'barcode', '', [29, 39.68, 30, 11], 1),
				box('fusslinie', GRUEN, [0, 52.9, KARTE_BREITE, 1.08])
			]
		},
		back: standardRueckseite(WEISS_THEME, [
			box('back-kopfband', SCHWARZ, [0, 0, KARTE_BREITE, 7.2]),
			box('back-kopflinie', GRUEN, [0, 7.2, KARTE_BREITE, 0.8])
		])
	};
}

/** @returns {ReturnType<typeof blanko>} */
function waldgruen() {
	const farben = { akzent: '#b9e07f', text: '#ffffff', nebentext: '#d8ecc0' };
	return {
		front: {
			theme: WALDGRUEN_THEME,
			elements: [
				box('fussband', '#f2f7e9', [0, 38.5, KARTE_BREITE, KARTE_HOEHE - 38.5]),
				el('logo', 'logo', '', [3, 3, 9.5, 9.5], 2),
				...schulkopf(15, 4.2, 67.6, { name: '#ffffff', adresse: '#d8ecc0' }),
				el('photo', 'photo', '', [3, 14.5, 21, 24], 2),
				...personenBlock(farben, 28, 14.5),
				el('barcode', 'barcode', '', [28, 39.68, 30, 11], 1)
			]
		},
		back: {
			theme: WALDGRUEN_THEME,
			elements: [
				box('back-panel', '#ffffff', [3, 9, 79.6, 31], 0, 3),
				el('text', 'back-header', 'Bestimmungen', [7, 12.5, 75.6, 5], 1, stil(8, SCHWARZ, 'bold')),
				el('text', 'back-info', BESTIMMUNGEN, [7, 18.5, 75.6, 15], 1, stil(7, '#3a4234')),
				...unterschrift(7, 33.5),
				el('text', 'back-fund', FUNDHINWEIS, [3, 48.28, 79.6, 2.7], 1, stil(6, '#d8ecc0'))
			]
		}
	};
}

/** @returns {ReturnType<typeof blanko>} */
function reisWelle() {
	const farben = { akzent: GRUEN_DUNKEL, text: SCHWARZ, nebentext: '#475569' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				...schulkopf(3, 3, 60, { name: SCHWARZ, adresse: '#64748b' }),
				el('logo', 'logo', '', [72.6, 3, 10, 10], 2),
				el('image', 'welle', welleSvg(GRUEN, 1, 70), [0, 13, KARTE_BREITE, 7], 0),
				el(
					'text',
					'welle-text',
					'Philipp Reis baute in Friedrichsdorf das erste Telefon (1861).',
					[29, 20.1, 53.6, 2.4],
					1,
					stil(4.5, '#7a8474', 'normal', 'right')
				),
				el('photo', 'photo', '', [3, 22.6, 20, 24], 2),
				...personenBlock(farben, 27, 22.6),
				el('barcode', 'barcode', '', [27, 39.68, 30, 11], 1)
			]
		},
		back: {
			theme: WEISS_THEME,
			elements: [
				el('text', 'back-header', 'Bestimmungen', [3, 3, 79.6, 5], 1, stil(8, SCHWARZ, 'bold')),
				el('text', 'back-info', BESTIMMUNGEN, [3, 9, 79.6, 15], 1, stil(7, '#3a4234')),
				el('image', 'back-welle', welleSvg(GRUEN, 0.25, 90), [0, 26, KARTE_BREITE, 12], 0),
				...unterschrift(50.6, 43),
				el('text', 'back-fund', FUNDHINWEIS, [3, 48.28, 45, 2.7], 1, stil(6, '#64748b'))
			]
		}
	};
}

/** @returns {ReturnType<typeof blanko>} */
function marine() {
	const farben = { akzent: '#a8741a', text: '#0f172a', nebentext: '#475569' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				box('seitenband', MARINE, [0, 0, 26, KARTE_HOEHE]),
				box('seitenlinie', GOLD, [26, 0, 1.2, KARTE_HOEHE]),
				el('photo', 'photo', '', [3, 14, 20, 24], 2),
				// Logo unter dem Foto im Seitenband, damit der Schulname rechts die volle
				// Spaltenbreite bekommt — in 39 mm brach ein langer Name in die Adresse.
				el('logo', 'logo', '', [8, 40.98, 10, 10], 2),
				...schulkopf(31, 3, 51.6, { name: '#16324f', adresse: '#64748b' }, 2),
				...personenBlock(farben, 31, 19),
				el('barcode', 'barcode', '', [31, 39.68, 30, 11], 1)
			]
		},
		back: standardRueckseite(
			WEISS_THEME,
			[
				box('back-kopfband', MARINE, [0, 0, KARTE_BREITE, 6.4]),
				box('back-kopflinie', GOLD, [0, 6.4, KARTE_BREITE, 0.8])
			],
			10.5
		)
	};
}

/** @returns {ReturnType<typeof blanko>} */
function indigoKopfkarte() {
	const farben = { akzent: INDIGO_AKZENT, text: '#111827', nebentext: '#475569' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				box('kopfkarte', INDIGO, [3, 3, 79.6, 14], 0, 2.5),
				el('logo', 'logo', '', [6, 5.2, 9.5, 9.5], 2),
				...schulkopf(17.5, 5.8, 62, { name: '#ffffff', adresse: INDIGO_HELL }),
				el('photo', 'photo', '', [3, 20.5, 20, 24], 2),
				...personenBlock(farben, 26.5, 21),
				el('barcode', 'barcode', '', [26.5, 39.68, 30, 11], 1),
				box('fusslinie', INDIGO_AKZENT, [3, 52.6, 79.6, 0.9], 0, 0.45)
			]
		},
		back: standardRueckseite(
			WEISS_THEME,
			[box('back-kopflinie', INDIGO, [3, 3, 79.6, 1.2], 0, 0.6)],
			8
		)
	};
}

/** @returns {ReturnType<typeof blanko>} */
function sonnenstreifen() {
	const farben = { akzent: ORANGE, text: '#1f2937', nebentext: '#57534e' };
	return {
		front: {
			theme: WEISS_THEME,
			elements: [
				box('topstreifen', AMBER, [0, 0, KARTE_BREITE, 1.6]),
				...schulkopf(3, 3, 60, { name: '#1f2937', adresse: '#78716c' }),
				el('logo', 'logo', '', [72.6, 3, 10, 10], 2),
				el('photo', 'photo', '', [3, 16.5, 20, 24], 2),
				...personenBlock(farben, 26.5, 17.5),
				el('barcode', 'barcode', '', [26.5, 37, 30, 11], 1),
				box('streifen-1', AMBER, [0, 49.2, KARTE_BREITE, 1.6]),
				box('streifen-2', ORANGE, [0, 50.8, KARTE_BREITE, 1.6]),
				box('streifen-3', ROT, [0, 52.4, KARTE_BREITE, 1.58])
			]
		},
		back: standardRueckseite(
			WEISS_THEME,
			[
				box('back-streifen-1', AMBER, [0, 0, KARTE_BREITE, 1.6]),
				box('back-streifen-2', ORANGE, [0, 1.6, KARTE_BREITE, 1.6]),
				box('back-streifen-3', ROT, [0, 3.2, KARTE_BREITE, 1.6])
			],
			8.5
		)
	};
}

const VORLAGEN_BAU = /** @type {Record<string, () => ReturnType<typeof blanko>>} */ ({
	blanko,
	'schwarz-gruen': schwarzGruen,
	waldgruen,
	'reis-welle': reisWelle,
	marine,
	'indigo-kopfkarte': indigoKopfkarte,
	sonnenstreifen
});

/**
 * @param {string} kennung
 * @returns {{front: {theme: string, elements: any[]}, back: {theme: string, elements: any[]}} | null}
 */
export function vorlage(kennung) {
	return VORLAGEN_BAU[kennung]?.() ?? null;
}

/**
 * Übernimmt eine Vorlage in den zentralen Store (beide Seiten, Elemente UND Theme).
 * Die Rückfrage gehört dem Aufrufer (idDesignPersistenz) — wie bei resetDesign().
 *
 * @param {string} kennung
 * @returns {boolean} false bei unbekannter Kennung (Store bleibt unangetastet)
 */
export function wendeVorlageAn(kennung) {
	const v = vorlage(kennung);
	if (!v) return false;
	// Ein bereits hochgeladenes Schullogo überlebt den Vorlagenwechsel: Das leere
	// Logofeld der Vorlage bedeutet „noch kein Logo", nicht „Logo entfernen" — sonst
	// müsste die Schule ihre Logodatei nach jedem Vorlagenklick erneut hochladen.
	const bisherigesLogo = idStore.front.elements.find((e) => e.type === 'logo' && e.content);
	if (bisherigesLogo) {
		const logoNeu = v.front.elements.find((e) => e.type === 'logo');
		if (logoNeu) logoNeu.content = bisherigesLogo.content;
	}
	idStore.front.elements = v.front.elements;
	idStore.front.theme = v.front.theme;
	idStore.back.elements = v.back.elements;
	idStore.back.theme = v.back.theme;
	return true;
}
