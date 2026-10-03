import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, oeffneSchuelerProfil, uniqueSuffix } from './helpers.js';

// Die Aufnahme des Passbilds geht aus der Leserakte auf, auch an der Theke. Der Handscanner
// tippt blind und endet mit Enter: Läge der erste Fokus auf „Foto aufnehmen", nähme ein
// gescanntes Buch ein Foto auf und ersetzte das vorhandene. Der erste Fokus liegt deshalb auf
// „Schließen" im Kopf des Dialogs.

/**
 * Ein Kamerabild ohne Kamera: der Strom einer Zeichenfläche. Sie zeichnet weiter, weil der
 * Strom nur bei einer Änderung ein neues Bild liefert.
 * @param {import('@playwright/test').Page} page
 */
async function kameraAttrappe(page) {
	await page.addInitScript(() => {
		const flaeche = document.createElement('canvas');
		flaeche.width = 1280;
		flaeche.height = 720;
		const ctx = /** @type {CanvasRenderingContext2D} */ (flaeche.getContext('2d'));
		let bild = 0;
		setInterval(() => {
			ctx.fillStyle = bild++ % 2 ? '#35506b' : '#36516c';
			ctx.fillRect(0, 0, 1280, 720);
		}, 100);
		Object.defineProperty(navigator, 'mediaDevices', {
			configurable: true,
			value: { getUserMedia: async () => flaeche.captureStream(10) }
		});
	});
}

/**
 * Legt einen Leser an, öffnet seine Akte und darin die Aufnahme. Der Upload wird abgefangen
 * und gezählt; gespeichert wird nichts.
 * @param {import('@playwright/test').Page} page
 */
async function oeffneAufnahme(page) {
	const s = uniqueSuffix();
	const vorname = `Webcam${s}`;
	seedSQL(`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	         VALUES ('E2E-WEB-${s}', '${vorname}', 'Probe', '07A', 2030);`);
	const aufraeumen = () => seedSQL(`DELETE FROM schueler WHERE barcode_id = 'E2E-WEB-${s}';`);

	/** @type {string[]} */
	const uploads = [];
	await page.route('**/api/schueler/*/photo', async (route) => {
		if (route.request().method() !== 'POST') return route.continue();
		uploads.push(route.request().url());
		await route.fulfill({ status: 200, contentType: 'application/json', body: '{"url":""}' });
	});

	await kameraAttrappe(page);
	await uiLogin(page);
	await oeffneSchuelerProfil(page, vorname);
	await page.getByRole('button', { name: 'Passbild mit Webcam aufnehmen' }).click();
	const dialog = page.getByRole('dialog', { name: 'Passbild aufnehmen' });
	await expect(dialog.getByRole('button', { name: 'Foto aufnehmen' })).toBeVisible();
	return { dialog, uploads, aufraeumen };
}

test('Passbild: ein Scan bei offener Aufnahme nimmt kein Foto auf', async ({ page }) => {
	const { dialog, uploads, aufraeumen } = await oeffneAufnahme(page);
	try {
		await expect(dialog.getByRole('button', { name: 'Schließen' })).toBeFocused();

		// Blind getippt wie vom Handscanner, ohne Klick ins Fenster.
		await page.keyboard.type('B-12345', { delay: 20 });
		await page.keyboard.press('Enter');

		await expect(dialog, 'der Scan schließt die Aufnahme').toBeHidden();
		expect(uploads, 'der Scan darf kein Foto hochladen').toEqual([]);
	} finally {
		aufraeumen();
	}
});

// Gegenprobe am Horcher: Mit dem Fokus auf der Aufnahme löst dasselbe Enter den Upload aus.
// Ohne sie bliebe der Test oben auch dann grün, wenn er Uploads gar nicht sähe.
test('Passbild: mit dem Fokus auf der Aufnahme löst Enter den Upload aus', async ({ page }) => {
	const { dialog, uploads, aufraeumen } = await oeffneAufnahme(page);
	try {
		await dialog.getByRole('button', { name: 'Foto aufnehmen' }).focus();
		await page.keyboard.press('Enter');
		await expect.poll(() => uploads.length, 'der Upload wird gesehen').toBe(1);
	} finally {
		aufraeumen();
	}
});
