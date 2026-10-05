import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Titel-Verwaltung: Es gilt die zuletzt angeforderte Maske. Wer einen Titel anklickt und
// gleich darauf einen anderen (oder „Neues Buch"), bekam die Antwort zum ersten über die
// offene Maske gelegt, sobald sie später ankam: Der Titel in der Maske wechselte, und was
// dort getippt war, ging verloren.
const s = uniqueSuffix().slice(0, 6);
const ALT = `Oeffnenalt ${s}`;
const NEU = `Oeffnenneu ${s}`;

/**
 * Hält die Antwort zum Einzelabruf eines Titels zurück, bis `frei()` gerufen wird.
 * @param {import('@playwright/test').Page} page @param {string} id
 */
async function halteTitel(page, id) {
	/** @type {(wert?: unknown) => void} */
	let frei = () => {};
	const freigabe = new Promise((fertig) => (frei = fertig));
	await page.route(new RegExp(`/api/books/${id}$`), async (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		await freigabe;
		await route.continue();
	});
	return frei;
}

/** @param {import('@playwright/test').Page} page */
async function zurTabelle(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(s);
	await expect(page.getByText(ALT).first()).toBeVisible();
	await expect(page.getByText(NEU).first()).toBeVisible();
}

test.describe('Titel-Verwaltung: die jüngste Öffnung gilt', () => {
	test.beforeAll(() => {
		seedSQL(`
			WITH t AS (INSERT INTO buecher_titel (titel, autor, signatur, medientyp) VALUES
				('${ALT}', 'E2E', 'Alt 1', 'Buch'), ('${NEU}', 'E2E', 'Neu 1', 'Buch') RETURNING id, titel)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-OR-' || left(titel, 10) || '-${s}', true FROM t;`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel IN ('${ALT}', '${NEU}'));
			DELETE FROM buecher_titel WHERE titel IN ('${ALT}', '${NEU}');`);
	});

	test('die späte Antwort zum ersten Titel lässt die Maske des zweiten stehen', async ({
		page
	}) => {
		const frei = await halteTitel(
			page,
			querySQL(`SELECT id FROM buecher_titel WHERE titel = '${ALT}'`)
		);
		await zurTabelle(page);

		await page.getByText(ALT).first().click();
		await page.getByText(NEU).first().click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(NEU);
		await page.locator('#buch-signatur').fill('Getippt 9');

		frei();
		await page.waitForTimeout(600);
		await expect(page.locator('#buch-titel'), 'der Titel in der Maske').toHaveValue(NEU);
		await expect(page.locator('#buch-signatur'), 'das Getippte').toHaveValue('Getippt 9');
	});

	test('die späte Antwort legt sich nicht über die Maske „Neues Buch"', async ({ page }) => {
		const frei = await halteTitel(
			page,
			querySQL(`SELECT id FROM buecher_titel WHERE titel = '${ALT}'`)
		);
		await zurTabelle(page);

		await page.getByText(ALT).first().click();
		await page.getByRole('button', { name: 'Neues Buch' }).click();
		await expect(page.locator('#buch-titel')).toHaveValue('');
		await page.locator('#buch-titel').fill('Frisch getippt');

		frei();
		await page.waitForTimeout(600);
		await expect(page.locator('#buch-titel'), 'der Titel in der Maske').toHaveValue(
			'Frisch getippt'
		);
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toHaveCount(0);
	});
});
