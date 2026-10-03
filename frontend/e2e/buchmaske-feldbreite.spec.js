// Gate: In der Buchmaske enden die Textfelder bei 44rem, auch im breiten Fenster; die Maske
// selbst bleibt breit.
//
// Material 3, Text fields, Guidelines: „Text fields shouldn’t span the full width of a large
// screen.“ Gemessen wird im Browser, weil die Breite erst beim Layout entsteht.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

for (const fenster of [
	{ width: 1710, height: 952 },
	{ width: 1366, height: 768 }
]) {
	test(`Buchmaske im Fenster von ${fenster.width} px: Felder höchstens 704 px breit`, async ({
		page
	}) => {
		await page.setViewportSize(fenster);
		await uiLogin(page);
		await gehZu(page, '/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();

		for (const id of ['buch-titel', 'buch-autor']) {
			const feld = await page.locator(`#${id}`).boundingBox();
			expect(feld, `#${id} hat keine Fläche`).not.toBeNull();
			expect(feld?.width, `#${id} ist breiter als 44rem`).toBeLessThanOrEqual(704);
			// Gegenprobe: Das Feld ist nicht zusammengefallen.
			expect(feld?.width, `#${id} ist schmaler als erwartet`).toBeGreaterThan(600);
		}

		// Begrenzt sind die Felder, nicht die Seite: Der Kopf der Maske reicht weiter über die Fläche.
		const kopf = await page
			.getByRole('heading', { name: 'Neues Buch' })
			.locator('xpath=..')
			.boundingBox();
		expect(kopf?.width, 'der Kopf der Maske').toBeGreaterThan(fenster.width - 400);
	});
}
