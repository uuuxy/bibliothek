import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

// Scheitert der Abruf des Ausweis-Designs, käme ein Ausweis mit den Standardwerten aus dem
// Drucker. Akte und Leserdatei drucken dann nicht und sagen es beim Druck; die Theke bleibt
// still, solange dort nur ein Leser geladen wird.

const DRUCKEN_GESPERRT = 'Nicht gedruckt: Das Ausweis-Design ist nicht geladen.';
const SEITE_OHNE_DESIGN =
	'Ausweis-Design nicht geladen: Ausweise lassen sich gerade nicht drucken.';

/** Zählt die Aufrufe des Druckdialogs, statt ihn zu öffnen.
 * @param {import('@playwright/test').Page} page */
async function druckdialogZaehlen(page) {
	await page.addInitScript(() => {
		/** @type {any} */ (window).druckaufrufe = 0;
		window.print = () => {
			/** @type {any} */ (window).druckaufrufe += 1;
		};
	});
}

/** @param {import('@playwright/test').Page} page */
const druckaufrufe = (page) => page.evaluate(() => /** @type {any} */ (window).druckaufrufe);

/**
 * Lässt den Abruf des Designs scheitern, bis die zurückgegebene Funktion gerufen wird.
 * @param {import('@playwright/test').Page} page
 */
async function designFaelltAus(page) {
	/** @param {import('@playwright/test').Route} route */
	const scheitern = (route) =>
		route.request().method() === 'GET'
			? route.fulfill({
					status: 500,
					contentType: 'application/json',
					body: '{"error":"Lesefehler"}'
				})
			: route.continue();
	await page.route('**/api/ausweis-layout', scheitern);
	return () => page.unroute('**/api/ausweis-layout', scheitern);
}

/** @param {import('@playwright/test').Page} page @param {string} text */
const meldung = (page, text) => page.getByRole('alert').filter({ hasText: text });

test.describe('Ausweis-Design nicht geladen', () => {
	const s = uniqueSuffix().toUpperCase();
	const ausweis = `S-AD${s}`;
	const vorname = `Design${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('${ausweis}', '${vorname}', 'Ladeprobe', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
		`);
	});

	test.afterAll(() => {
		seedSQL(`DELETE FROM schueler WHERE barcode_id = '${ausweis}';`);
	});

	test('Theke: Der Leser lädt ohne Meldung, „Ausweis drucken" druckt nicht und sagt es', async ({
		page
	}) => {
		await druckdialogZaehlen(page);
		const wiederDa = await designFaelltAus(page);
		await uiLogin(page);

		const abruf = page.waitForResponse('**/api/ausweis-layout');
		await scanneWieScanner(page, ausweis);
		const knopf = page.getByRole('button', { name: 'Ausweis drucken', exact: true });
		await expect(knopf).toBeVisible();
		await abruf;
		await expect(page.getByRole('alert').filter({ hasText: /Ausweis-Design/ })).toHaveCount(0);

		await knopf.click();
		await expect(meldung(page, DRUCKEN_GESPERRT)).toBeVisible();
		expect(await druckaufrufe(page)).toBe(0);

		// Sobald der Server das Design wieder liefert, druckt derselbe Knopf.
		await wiederDa();
		await knopf.click();
		await expect.poll(() => druckaufrufe(page)).toBe(1);
	});

	test('Leserdatei: Die Seite sagt es, und „Ausweise drucken" druckt nicht', async ({ page }) => {
		await druckdialogZaehlen(page);
		const wiederDa = await designFaelltAus(page);
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await expect(meldung(page, SEITE_OHNE_DESIGN)).toBeVisible();

		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await expect(page.locator('tbody tr').filter({ hasText: vorname })).toHaveCount(1);
		await page.getByRole('checkbox', { name: /Alle angezeigten Leser/ }).check();
		const knopf = page
			.getByRole('region', { name: /Aktionen für die markierten/ })
			.getByRole('button', { name: 'Ausweise drucken' });

		await knopf.click();
		await expect(meldung(page, DRUCKEN_GESPERRT)).toBeVisible();
		expect(await druckaufrufe(page)).toBe(0);

		await wiederDa();
		await knopf.click();
		await expect.poll(() => druckaufrufe(page)).toBe(1);
	});
});
