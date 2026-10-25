/**
 * Die eine Adresse, unter der ein Strichcode-Bild geholt wird.
 *
 * `/api/barcode` antwortet wie jede Antwort unter /api/ mit `no-store`
 * (internal/middleware/sicherheitskopfzeilen.go): Die Adresse trägt den Inhalt, beim Ausweis
 * also die Ausweisnummer.
 *
 * Die Adresse trägt außerdem die Fassung der Kodierung: Ein Bild, das ein Browser oder ein
 * Zwischenspeicher unter derselben Adresse aufhebt, zeigte sonst nach einer Umstellung den
 * alten Strichcode, und gescannt würde eine Nummer, die es nicht gibt. Wer am Erzeuger etwas
 * ändert (pkg/strichcode), zählt FASSUNG hoch; dann ist es für jeden Browser ein neues Bild,
 * ohne dass jemand einen Cache leeren muss.
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
 * Die Größe kommt aus dem Element, in Druckauflösung, für beide Renderer (Bildschirm und
 * Kartenseite) aus dieser einen Funktion: Mit festen Pixelwerten zeigten Bildschirm und
 * Papier zwei Größen.
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
