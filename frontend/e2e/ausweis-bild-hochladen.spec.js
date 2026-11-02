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
	await page.route('**/api/ausweis-layout', (route) =>
		route.request().method() === 'PUT'
			? route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
			: route.continue()
	);
	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByRole('tab', { name: 'Schülerausweise' }).click();
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
