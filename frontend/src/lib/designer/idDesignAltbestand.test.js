import { describe, it, expect, beforeEach } from 'vitest';
import { heileAltBaender } from './idDesignAltbestand.js';
import { applyDesign, idStore, resetDesign } from './idDesignerStore.svelte.js';

// So sah das Kopfband der Vorlage „Schwarz-Grün" bis zum Umbau auf Farbflächen aus:
// ein Bild-Element mit eingebettetem SVG (schwarzes Rechteck + grüne Linie).
const ALT_KOPFBAND = {
	id: 'kopfband',
	type: 'image',
	content: 'data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E',
	x: 0,
	y: 0,
	width: 85.6,
	height: 14.3,
	zIndex: 0,
	show: true,
	proportional: false
};

describe('heileAltBaender', () => {
	it('übersetzt das alte Kopfband-Bild in Kopfband-Fläche + Grünlinie (zwei Flächen aus einem Bild)', () => {
		const neu = heileAltBaender([ALT_KOPFBAND]);
		expect(neu.map((e) => [e.id, e.type])).toEqual([
			['kopfband', 'box'],
			['kopflinie', 'box']
		]);
		const [band, linie] = neu;
		expect(band.style.color).toBe('#101410');
		expect(linie.style.color).toBe('#76b82a');
		// Volle Kartenbreite bleibt, die Höhen ergeben zusammen die alte Bildhöhe.
		expect(band.x).toBe(0);
		expect(band.width).toBe(85.6);
		expect(band.height + linie.height).toBeCloseTo(14.3, 2);
		expect(linie.y).toBeCloseTo(band.height, 2);
	});

	it('lässt ein Bild mit derselben ID in Ruhe, wenn es KEIN eingebettetes SVG ist (eigene Datei)', () => {
		const eigen = { ...ALT_KOPFBAND, content: 'data:image/png;base64,AAAA' };
		expect(heileAltBaender([eigen])).toEqual([eigen]);
	});

	it('lässt das Wellen-Motiv und unbekannte Bilder unangetastet', () => {
		const welle = { ...ALT_KOPFBAND, id: 'welle' };
		expect(heileAltBaender([welle])).toEqual([welle]);
	});

	it('übernimmt Sichtbarkeit und Ebene, gibt dem Panel seine runden Ecken zurück', () => {
		const panel = { ...ALT_KOPFBAND, id: 'back-panel', show: false, zIndex: 4, height: 31 };
		const [neu] = heileAltBaender([panel]);
		expect(neu.type).toBe('box');
		expect(neu.show).toBe(false);
		expect(neu.zIndex).toBe(4);
		expect(neu.style.radius).toBe(3);
	});

	it('lässt Flächen, Texte und alles Übrige durch', () => {
		const text = {
			id: 'title',
			type: 'text',
			content: 'X',
			x: 1,
			y: 1,
			width: 5,
			height: 2,
			zIndex: 1
		};
		const flaeche = {
			id: 'kopfband',
			type: 'box',
			content: '',
			x: 0,
			y: 0,
			width: 85.6,
			height: 13,
			zIndex: 0
		};
		expect(heileAltBaender([text, flaeche])).toEqual([text, flaeche]);
	});
});

describe('applyDesign heilt den Altbestand beim Laden', () => {
	beforeEach(() => resetDesign());

	it('ein zentral gespeichertes Design mit Bild-Bändern kommt als Farbflächen im Store an', () => {
		applyDesign({
			front: { elements: [ALT_KOPFBAND], theme: 'bg-white text-black border-slate-200' },
			back: { elements: [{ ...ALT_KOPFBAND, id: 'back-kopfband', height: 8 }], theme: 'x' }
		});
		expect(idStore.front.elements.map((e) => e.id)).toEqual(['kopfband', 'kopflinie']);
		expect(idStore.front.elements.every((e) => e.type === 'box')).toBe(true);
		expect(idStore.back.elements.map((e) => e.id)).toEqual(['back-kopfband', 'back-kopflinie']);
	});
});
