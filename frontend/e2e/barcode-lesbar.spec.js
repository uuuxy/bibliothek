import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';
import { GELESENE_FORMATE } from '../src/inventur/lib/components/scanner/barcode_detector.js';

/**
 * Was wir drucken, muss gelesen werden — und dasselbe heißen.
 *
 * Am 17.09.2026 war beides verletzt, in zwei voneinander unabhängigen Fehlern:
 *
 *  1. `GenerateBarcodePNG` druckte Code 39 MIT Prüfzeichen (`code39.Encode(x, true, true)`).
 *     Das Prüfzeichen steht in den Strichcode-Daten, und jeder Leser gibt es als Teil der
 *     Nummer zurück: Auf dem Ausweis stand „A-10003", gescannt wurde „A-100037". Gemessen
 *     mit zwei unabhängigen Erkennern und am fertigen PDF des Druck-Centers.
 *  2. Die Formatliste der Kamera kannte Code 39 gar nicht — unser eigener Aufdruck stand
 *     nicht darauf.
 *
 * Deshalb prüft dieses Gate die ganze Kette am ECHTEN Erzeugnis: Bild aus `/api/barcode`,
 * gelesen mit genau der Formatliste, die die Anwendung ihren Kameras gibt. Ein Gate an der
 * Formatliste allein wäre blind für Fehler 1, eines am Generator allein blind für Fehler 2.
 */
const NUMMERN = ['B-10001', 'A-10003', 'LMF-4711'];

test('Der gedruckte Barcode liest sich als das, was auf dem Papier steht', async ({ page }) => {
	await uiLogin(page);

	/** @type {Record<string, string>} */
	const bilder = {};
	for (const nummer of NUMMERN) {
		const antwort = await page.request.get(`/api/barcode?content=${encodeURIComponent(nummer)}`);
		expect(antwort.status(), `/api/barcode für ${nummer}`).toBe(200);
		bilder[nummer] = 'data:image/png;base64,' + (await antwort.body()).toString('base64');
	}

	// Gelesen wird auf einer leeren Seite: Die Anwendung selbst verbietet per CSP jedes
	// eingefügte Skript — zu Recht. Das Bild kommt trotzdem aus ihrer eigenen Tür.
	const leer = await page.context().newPage();
	await leer.addScriptTag({ path: 'node_modules/html5-qrcode/html5-qrcode.min.js' });

	const gelesen = await leer.evaluate(
		async ({ bilder, formate }) => {
			const F = /** @type {any} */ (window).Html5QrcodeSupportedFormats;
			// Die Namen des eingebauten Erkenners auf die Zahlen der Rückfall-Bibliothek.
			const nachZahl = {
				code_39: F.CODE_39,
				code_128: F.CODE_128,
				ean_13: F.EAN_13,
				ean_8: F.EAN_8,
				upc_a: F.UPC_A,
				upc_e: F.UPC_E
			};
			const erlaubt = formate
				.map((/** @type {string} */ n) => nachZahl[n])
				.filter((z) => z !== undefined);

			/** @type {Record<string, string>} */
			const ergebnis = {};
			let i = 0;
			for (const [nummer, quelle] of Object.entries(bilder)) {
				const blob = await (await fetch(quelle)).blob();
				const datei = new File([blob], 'b.png', { type: 'image/png' });
				const knoten = document.createElement('div');
				knoten.id = 'lesen-' + i++;
				document.body.appendChild(knoten);
				const scanner = new /** @type {any} */ (window).Html5Qrcode(knoten.id, {
					formatsToSupport: erlaubt,
					verbose: false
				});
				try {
					const r = await scanner.scanFileV2(datei, false);
					ergebnis[nummer] = r.decodedText;
				} catch {
					ergebnis[nummer] = '(nichts erkannt)';
				}
			}
			return ergebnis;
		},
		{ bilder, formate: GELESENE_FORMATE }
	);
	await leer.close();

	for (const nummer of NUMMERN) {
		expect(
			gelesen[nummer],
			`Aufdruck „${nummer}" wird als „${gelesen[nummer]}" gelesen — Aufdruck und Scanwert müssen ` +
				`gleich sein, sonst sucht der Server eine Nummer, die es nicht gibt`
		).toBe(nummer);
	}
});
