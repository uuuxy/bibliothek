/**
 * @file idDesignAltbestand.js
 * Heilung des zentral gespeicherten Designs beim Laden: Bänder, die eine frühere
 * Vorlagen-Fassung als SVG-BILD gespeichert hat, werden in Farbflächen ('box')
 * übersetzt.
 *
 * Warum beim Laden und nicht nur in den Vorlagen: Das Design liegt zentral in der
 * Datenbank; applyDesign() lädt es und überschreibt jede Vorgabe. Eine Installation,
 * die „Schwarz-Grün" zwischen dem 01.09.2026 (Vorlagen eingeführt) und dem Umbau auf
 * Farbflächen angewendet hat, trüge die Bild-Bänder sonst für immer weiter — mit
 * genau den weißen Randstreifen und der nicht editierbaren Farbe, die der Umbau
 * beseitigt. Derselbe Weg, auf dem schon der Musterstadt-Kopf und die Klassenzeile
 * hängenblieben (applySeite in idDesignerStore.svelte.js).
 *
 * Die Farben sind die der ALTEN Vorlagen (eingefroren — sie beschreiben, was damals
 * gezeichnet wurde, nicht was heute gilt). Ein Bild mit einer dieser IDs, das KEIN
 * eingebettetes SVG ist (jemand hat eine eigene Datei hochgeladen), bleibt unangetastet.
 */

const SVG_DATA_URI = 'data:image/svg+xml';

/** Kopfband alt: 856×143 — 135 schwarz, darunter 8 grün. Schmales Band: 72 + 8. */
const ALT = {
	schwarz: '#101410',
	gruen: '#76b82a',
	fussbandHell: '#f2f7e9',
	panelWeiss: '#ffffff',
	unterschriftLinie: '#3a4234'
};

/**
 * @param {any} alt das bisherige Bild-Element (liefert Lage, Sichtbarkeit, Ebene)
 * @param {string} id
 * @param {string} farbe
 * @param {number} y
 * @param {number} height
 * @param {number} [radius]
 */
function box(alt, id, farbe, y, height, radius = 0) {
	return {
		id,
		type: 'box',
		content: '',
		x: alt.x,
		y,
		width: alt.width,
		height,
		zIndex: alt.zIndex ?? 0,
		show: alt.show !== false,
		proportional: false,
		style: { color: farbe, radius }
	};
}

/**
 * Kopfband mit Farblinie darunter: aus EINEM Bild werden ZWEI Flächen.
 * @param {any} alt
 * @param {string} bandId
 * @param {string} linieId
 * @param {number} anteilBand Anteil der Bandhöhe am Bild (135/143 bzw. 72/80)
 */
function kopfbandMitLinie(alt, bandId, linieId, anteilBand) {
	const hBand = Math.round(alt.height * anteilBand * 100) / 100;
	return [
		box(alt, bandId, ALT.schwarz, alt.y, hBand),
		box(alt, linieId, ALT.gruen, alt.y + hBand, Math.round((alt.height - hBand) * 100) / 100)
	];
}

/**
 * @param {any[]} elements Elemente einer Seite, wie aus der Datenbank geladen
 * @returns {any[]} dieselbe Liste, Alt-Bänder durch Farbflächen ersetzt
 */
export function heileAltBaender(elements) {
	/** @type {any[]} */
	const out = [];
	for (const el of elements) {
		const istAltBild =
			el?.type === 'image' && typeof el.content === 'string' && el.content.startsWith(SVG_DATA_URI);
		if (!istAltBild) {
			out.push(el);
			continue;
		}
		switch (el.id) {
			case 'kopfband':
				out.push(...kopfbandMitLinie(el, 'kopfband', 'kopflinie', 135 / 143));
				break;
			case 'back-kopfband':
				out.push(...kopfbandMitLinie(el, 'back-kopfband', 'back-kopflinie', 72 / 80));
				break;
			case 'fussband':
				out.push(box(el, 'fussband', ALT.fussbandHell, el.y, el.height));
				break;
			case 'back-panel':
				out.push(box(el, 'back-panel', ALT.panelWeiss, el.y, el.height, 3));
				break;
			case 'back-signatur-linie':
				// Die Linie lag im unteren Viertel eines 0,8 mm hohen Bildes.
				out.push(box(el, 'back-signatur-linie', ALT.unterschriftLinie, el.y + 0.3, 0.5));
				break;
			default:
				// Wellen-Motiv und alles, was wir nicht kennen: bleibt ein Bild.
				out.push(el);
		}
	}
	return out;
}
