import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Das Schließen-Symbol der Buchmaske verwirft, was nicht gespeichert ist. Vorher fragt es, wie
// Material 3 es für den Full-screen dialog verlangt („Dismissing"); an einer Maske, an der
// niemand getippt hat, fragt es nicht (docs/OFFEN.md 5.75).
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Verwerfen ${s}`;

/** @param {import('@playwright/test').Page} page */
async function oeffneTitel(page) {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL);
	await page.getByText(TITEL).first().click();
	await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
}

/** @param {import('@playwright/test').Page} page */
const schliessen = (page) => page.getByRole('button', { name: 'Schließen', exact: true });
/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });

test.describe.serial('Buchmaske: Schließen mit Ungespeichertem', () => {
	test.beforeAll(() => {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, autor, verlag)
				VALUES ('${TITEL}', 'Ende, Michael', 'Thienemann')
				RETURNING id
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-vw-${s}', true FROM t;
		`);
	});
	test.afterAll(() => {
		seedSQL(`DELETE FROM buecher_titel WHERE titel = '${TITEL}';`);
	});

	test('fragt vor dem Verwerfen; „Bleiben" lässt die Eingabe stehen', async ({ page }) => {
		await oeffneTitel(page);
		await page.getByLabel('Verlag').fill('Cornelsen');
		await schliessen(page).click();

		await expect(rueckfrage(page)).toBeVisible();
		await expect(
			rueckfrage(page).getByRole('button', { name: 'Bleiben' }),
			'Enter verwirft nichts'
		).toBeFocused();
		await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
		await expect(rueckfrage(page)).toHaveCount(0);
		await expect(page.getByLabel('Verlag')).toHaveValue('Cornelsen');

		await schliessen(page).click();
		await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toHaveCount(0);
		expect(querySQL(`SELECT verlag FROM buecher_titel WHERE titel = '${TITEL}'`)).toBe(
			'Thienemann'
		);
	});

	test('ohne Eingabe schließt die Maske ohne Frage, auch „Neues Buch"', async ({ page }) => {
		await oeffneTitel(page);
		await schliessen(page).click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toHaveCount(0);

		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
		await schliessen(page).click();
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toHaveCount(0);
		await expect(rueckfrage(page)).toHaveCount(0);
	});
});
