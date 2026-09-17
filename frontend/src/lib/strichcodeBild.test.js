import { describe, it, expect } from 'vitest';
import { strichcodeBildUrl, FASSUNG } from './strichcodeBild.js';

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
