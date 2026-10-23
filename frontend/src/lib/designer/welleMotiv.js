/**
 * Das Motiv der Vorlage „Reis-Welle": Schallwellen-Balken (Philipp Reis, Telefon 1861) als
 * SVG-Bild über die ganze Kartenbreite. Es bleibt ein Bild, weil 40 Balken kein Rechteck sind.
 */

/** Balkenhöhen von links nach rechts; 62 ist der höchste Balken. */
const HOEHEN = [
	8, 12, 10, 16, 22, 30, 44, 58, 50, 38, 26, 18, 12, 10, 14, 20, 32, 48, 62, 54, 40, 28, 16, 10, 8,
	12, 18, 28, 42, 56, 46, 34, 22, 14, 10, 8, 12, 16, 10, 8
];

/**
 * @param {string} farbe
 * @param {number} deckkraft
 * @param {number} hoehe in Zehntelmillimetern; die Breite ist 856, die ganze Karte
 */
export function welleSvg(farbe, deckkraft, hoehe) {
	const schritt = 856 / HOEHEN.length;
	const balken = HOEHEN.map((h, i) => {
		const bh = (h / 62) * (hoehe - 6);
		const x = (i * schritt + schritt / 2 - 3).toFixed(1);
		return `<rect x="${x}" y="${(hoehe / 2 - bh / 2).toFixed(1)}" width="6" rx="3" height="${bh.toFixed(1)}" fill="${farbe}" fill-opacity="${deckkraft}"/>`;
	}).join('');
	return (
		'data:image/svg+xml;utf8,' +
		encodeURIComponent(
			`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 856 ${hoehe}">${balken}</svg>`
		)
	);
}
