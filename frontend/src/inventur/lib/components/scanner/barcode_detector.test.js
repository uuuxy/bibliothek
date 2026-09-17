import { describe, it, expect } from 'vitest';
import { istNichtsGefunden, GELESENE_FORMATE } from './barcode_detector.js';

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
		}
		expect(istNichtsGefunden(undefined)).toBe(false);
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
