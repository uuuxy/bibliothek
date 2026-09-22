/**
 * Die eine Adresse, unter der ein Strichcode-Bild geholt wird.
 *
 * Der Server liefert `/api/barcode` mit `Cache-Control: max-age=31536000` — ein Jahr.
 * Das ist richtig, solange dieselbe Adresse dasselbe Bild bedeutet: Ein Ausweisbogen mit
 * 30 Karten holt sonst 30-mal dasselbe PNG.
 *
 * Genau diese Annahme brach am 17.09.2026. Der Drucker stellte von Code 39 (mit
 * Prüfzeichen) auf Code 128 um, die Adresse blieb gleich — und jedes Gerät, das die Karte
 * vorher einmal angezeigt hatte, zeigte weiter den ALTEN Strichcode. Auf dem Handy stand
 * unter der Karte „S-10001", gescannt wurde „S-10001N", und der Server suchte eine Nummer,
 * die es nicht gibt. Der Code war da schon repariert; im Browser lag noch das Bild von
 * gestern.
 *
 * Deshalb trägt die Adresse die Fassung der Kodierung. Wer am Erzeuger etwas ändert
 * (api/barcode_generate.go), zählt FASSUNG hoch — dann ist es für jeden Browser ein neues
 * Bild, ohne dass jemand einen Cache leeren muss.
 */
export const FASSUNG = 'c128';

/**
 * @param {string | undefined | null} inhalt die Nummer, die im Strichcode stehen soll
 * @param {{ qr?: boolean, width?: number, height?: number }} [optionen]
 * @returns {string}
 */
export function strichcodeBildUrl(inhalt, optionen = {}) {
	const { qr = false, width, height } = optionen;
	const teile = [`content=${encodeURIComponent(inhalt ?? '')}`, `qr=${qr ? 'true' : 'false'}`];
	if (width) teile.push(`width=${width}`);
	if (height) teile.push(`height=${height}`);
	teile.push(`v=${FASSUNG}`);
	return `/api/barcode?${teile.join('&')}`;
}

/** Druckauflösung, in der das Strichcode-Bild für den Ausweis erzeugt wird. */
export const DRUCK_DPI = 300;
/** Höhe der Nummernzeile unter dem Strichcode (6,5 pt plus Abstand), in mm. */
export const NUMMERNZEILE_MM = 3;
/** Kürzer druckt kein Scanner-tauglicher Code 128 — Untergrenze der Bildhöhe in mm. */
export const MINDESTHOEHE_MM = 8;

/**
 * Bildgröße für das Strichcode-Element eines Ausweises, aus den Millimetern des
 * Elements im Designer.
 *
 * Bis zum 22.09.2026 hatten beide Renderer feste Pixelwerte (200 × 50, QR 80 × 80) — und
 * das Papier (CardFace) obendrein eine feste Anzeigehöhe von 8 mm, egal wie hoch das
 * Element im Designer gezogen war. Der Bildschirm skalierte ins Element, das Papier
 * nicht: zwei Renderer, zwei Größen (OFFEN.md 5.5). Jetzt kommt die Größe aus dem
 * Element, in Druckauflösung, für beide Renderer aus dieser einen Funktion.
 *
 * @param {{ width: number, height: number }} el Element in mm
 * @param {'code39'|'qr'|string} barcodeType
 * @returns {{ qr: boolean, width: number, height: number }} Pixel für /api/barcode
 */
export function strichcodeBildOptionenFuerElement(el, barcodeType) {
	const px = (/** @type {number} */ mm) => Math.round((mm / 25.4) * DRUCK_DPI);
	const bildHoeheMm = Math.max(MINDESTHOEHE_MM, (el?.height ?? 0) - NUMMERNZEILE_MM);
	if (barcodeType === 'qr') {
		const seite = px(Math.min(el?.width ?? 0, bildHoeheMm));
		return { qr: true, width: seite, height: seite };
	}
	return { qr: false, width: px(el?.width ?? 0), height: px(bildHoeheMm) };
}
