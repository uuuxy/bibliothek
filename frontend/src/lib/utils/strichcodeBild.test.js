import { describe, it, expect } from 'vitest';
import {
	strichcodeBildUrl,
	strichcodeBildOptionenFuerElement,
	FASSUNG,
	DRUCK_DPI,
	NUMMERNZEILE_MM,
	MINDESTHOEHE_MM
} from './strichcodeBild.js';

describe('Adresse des Strichcode-Bildes', () => {
	// Der Server erlaubt ein Jahr Cache. Das ist nur dann richtig, wenn dieselbe Adresse
	// dasselbe Bild bedeutet — sonst zeigt ein Browser nach einer Umstellung der Kodierung
	// weiter den Strichcode von gestern. Genau das ist am 17.09.2026 passiert: Unter der
	// Karte stand „S-10001", der gecachte Strichcode lieferte „S-10001N".
	it('trägt die Fassung der Kodierung', () => {
		expect(strichcodeBildUrl('S-10001')).toContain(`v=${FASSUNG}`);
	});

	it('codiert den Inhalt, damit Sonderzeichen die Adresse nicht zerlegen', () => {
		expect(strichcodeBildUrl('LMF 4711/2')).toContain('content=LMF%204711%2F2');
	});

	it('nennt Größe nur, wenn eine gewünscht ist', () => {
		expect(strichcodeBildUrl('B-1')).not.toContain('width=');
		expect(strichcodeBildUrl('B-1', { width: 200, height: 50 })).toContain('width=200&height=50');
	});

	it('unterscheidet QR vom Strichcode', () => {
		expect(strichcodeBildUrl('B-1', { qr: true })).toContain('qr=true');
		expect(strichcodeBildUrl('B-1')).toContain('qr=false');
	});
});

// OFFEN.md 5.5 (22.09.2026): Die Barcode-Höhe im Druck war fest (8 mm), egal wie hoch
// das Element im Designer gezogen war. Jetzt bestimmt das Element die Bildgröße — in
// Druckauflösung, für Bildschirm und Papier aus derselben Funktion.
describe('Strichcode-Bildgröße aus dem Element', () => {
	const px = (/** @type {number} */ mm) => Math.round((mm / 25.4) * DRUCK_DPI);

	it('rechnet Millimeter des Elements in Druckpixel um, abzüglich der Nummernzeile', () => {
		const o = strichcodeBildOptionenFuerElement({ width: 30, height: 20 }, 'code39');
		expect(o).toEqual({ qr: false, width: px(30), height: px(20 - NUMMERNZEILE_MM) });
	});

	it('unterschreitet die Scanner-Untergrenze nicht, auch wenn das Element kleiner ist', () => {
		const o = strichcodeBildOptionenFuerElement({ width: 30, height: 5 }, 'code39');
		expect(o.height).toBe(px(MINDESTHOEHE_MM));
	});

	it('hält den QR-Code quadratisch an der kürzeren Seite', () => {
		const o = strichcodeBildOptionenFuerElement({ width: 30, height: 14 }, 'qr');
		expect(o).toEqual({
			qr: true,
			width: px(14 - NUMMERNZEILE_MM),
			height: px(14 - NUMMERNZEILE_MM)
		});
	});
});
