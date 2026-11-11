import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../utils/bildVerkleinern.js', () => ({ verkleinereBild: vi.fn() }));

import { verkleinereBild } from '../utils/bildVerkleinern.js';
import { leseKartenBild } from './kartenBild.js';

/** @param {number} bytes @param {string} [typ] @param {string} [name] */
const datei = (bytes, typ = 'image/png', name = 'foto.png') =>
	new File([new Uint8Array(bytes)], name, { type: typ });

// Bilder des Ausweis-Designs gingen unverkleinert als Base64 in das Design: Ein Foto vom Handy
// machte daraus mehrere Megabyte, die bei jeder Änderung neu an den Server gingen. War das
// Design über 5 MiB, hieß es nur „Ausweis-Design ist zu groß", ohne das Bild zu nennen.
describe('leseKartenBild', () => {
	// Mit Klammern: Was beforeEach zurückgibt, ruft Vitest nach dem Test als Aufräumfunktion.
	beforeEach(() => {
		vi.mocked(verkleinereBild).mockReset();
	});

	it('lässt ein kleines Bild, wie es ist', async () => {
		const url = await leseKartenBild(datei(1000));
		expect(url.startsWith('data:image/png;base64,')).toBe(true);
		expect(verkleinereBild).not.toHaveBeenCalled();
	});

	it('verkleinert ein großes Bild auf höchstens 1200 Pixel Kantenlänge', async () => {
		vi.mocked(verkleinereBild).mockResolvedValue(
			new Blob([new Uint8Array(50_000)], { type: 'image/webp' })
		);
		const url = await leseKartenBild(datei(3_000_000));
		expect(verkleinereBild).toHaveBeenCalledWith(expect.any(File), {
			maxBreite: 1200,
			maxHoehe: 1200,
			guete: 0.9
		});
		expect(url.startsWith('data:image/webp;base64,')).toBe(true);
		// 50.000 Bytes als Base64, nicht die drei Megabyte der Datei.
		expect(url.length).toBeLessThan(80_000);
	});

	it('weist ab, was auch verkleinert zu groß ist, und nennt die Datei', async () => {
		vi.mocked(verkleinereBild).mockResolvedValue(new Blob([new Uint8Array(2_100_000)]));
		await expect(leseKartenBild(datei(6_000_000, 'image/png', 'plakat.png'))).rejects.toThrow(
			'„plakat.png“ ist auch verkleinert zu groß (2,1 MB). Bitte kleiner zuschneiden oder als JPEG speichern.'
		);
	});

	it('nennt die Datei, die der Browser nicht als Bild lesen kann', async () => {
		vi.mocked(verkleinereBild).mockRejectedValue(new Error('Invalid image'));
		await expect(leseKartenBild(datei(400_000, 'image/png', 'kaputt.png'))).rejects.toThrow(
			'„kaputt.png“ ist kein Bild, das der Browser lesen kann.'
		);
	});

	it('lässt eine Vektorgrafik Vektor', async () => {
		const url = await leseKartenBild(datei(400_000, 'image/svg+xml', 'logo.svg'));
		expect(url.startsWith('data:image/svg+xml;base64,')).toBe(true);
		expect(verkleinereBild).not.toHaveBeenCalled();
	});
});
