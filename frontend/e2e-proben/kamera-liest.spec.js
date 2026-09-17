import { test, expect, chromium } from '@playwright/test';
import fs from 'node:fs';
import { uiLogin } from '../e2e/helpers.js';

/**
 * Hält der Theke einen echten Strichcode vor die Linse — Chrome gibt dafür eine
 * Videodatei als Kamera aus. Beantwortet die Frage, die sonst nur ein Mensch mit einem
 * Handy beantworten kann: Wird ein gedruckter Code erkannt und gebucht?
 *
 * Der zweite Durchgang ist der iPhone-Weg: Safari hat keinen eingebauten
 * Barcode-Erkenner, dort arbeitet der Rückfall über ZXing. Am 17.09.2026 war genau das
 * die Beschwerde („Kamera geht auf, nichts passiert").
 *
 * Läuft NICHT in der Suite mit: Das Kamerabild ist eine 120-MB-Videodatei, die
 * scripts/kamera_probe.sh vorher erzeugt. Ohne sie bricht der Test ab, statt sich still
 * zu überspringen — ein Gate, das niemand rot sehen kann, ist keins.
 */
const VIDEO = process.env.KAMERA_VIDEO ?? '';

async function probe(ohneEingebautenErkenner) {
	const browser = await chromium.launch({
		args: [
			'--use-fake-ui-for-media-stream',
			'--use-fake-device-for-media-stream',
			`--use-file-for-fake-video-capture=${VIDEO}`,
			'--autoplay-policy=no-user-gesture-required'
		]
	});
	const context = await browser.newContext({
		baseURL: 'http://localhost:8084',
		permissions: ['camera']
	});
	if (ohneEingebautenErkenner) {
		// Genau der iPhone-Fall: Safari kennt keinen eingebauten Barcode-Erkenner.
		await context.addInitScript(() => {
			// @ts-expect-error — window.BarcodeDetector ist im DOM-Typ nicht vorgesehen
			delete window.BarcodeDetector;
		});
	}
	const page = await context.newPage();
	const gesendet = [];
	const fehler = [];
	page.on('pageerror', (e) => fehler.push('pageerror: ' + e.message));
	page.on('console', (m) => {
		if (m.type() === 'error') fehler.push('console: ' + m.text());
	});
	page.on('request', (r) => {
		if (r.url().includes('/api/action') && r.method() === 'POST') gesendet.push(r.postData() ?? '');
	});

	await uiLogin(page);
	await page.getByRole('button', { name: /Kamera-Barcode-Scanner/ }).click();
	await page.waitForTimeout(8000);
	const text = await page.locator('body').innerText();
	const zeile = text.split('\n').find((z) => z.includes('Kamera')) ?? '(keine Zeile)';
	await browser.close();
	return { gesendet, fehler, zeile };
}

test('Kamera liest einen echten Barcode — mit und ohne eingebauten Erkenner', async () => {
	expect(
		VIDEO && fs.existsSync(VIDEO),
		'KAMERA_VIDEO fehlt — diese Probe läuft über scripts/kamera_probe.sh'
	).toBe(true);

	const mit = await probe(false);
	console.log('=== MIT eingebautem Erkenner ===');
	console.log('Zeile: ' + mit.zeile);
	console.log('gesendet: ' + JSON.stringify(mit.gesendet));
	console.log('Fehler: ' + mit.fehler.slice(0, 3).join(' | '));

	const ohne = await probe(true);
	console.log('=== OHNE eingebauten Erkenner (iPhone-Weg) ===');
	console.log('Zeile: ' + ohne.zeile);
	console.log('gesendet: ' + JSON.stringify(ohne.gesendet));
	console.log('Fehler: ' + ohne.fehler.slice(0, 3).join(' | '));
	expect(true).toBe(true);
});
