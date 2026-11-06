import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, seedCoverDatei, uniqueSuffix } from './helpers.js';

// Wer im Vorschlagsmenü der Signatur auf einen Eintrag zeigt, sieht zwei vorhandene Cover der
// Titel darunter. Die Fläche geht neben dem Eintrag auf wie ein Untermenü und verdeckt die Liste
// nicht. Im Regal steht vorn ein Titel ohne Cover: Ohne den Filter der Abfrage stünde an seiner
// Stelle ein leerer Platz.
const s = uniqueSuffix().slice(0, 6);
const SIGNATUR = `Cov${s}`;

test.describe('Signatur: Cover beim Zeigen auf einen Vorschlag', () => {
	test.beforeAll(() => {
		const bild1 = seedCoverDatei(`e2e-signatur-cover-${s}-1`);
		const bild2 = seedCoverDatei(`e2e-signatur-cover-${s}-2`);
		seedSQL(`
			INSERT INTO buecher_titel (titel, signatur, cover_url) VALUES
				('A ohne Cover ${s}', '${SIGNATUR}', NULL),
				('B mit Cover ${s}', '${SIGNATUR}', '${bild1}'),
				('C mit Cover ${s}', '${SIGNATUR}', '${bild2}');
		`);
	});
	test.afterAll(() => {
		seedSQL(`DELETE FROM buecher_titel WHERE signatur = '${SIGNATUR}';`);
	});

	test('zeigt zwei Cover neben dem Eintrag, ohne die Liste zu verdecken', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('button', { name: 'Titel anlegen' }).first().click();
		await page.locator('#buch-signatur').fill(SIGNATUR);

		const eintrag = page.getByRole('option', { name: new RegExp(`^${SIGNATUR}`) });
		await eintrag.hover();
		const vorschau = page.locator('[data-menue-vorschau]');
		const bilder = vorschau.locator('img');
		await expect(bilder).toHaveCount(2);
		// Geladen heißt breiter als das 1×1-GIF, mit dem der Proxy ein fehlendes Bild beantwortet.
		for (const bild of await bilder.all()) {
			await expect
				.poll(() => bild.evaluate((b) => /** @type {HTMLImageElement} */ (b).naturalWidth))
				.toBeGreaterThan(1);
		}

		const liste = await page.getByRole('listbox').boundingBox();
		const flaeche = await vorschau.boundingBox();
		const zeile = await eintrag.boundingBox();
		if (!liste || !flaeche || !zeile) throw new Error('Liste, Vorschau oder Eintrag ohne Maße');
		const daneben =
			flaeche.x >= liste.x + liste.width - 1 || flaeche.x + flaeche.width <= liste.x + 1;
		expect(
			daneben,
			`Vorschau ${JSON.stringify(flaeche)} liegt über der Liste ${JSON.stringify(liste)}`
		).toBe(true);
		// Oben bündig mit dem Eintrag, außer der untere Fensterrand schiebt sie hoch (8 px Abstand).
		const fensterHoehe = page.viewportSize()?.height ?? 0;
		const oben = Math.min(zeile.y, fensterHoehe - 8 - flaeche.height);
		expect(Math.abs(flaeche.y - oben), 'oben bündig mit dem Eintrag').toBeLessThanOrEqual(1);

		await page.keyboard.press('Escape');
		await expect(vorschau).toHaveCount(0);
	});
});
