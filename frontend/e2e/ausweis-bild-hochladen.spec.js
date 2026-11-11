import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// Nach dem Einlesen ist das Dateifeld im Ausweis-Designer wieder leer. Bliebe die Datei darin
// stehen, meldete der Browser beim nächsten Wählen derselben Datei keine Änderung, und ein
// gelöschtes Logo käme nicht mehr auf die Karte. Das Einlesen ist asynchron; nach dem await ist
// currentTarget des Ereignisses null.

// Ein Bild aus einem Pixel, als PNG.
const PIXEL = Buffer.from(
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
	'base64'
);

test.beforeEach(async ({ page }) => {
	// Das Design gilt für alle Arbeitsplätze; aus dieser Probe wird nichts gespeichert.
	// Mit Stern: Das Speichern trägt den Änderungsstand als ?stand=….
	await page.route('**/api/ausweis-layout*', (route) =>
		route.request().method() === 'PUT'
			? route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
			: route.continue()
	);
	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByRole('tab', { name: 'Ausweis-Design' }).click();
	await expect(page.getByRole('button', { name: /Testdruck Vorderseite/ })).toBeVisible();
	// Bis das Design geladen ist, ist der Designer inert, und die Antwort überschriebe ein Bild.
	// setInputFiles prüft das nicht; der Probeklick wartet, bis der Designer Eingaben annimmt.
	await page.getByRole('button', { name: '+ Bild(er)' }).click({ trial: true });
});

test('Das Dateifeld ist nach dem Hochladen leer, ohne Fehler auf der Seite', async ({ page }) => {
	/** @type {string[]} */
	const fehler = [];
	page.on('pageerror', (f) => fehler.push(f.message));

	const feld = page.locator('input[type="file"][multiple]');
	await feld.setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: PIXEL });

	await expect(feld).toHaveValue('');
	expect(fehler).toEqual([]);
});

// Dasselbe Feld bedient jedes gewählte Bild. Bliebe die Datei darin, käme sie beim zweiten Bild nicht an.
test('Auch das Bildfeld der Eigenschaften ist nach dem Wählen leer', async ({ page }) => {
	await page
		.locator('input[type="file"][multiple]')
		.setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: PIXEL });
	await page
		.locator('[data-ausweis-vorschau]')
		.first()
		.locator('img[src^="data:image/png"]')
		.last()
		.click({ force: true, position: { x: 3, y: 3 } });

	const feld = page.locator('input[type="file"]:not([multiple])');
	await feld.setInputFiles({ name: 'tausch.png', mimeType: 'image/png', buffer: PIXEL });

	await expect(feld).toHaveValue('');
});

// Ein Foto vom Handy ging unverkleinert als Base64 in das Design und mit jeder Änderung neu an
// den Server. Der Designer verkleinert es jetzt auf höchstens 1200 Pixel Kantenlänge
// (designer/kartenBild.js); jsdom zeichnet kein Bild, deshalb steht die Probe hier.
test('Ein großes Foto kommt verkleinert auf die Karte', async ({ page }) => {
	// Ein Verlauf mit leichtem Rauschen, 3000 × 2000 Pixel, als JPEG.
	const base64 = await page.evaluate(() => {
		const flaeche = document.createElement('canvas');
		flaeche.width = 3000;
		flaeche.height = 2000;
		const stift = /** @type {CanvasRenderingContext2D} */ (flaeche.getContext('2d'));
		const bild = stift.createImageData(3000, 2000);
		for (let i = 0; i < bild.data.length; i += 4) {
			const x = (i / 4) % 3000;
			const rauschen = Math.random() * 24;
			bild.data[i] = (x / 3000) * 230 + rauschen;
			bild.data[i + 1] = 120 + rauschen;
			bild.data[i + 2] = 200 - (x / 3000) * 150 + rauschen;
			bild.data[i + 3] = 255;
		}
		stift.putImageData(bild, 0, 0);
		return flaeche.toDataURL('image/jpeg', 0.9).split(',')[1];
	});
	const buffer = Buffer.from(base64, 'base64');
	expect(buffer.length, 'die Probe braucht eine Datei über der Schwelle').toBeGreaterThan(300_000);

	const bilder = page.locator('[data-ausweis-vorschau]').first().locator('img[src^="data:image"]');
	const vorher = await bilder.count();
	await page
		.locator('input[type="file"][multiple]')
		.setInputFiles({ name: 'foto.jpg', mimeType: 'image/jpeg', buffer });
	await expect(bilder).toHaveCount(vorher + 1);

	const masse = await bilder.last().evaluate(
		(/** @type {HTMLImageElement} */ img) =>
			new Promise((fertig) => {
				const lies = () =>
					fertig({ breite: img.naturalWidth, hoehe: img.naturalHeight, laenge: img.src.length });
				if (img.complete && img.naturalWidth > 0) lies();
				else img.onload = lies;
			})
	);
	expect(masse).toMatchObject({ breite: 1200, hoehe: 800 });
	expect(masse.laenge).toBeLessThan(buffer.length);
});
