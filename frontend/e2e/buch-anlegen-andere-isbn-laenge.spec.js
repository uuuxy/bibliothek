import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';
import { isbnFormen } from '../src/lib/utils/isbnFormen.js';

// Dieselbe ISBN gibt es in zwei Längen. Ein Titel aus Littera trägt die zehnstellige vom
// Titelblatt, der Scanner liest vom Buch die dreizehnstellige; die Maske „Neues Buch" fand
// ihn darunter nicht, und das Buch stand danach zweimal im Katalog. Jetzt fragt sie, ob es
// dasselbe Buch ist. Die Antwort entscheidet: Unter der anderen Länge kann ein anderes Buch
// stehen, deshalb lehnt das Speichern nicht ab.
const s = uniqueSuffix().slice(0, 6);
const ean = (/** @type {number} */ versatz) =>
	isbnFormen(`3${String(Date.now() + versatz).slice(-8)}0`)[1];
const [EAN_SELBES, EAN_ANDERES] = [ean(0), ean(41)];
const [ZEHN_SELBES, ZEHN_ANDERES] = [isbnFormen(EAN_SELBES)[1], isbnFormen(EAN_ANDERES)[1]];
const titelAnzahl = (/** @type {string} */ isbn) =>
	querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${isbn}'`);

/** @param {import('@playwright/test').Page} page @param {string} isbn */
async function neuesBuchMitIsbn(page, isbn) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await page.locator('#buch-isbn').fill(isbn);
	await page.locator('#buch-isbn').press('Tab');
	return page.getByRole('dialog').filter({ hasText: 'Ist es dasselbe Buch?' });
}

test.describe('Neues Buch: die ISBN steht in der anderen Länge im Katalog', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, medientyp)
			VALUES ('Aus Littera ${s}', 'E2E', '${ZEHN_SELBES}', 'Lit 1', 'Buch'),
			       ('Fremdes Buch ${s}', 'E2E', '${ZEHN_ANDERES}', 'Lit 2', 'Buch');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-AL-' || isbn, true FROM buecher_titel WHERE isbn IN ('${ZEHN_SELBES}', '${ZEHN_ANDERES}');`);
	});
	test.afterAll(() => {
		const alle = `'${ZEHN_SELBES}','${ZEHN_ANDERES}','${EAN_SELBES}','${EAN_ANDERES}'`;
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN (${alle}));
			DELETE FROM buecher_titel WHERE isbn IN (${alle});`);
	});

	test('dasselbe Buch: die Frage nennt den Titel und öffnet ihn', async ({ page }) => {
		const frage = await neuesBuchMitIsbn(page, EAN_SELBES);
		await expect(frage).toContainText(`zehnstelliger Form (${ZEHN_SELBES})`);
		await expect(frage).toContainText(`Aus Littera ${s}`);

		await frage.getByRole('button', { name: 'Titel öffnen' }).click();

		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(`Aus Littera ${s}`);
		await expect(page.locator('#buch-isbn')).toHaveValue(ZEHN_SELBES);
		expect(titelAnzahl(EAN_SELBES)).toBe('0');
	});

	test('ein anderes Buch: die Maske bleibt, fragt nicht noch einmal und legt an', async ({
		page
	}) => {
		// Die Katalogdienste sind ersetzt: Zu einer erfundenen ISBN wissen sie nichts.
		await page.route('**/api/lookup/**', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ data: { title: `Anderes Buch ${s}`, author: 'Probe, Paula' } })
			})
		);
		const frage = await neuesBuchMitIsbn(page, EAN_ANDERES);
		await expect(frage).toContainText(`Fremdes Buch ${s}`);

		await frage.getByRole('button', { name: 'Anderes Buch' }).click();

		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(`Anderes Buch ${s}`);
		await page.locator('#buch-isbn').focus();
		await page.locator('#buch-signatur').fill('Lit 3');
		await expect(frage).toHaveCount(0);

		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect([titelAnzahl(EAN_ANDERES), titelAnzahl(ZEHN_ANDERES)]).toEqual(['1', '1']);
	});
});
