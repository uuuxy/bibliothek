import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { passendeMasse, verkleinereBild } from './bildVerkleinern.js';

describe('passendeMasse', () => {
	it('verkleinert im Seitenverhältnis auf den Rahmen und vergrößert nie', () => {
		expect(passendeMasse(4000, 3000, 1200, 1200)).toEqual({ breite: 1200, hoehe: 900 });
		expect(passendeMasse(3000, 4000, 1200, 1200)).toEqual({ breite: 900, hoehe: 1200 });
		expect(passendeMasse(1200, 1800, 600, 900)).toEqual({ breite: 600, hoehe: 900 });
		expect(passendeMasse(300, 200, 1200, 1200)).toEqual({ breite: 300, hoehe: 200 });
	});
});

// jsdom dekodiert keine Bilder und zeichnet nicht; Bild und Zeichenfläche antworten hier, wie ein
// Browser es täte. Safari kodiert kein WebP: toBlob liefert dann PNG, ein großes Foto lag damit
// über der Grenze des Ausweis-Designers (e2e/ausweis-bild-hochladen.spec.js, in WebKit rot).
describe('verkleinereBild', () => {
	/** @type {string[]} */
	let angefragt;
	let kannWebp = false;
	const original = { erzeuge: URL.createObjectURL, gibFrei: URL.revokeObjectURL };
	const rahmen = { maxBreite: 1200, maxHoehe: 1200, guete: 0.9 };

	class ProbeBild {
		width = 3000;
		height = 2000;
		onload = () => {};
		onerror = () => {};
		quelle = '';
		set src(wert) {
			this.quelle = wert;
			queueMicrotask(() => this.onload());
		}
		get src() {
			return this.quelle;
		}
	}

	beforeEach(() => {
		angefragt = [];
		URL.createObjectURL = vi.fn(() => 'blob:probe');
		URL.revokeObjectURL = vi.fn();
		vi.stubGlobal('Image', ProbeBild);
		vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
			/** @type {any} */ ({ drawImage: vi.fn() })
		);
		vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation((rueckruf, typ) => {
			angefragt.push(String(typ));
			const geliefert = typ === 'image/webp' && !kannWebp ? 'image/png' : String(typ);
			rueckruf(new Blob(['x'], { type: geliefert }));
		});
	});
	afterEach(() => {
		URL.createObjectURL = original.erzeuge;
		URL.revokeObjectURL = original.gibFrei;
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it('kodiert ein Foto ohne WebP-Kodierer als JPEG statt als PNG', async () => {
		kannWebp = false;
		const foto = new File(['x'], 'foto.jpg', { type: 'image/jpeg' });
		const blob = await verkleinereBild(foto, rahmen);
		expect(blob.type).toBe('image/jpeg');
		expect(angefragt).toEqual(['image/webp', 'image/jpeg']);
	});

	it('lässt ein PNG ohne WebP-Kodierer ein PNG, damit die Transparenz bleibt', async () => {
		kannWebp = false;
		const logo = new File(['x'], 'logo.png', { type: 'image/png' });
		const blob = await verkleinereBild(logo, rahmen);
		expect(blob.type).toBe('image/png');
		expect(angefragt).toEqual(['image/webp']);
	});

	it('bleibt mit WebP-Kodierer bei WebP', async () => {
		kannWebp = true;
		const foto = new File(['x'], 'foto.jpg', { type: 'image/jpeg' });
		const blob = await verkleinereBild(foto, rahmen);
		expect(blob.type).toBe('image/webp');
		expect(angefragt).toEqual(['image/webp']);
	});
});
