import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { istNichtsGefunden, GELESENE_FORMATE, createBarcodeDetector } from './barcode_detector.js';

// Die Rückfall-Bibliothek, wie der Erkenner sie braucht: ein Objekt, das eine Bilddatei liest.
const { scanFileV2 } = vi.hoisted(() => ({ scanFileV2: vi.fn() }));
vi.mock('html5-qrcode', () => ({
	Html5Qrcode: class {
		/** @param {File} datei @param {boolean} zeigen */
		scanFileV2(datei, zeigen) {
			return scanFileV2(datei, zeigen);
		}
	},
	Html5QrcodeSupportedFormats: {}
}));

describe('Rückfall-Erkenner: was ist ein Fehler, was ist der Normalfall', () => {
	// Die Bibliothek meldet beides als Fehler. Fällt die Unterscheidung weg, ist die Wahl
	// zwischen zwei Schäden: entweder zehn Meldungen je Sekunde bei einer leeren Linse,
	// oder ein Browser, in dem die Erkennung gar nicht arbeitet, sieht aus wie eine Kamera
	// ohne Buch davor. Genau das war am 17.09.2026 der Fall.
	it('„kein Code im Bild" ist kein Fehler', () => {
		for (const text of [
			'QR code parse error, error = NotFoundException: No MultiFormat Readers were able to detect the code.',
			'D: No MultiFormat Readers were able to detect the code.',
			'NotFoundException'
		]) {
			expect(istNichtsGefunden(text), text).toBe(true);
			expect(istNichtsGefunden(new Error(text)), 'als Error: ' + text).toBe(true);
		}
	});

	it('alles andere ist ein Grund, warum die Erkennung nicht arbeitet', () => {
		for (const text of [
			'TypeError: File is not a constructor',
			'Der Browser liefert kein Bild aus der Kamera (toBlob).',
			'Cannot start file scan - ongoing camera scan'
		]) {
			expect(istNichtsGefunden(text), text).toBe(false);
			expect(istNichtsGefunden(new Error(text)), 'als Error: ' + text).toBe(false);
		}
		expect(istNichtsGefunden(undefined)).toBe(false);
		expect(istNichtsGefunden(null)).toBe(false);
		expect(istNichtsGefunden(123)).toBe(false);
		expect(istNichtsGefunden({})).toBe(false);
	});
});

describe('Die Formatliste der Kamera', () => {
	// Was die Anwendung druckt, muss sie lesen können — Code 128 seit dem 17.09.2026,
	// Code 39 aus der Zeit davor, QR als zweite Form aus Designer und Etikettendruck.
	// Die EAN-Arten sind die Littera-Altetiketten und die Verlags-Barcodes auf den Büchern.
	it('führt jede Form, die an dieser Schule vorkommt', () => {
		for (const format of ['code_128', 'code_39', 'qr_code', 'ean_13']) {
			expect(GELESENE_FORMATE, format).toContain(format);
		}
	});
});

// jsdom hat keinen eingebauten Erkenner, createBarcodeDetector fällt also auf die Bibliothek
// zurück — der Weg von Safari auf dem iPhone.
describe('Rückfall-Erkenner: was detect() aus einem Bild macht', () => {
	const bild = /** @type {any} */ ({ width: 4, height: 4 });
	/** @type {Blob | null} */
	let ausDerKamera;

	beforeEach(() => {
		scanFileV2.mockReset();
		ausDerKamera = new Blob(['bild'], { type: 'image/jpeg' });
		// jsdom zeichnet nicht: Die Leinwand liefert keinen Kontext, das Bild kommt aus der Attrappe.
		vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
		vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation((fertig) =>
			fertig(ausDerKamera)
		);
	});
	afterEach(() => {
		vi.restoreAllMocks();
	});

	async function erkenner() {
		const gefunden = await createBarcodeDetector();
		expect(gefunden?.name).toBe('zxing-fallback');
		return /** @type {NonNullable<typeof gefunden>} */ (gefunden).detector;
	}

	it('liefert den gelesenen Code', async () => {
		scanFileV2.mockResolvedValue({ decodedText: 'B-10234' });

		expect(await (await erkenner()).detect(bild)).toEqual([{ rawValue: 'B-10234' }]);
	});

	it('liefert eine leere Liste, wenn im Bild kein Code steht', async () => {
		scanFileV2.mockRejectedValue(
			'QR code parse error, error = NotFoundException: No MultiFormat Readers were able to detect the code.'
		);

		expect(await (await erkenner()).detect(bild)).toEqual([]);
	});

	it('reicht jeden anderen Fehler der Bibliothek als Error nach oben', async () => {
		const detektor = await erkenner();

		scanFileV2.mockRejectedValue('Cannot start file scan - ongoing camera scan');
		const ausText = await detektor.detect(bild).catch((/** @type {unknown} */ e) => e);
		expect(ausText).toBeInstanceOf(Error);
		expect(/** @type {Error} */ (ausText).message).toBe(
			'Cannot start file scan - ongoing camera scan'
		);

		const eigener = new TypeError('File is not a constructor');
		scanFileV2.mockRejectedValue(eigener);
		await expect(detektor.detect(bild)).rejects.toBe(eigener);
	});

	it('meldet es, wenn der Browser kein Bild aus der Kamera liefert', async () => {
		ausDerKamera = null;

		await expect((await erkenner()).detect(bild)).rejects.toThrow(
			'Der Browser liefert kein Bild aus der Kamera (toBlob).'
		);
		expect(scanFileV2).not.toHaveBeenCalled();
	});
});
