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

test('Das Dateifeld ist nach dem Hochladen leer, ohne Fehler auf der Seite', async ({ page }) => {
	// Das Design gilt für alle Arbeitsplätze; aus dieser Probe wird nichts gespeichert.
	await page.route('**/api/ausweis-layout', (route) =>
		route.request().method() === 'PUT'
			? route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
			: route.continue()
	);
	/** @type {string[]} */
	const fehler = [];
	page.on('pageerror', (f) => fehler.push(f.message));

	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByRole('tab', { name: 'Schülerausweise' }).click();
	await expect(page.getByRole('button', { name: /Testdruck Vorderseite/ })).toBeVisible();

	const feld = page.locator('input[type="file"][multiple]');
	await feld.setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: PIXEL });

	await expect(feld).toHaveValue('');
	expect(fehler).toEqual([]);
});
