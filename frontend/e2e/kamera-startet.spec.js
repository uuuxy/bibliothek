import { test, expect, chromium } from '@playwright/test';
import { uiLogin } from './helpers.js';

/**
 * Die Kamera an der Theke geht auf und läuft.
 *
 * Am 17.09.2026 tat sie das nicht mehr: Die Theke startete ihre Kamera an einer ZWEITEN
 * Stelle (`Omnibox.svelte` baute selbst ein `Html5Qrcode` gegen das DOM-Element
 * `camera-scan-region`), während das Bauteil daneben seine eigene Technik mitbrachte.
 * Als das Bauteil sein Element wechselte, zeigte der Aufruf ins Leere — die Theke
 * meldete „Kamera konnte nicht gestartet werden", und kein Test sah es.
 *
 * Deshalb misst dieses Gate am laufenden Stack mit einer künstlichen Kamera: Nach dem
 * Klick auf den Kamera-Knopf muss ein Video laufen UND die Zeile darüber muss sagen, dass
 * die Kamera aktiv ist. Ein Bild ohne Meldung wäre die halbe Wahrheit — und eine Meldung
 * ohne Bild die andere Hälfte.
 */
test('Die Kamera der Theke startet und meldet sich aktiv', async () => {
	const browser = await chromium.launch({
		args: [
			'--use-fake-ui-for-media-stream',
			'--use-fake-device-for-media-stream',
			'--autoplay-policy=no-user-gesture-required'
		]
	});
	try {
		const context = await browser.newContext({
			baseURL: process.env.E2E_BASE_URL || 'http://localhost:8084',
			permissions: ['camera']
		});
		const page = await context.newPage();
		/** @type {string[]} */
		const seitenfehler = [];
		page.on('pageerror', (e) => seitenfehler.push(e.message));

		await uiLogin(page);
		await page.getByRole('button', { name: /Kamera-Barcode-Scanner/ }).click();

		const video = page.locator('video');
		await expect(video, 'Nach dem Klick muss ein Video-Element stehen').toHaveCount(1);
		await expect(
			page.getByText('Kamera aktiv', { exact: false }),
			'Die Zeile über dem Bild muss den Zustand nennen — „läuft" sieht sonst aus wie „hängt"'
		).toBeVisible({ timeout: 10_000 });

		expect(seitenfehler, 'Kein Fehler beim Starten der Kamera').toEqual([]);
	} finally {
		await browser.close();
	}
});
