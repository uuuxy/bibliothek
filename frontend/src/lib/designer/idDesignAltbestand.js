/**
 * @file idDesignAltbestand.js
 * Heilung des zentral gespeicherten Designs beim Laden: Bänder, die eine frühere
 * Vorlagen-Fassung als SVG-Bild gespeichert hat, werden in Farbflächen ('box') übersetzt,
 * und ein fest eingetippter Titel „Schülerausweis" wird zum Dokumenttyp-Feld, das die Art
 * des Lesers liest.
 *
 * Warum beim Laden und nicht nur in den Vorlagen: Das Design liegt zentral in der
 * Datenbank; applyDesign() lädt es und überschreibt jede Vorgabe. Ein gespeichertes Design
 * mit Bild-Bändern trüge sonst für immer die weißen Randstreifen und die nicht editierbare
 * Farbe weiter. Derselbe Weg wie für den Musterstadt-Kopf und die Klassenzeile (applySeite
 * in idDesignerStore.svelte.js).
 *
 * Die Farben sind die der alten Vorlagen, eingefroren: Sie beschreiben, was gespeichert
 * wurde, nicht was heute gilt. Ein Bild mit einer dieser IDs, das kein eingebettetes SVG
 * ist (jemand hat eine eigene Datei hochgeladen), bleibt unangetastet.
 */

const SVG_ANFANG = 'data:image/svg+xml';

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
 * Ein fest eingetippter Titel „Schülerausweis" wird zum Dokumenttyp-Feld.
 *
 * Als gewöhnliches Textelement stünde auf dem Ausweis einer Lehrkraft „Schülerausweis",
 * denn der Text kennt die Art des Lesers nicht. Ein gespeichertes Design trägt diesen Text
 * weiter, bis er hier ersetzt wird; derselbe Weg wie bei den Bild-Bändern.
 *
 * Die Vorlagen schreiben den Titel in Großbuchstaben. Weil der Inhalt jetzt aus der Art
 * kommt, wird aus dieser Schreibweise eine Angabe am Stil — sonst stünde auf einer
 * Karte, die bisher „SCHÜLERAUSWEIS" trug, plötzlich „Lehrerausweis" in gemischter
 * Schrift.
 *
 * @param {any} el
 * @returns {any|null} das ersetzte Element, oder null wenn dieses hier nicht gemeint ist
 */
function heileTitel(el) {
	if (el?.type !== 'text' || typeof el.content !== 'string') return null;
	const wort = el.content.trim();
	if (wort.toLocaleLowerCase('de-DE') !== 'schülerausweis') return null;
	const grossgeschrieben = wort === wort.toLocaleUpperCase('de-DE');
	return {
		...el,
		type: 'dokumenttyp',
		content: '',
		style: grossgeschrieben ? { ...el.style, textTransform: 'uppercase' } : { ...el.style }
	};
}

/**
 * @param {any[]} elements Elemente einer Seite, wie aus der Datenbank geladen
 * @returns {any[]} dieselbe Liste, Alt-Bänder durch Farbflächen ersetzt
 */
export function heileAltBaender(elements) {
	/** @type {any[]} */
	const out = [];
	for (const el of elements) {
		const geheilterTitel = heileTitel(el);
		if (geheilterTitel) {
			out.push(geheilterTitel);
			continue;
		}
		const istAltBild =
			el?.type === 'image' && typeof el.content === 'string' && el.content.startsWith(SVG_ANFANG);
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
