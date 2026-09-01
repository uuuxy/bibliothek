import {
	idStore,
	PLATZHALTER_SCHULNAME,
	PLATZHALTER_ADRESSE,
	FRONT_THEME_DEFAULT
} from './idDesignerStore.svelte.js';

/**
 * @file ausweisVorlagen.js
 * Fertige Design-Vorlagen für den Ausweis-Designer: Ein Klick füllt beide Seiten mit
 * einem abgestimmten Layout, danach ist alles wie gewohnt frei editierbar.
 *
 * Warum als Daten und nicht als neue Renderer-Fähigkeiten: CardFace.svelte ist die
 * einzige Render-Quelle aller Druckwege. Kopfbalken, Farbband und Wellen-Motiv sind
 * deshalb gewöhnliche Bild-Elemente (eingebettetes SVG, unterste Ebene) — sie drucken
 * überall dort korrekt, wo heute schon Bilder drucken, ohne neuen Code im Renderer.
 *
 * Bewusste Regeln, die alle Vorlagen einhalten:
 *   - Schulname/Adresse tragen die IDs 'header'/'address' mit den PLATZHALTER-Texten,
 *     damit wendeSchulstammdatenAn() sie wie beim Standard-Design ersetzt.
 *   - Das Logofeld bleibt LEER (die Schule lädt ihre echte Logodatei hoch); leere
 *     Logofelder lässt der Druck einfach weg.
 *   - Keine Klassen-/Schuljahreszeile — der Ausweis gilt die ganze Schulzeit
 *     (Begründung am details-Filter in idDesignerStore.applySeite).
 *   - Der Barcode liegt IMMER auf heller Fläche: CardFace rendert die Nummer dunkel,
 *     auf dunklem Grund wäre sie unlesbar.
 */

/** Grün der Schul-Homepage; Kopfleiste dort ist schwarz. */
const GRUEN = '#76b82a';
const GRUEN_DUNKEL = '#4a7a16';
const SCHWARZ = '#101410';

/** Muss als Literal hier stehen, damit Tailwind die Klassen in den Build aufnimmt. */
export const WALDGRUEN_THEME =
	'bg-linear-to-tr from-[#16330a] via-[#2d5a12] to-[#3f7418] text-white border-[#0d2405]';
const WEISS_THEME = FRONT_THEME_DEFAULT;

/** Für das Auswahl-Dropdown in der Werkzeugleiste. */
export const AUSWEIS_VORLAGEN = [
	{ value: 'schwarz-gruen', label: 'Schwarz-Grün (Webauftritt)' },
	{ value: 'waldgruen', label: 'Waldgrün-Verlauf' },
	{ value: 'reis-welle', label: 'Reis-Welle' }
];

/**
 * @param {number} breite SVG-Koordinaten (10 Einheiten = 1 mm)
 * @param {number} hoehe
 * @param {string} inhalt
 */
function svgUrl(breite, hoehe, inhalt) {
	return (
		'data:image/svg+xml;utf8,' +
		encodeURIComponent(
			`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${breite} ${hoehe}">${inhalt}</svg>`
		)
	);
}

/**
 * Schallwellen-Balken (Philipp Reis, Telefon 1861) — Seitenverhältnis 856:90.
 * @param {number} deckkraft
 */
function welleSvg(deckkraft) {
	const hoehen = [
		8, 12, 10, 16, 22, 30, 44, 58, 50, 38, 26, 18, 12, 10, 14, 20, 32, 48, 62, 54, 40, 28, 16, 10,
		8, 12, 18, 28, 42, 56, 46, 34, 22, 14, 10, 8, 12, 16, 10, 8
	];
	const schritt = 856 / hoehen.length;
	const balken = hoehen
		.map((h, i) => {
			const bh = (h / 62) * 84;
			const x = (i * schritt + schritt / 2 - 3).toFixed(1);
			return `<rect x="${x}" y="${(45 - bh / 2).toFixed(1)}" width="6" rx="3" height="${bh.toFixed(1)}" fill="${GRUEN}" fill-opacity="${deckkraft}"/>`;
		})
		.join('');
	return svgUrl(856, 90, balken);
}

const KOPFBAND = svgUrl(
	856,
	143,
	`<rect width="856" height="135" fill="${SCHWARZ}"/><rect y="135" width="856" height="8" fill="${GRUEN}"/>`
);
const KOPFBAND_SCHMAL = svgUrl(
	856,
	80,
	`<rect width="856" height="72" fill="${SCHWARZ}"/><rect y="72" width="856" height="8" fill="${GRUEN}"/>`
);
const FUSSBAND_HELL = svgUrl(856, 139, '<rect width="856" height="139" fill="#f2f7e9"/>');
const PANEL_WEISS = svgUrl(776, 310, '<rect width="776" height="310" rx="30" fill="#ffffff"/>');
const UNTERSCHRIFT_LINIE = svgUrl(320, 8, '<rect y="6" width="320" height="2" fill="#3a4234"/>');

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

/**
 * @param {string} type
 * @param {string} id
 * @param {string} content
 * @param {number} x
 * @param {number} y
 * @param {number} width
 * @param {number} height
 * @param {number} zIndex
 * @param {ReturnType<typeof stil>} [style]
 */
function el(type, id, content, x, y, width, height, zIndex, style) {
	const proportional = type === 'photo' || type === 'logo' || type === 'barcode';
	return { id, type, content, x, y, width, height, zIndex, show: true, proportional, style };
}

/**
 * Die drei festen Blöcke, die jede Vorlagen-Vorderseite gleich aufbaut.
 * @param {{akzent: string, text: string, nebentext: string}} farben
 * @param {number} [x]
 * @param {number} [yTitel]
 */
function personenBlock(farben, x = 30, yTitel = 16.5) {
	return [
		el('text', 'title', 'SCHÜLERAUSWEIS', x, yTitel, 50, 4, 1, stil(6, farben.akzent, 'bold')),
		el('name', 'name', '', x, yTitel + 4.5, 51, 8, 1, stil(10, farben.text, 'bold')),
		el('validity', 'validity', '', x, yTitel + 13, 48, 5, 1, stil(7.5, farben.nebentext))
	];
}

/**
 * Unterschriftszeile für die Rückseiten (Linie als Bild + Beschriftung).
 * @param {number} x
 * @param {number} y
 */
function unterschrift(x, y) {
	return [
		el('image', 'back-signatur-linie', UNTERSCHRIFT_LINIE, x, y, 32, 0.8, 1),
		el(
			'text',
			'back-signatur',
			'Unterschrift Schüler/in',
			x,
			y + 1.2,
			32,
			3.5,
			1,
			stil(5.5, '#64748b')
		)
	];
}

/**
 * @param {string} kennung
 * @returns {{front: {theme: string, elements: any[]}, back: {theme: string, elements: any[]}} | null}
 */
export function vorlage(kennung) {
	if (kennung === 'schwarz-gruen') {
		const farben = { akzent: GRUEN_DUNKEL, text: SCHWARZ, nebentext: '#475569' };
		return {
			front: {
				theme: WEISS_THEME,
				elements: [
					el('image', 'kopfband', KOPFBAND, 0, 0, 85.6, 14.3, 0),
					el('logo', 'logo', '', 4, 2.2, 10, 10, 2),
					el(
						'header',
						'header',
						PLATZHALTER_SCHULNAME,
						16.5,
						4.5,
						64,
						5.5,
						1,
						stil(7.5, '#ffffff', 'bold')
					),
					el('photo', 'photo', '', 4, 17, 22, 27, 2),
					...personenBlock(farben),
					el('barcode', 'barcode', '', 30, 35.5, 30, 11, 1),
					el('address', 'address', PLATZHALTER_ADRESSE, 4, 48.7, 77, 4, 1, stil(5.5, '#64748b'))
				]
			},
			back: {
				theme: WEISS_THEME,
				elements: [
					el('image', 'back-kopfband', KOPFBAND_SCHMAL, 0, 0, 85.6, 8, 0),
					el('text', 'back-header', 'Bestimmungen', 5, 11, 40, 5, 1, stil(8, SCHWARZ, 'bold')),
					el('text', 'back-info', BESTIMMUNGEN, 5, 17.5, 56, 16, 1, stil(7, '#3a4234')),
					...unterschrift(5, 40),
					el('text', 'back-fund', FUNDHINWEIS, 5, 48.7, 70, 4, 1, stil(6, '#64748b'))
				]
			}
		};
	}
	if (kennung === 'waldgruen') {
		const farben = { akzent: '#b9e07f', text: '#ffffff', nebentext: '#d8ecc0' };
		return {
			front: {
				theme: WALDGRUEN_THEME,
				elements: [
					el('image', 'fussband', FUSSBAND_HELL, 0, 40, 85.6, 13.9, 0),
					el('logo', 'logo', '', 4, 2.5, 9.5, 9.5, 2),
					el(
						'header',
						'header',
						PLATZHALTER_SCHULNAME,
						16,
						4.2,
						64,
						5.5,
						1,
						stil(7.5, '#ffffff', 'bold')
					),
					el('photo', 'photo', '', 4, 14.5, 21, 24, 2),
					...personenBlock(farben, 29, 14.5),
					el('barcode', 'barcode', '', 29, 41.5, 30, 11, 1),
					el('address', 'address', PLATZHALTER_ADRESSE, 4, 42, 23, 10, 1, stil(5, '#4a5546'))
				]
			},
			back: {
				theme: WALDGRUEN_THEME,
				elements: [
					el('image', 'back-panel', PANEL_WEISS, 4, 9, 77.6, 31, 0),
					el('text', 'back-header', 'Bestimmungen', 8, 12.5, 40, 5, 1, stil(8, SCHWARZ, 'bold')),
					el('text', 'back-info', BESTIMMUNGEN, 8, 18.5, 58, 15, 1, stil(7, '#3a4234')),
					...unterschrift(8, 34.5),
					el('text', 'back-fund', FUNDHINWEIS, 4, 46, 70, 4, 1, stil(6, '#d8ecc0'))
				]
			}
		};
	}
	if (kennung === 'reis-welle') {
		const farben = { akzent: GRUEN_DUNKEL, text: SCHWARZ, nebentext: '#475569' };
		return {
			front: {
				theme: WEISS_THEME,
				elements: [
					el(
						'header',
						'header',
						PLATZHALTER_SCHULNAME,
						4,
						3.5,
						60,
						5.5,
						1,
						stil(7.5, SCHWARZ, 'bold')
					),
					el('address', 'address', PLATZHALTER_ADRESSE, 4, 8.7, 60, 3.5, 1, stil(5.5, '#64748b')),
					el('logo', 'logo', '', 71.5, 3, 10, 10, 2),
					el('image', 'welle', welleSvg(1), 0, 13, 85.6, 9, 0),
					el(
						'text',
						'welle-text',
						'Philipp Reis baute in Friedrichsdorf das erste Telefon (1861).',
						28,
						22.3,
						53.6,
						3.5,
						1,
						stil(4.5, '#7a8474', 'normal', 'right')
					),
					el('photo', 'photo', '', 4, 26.5, 20, 24.5, 2),
					...personenBlock(farben, 28, 26.5),
					el('barcode', 'barcode', '', 28, 42.5, 30, 11, 1)
				]
			},
			back: {
				theme: WEISS_THEME,
				elements: [
					el('text', 'back-header', 'Bestimmungen', 4, 4.5, 40, 5, 1, stil(8, SCHWARZ, 'bold')),
					el('text', 'back-info', BESTIMMUNGEN, 4, 10.5, 62, 15, 1, stil(7, '#3a4234')),
					el('image', 'back-welle', welleSvg(0.25), 0, 26, 85.6, 12, 0),
					...unterschrift(50, 43.5),
					el('text', 'back-fund', FUNDHINWEIS, 4, 48.7, 45, 4, 1, stil(6, '#64748b'))
				]
			}
		};
	}
	return null;
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
	// (Raster-Fund 01.09.2026, Frage 8 Lebenszyklus: Was überlebt den Ersetzen-Pfad?)
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
