import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';
import { isbnFormen } from '../src/lib/utils/isbnFormen.js';

// Die zehnstellige ISBN vom Titelblatt und die dreizehnstellige vom Strichcode sind dieselbe
// Nummer (Migration 157). Der Katalog führt sie dreizehnstellig, auch wenn der Titel
// zehnstellig angelegt wurde. Die Maske „Neues Buch" findet ihn in beiden Längen und fragt wie
// bei jeder vergebenen ISBN „Vorhandenen Titel öffnen?". Eine zehnstellige Nummer mit falschem
// Prüfzeichen ist eine andere Nummer: Zu ihr fragt die Maske nicht, und das Speichern legt an.
const s = uniqueSuffix().slice(0, 6);
const ean = (/** @type {number} */ versatz) =>
	isbnFormen(`3${String(Date.now() + versatz).slice(-8)}0`)[1];
const [EAN_SELBES, EAN_ANDERES] = [ean(0), ean(41)];
const [ZEHN_SELBES, ZEHN_ANDERES] = [isbnFormen(EAN_SELBES)[1], isbnFormen(EAN_ANDERES)[1]];
// Dieselben neun Ziffern mit einem anderen Prüfzeichen: Von elf ist genau eines richtig.
const ZEHN_FALSCH = ZEHN_ANDERES.slice(0, 9) + (ZEHN_ANDERES.endsWith('0') ? '1' : '0');
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
	return page.getByRole('dialog').filter({ hasText: 'Vorhandenen Titel öffnen?' });
}

test.describe('Neues Buch: die ISBN in der anderen Länge ist dieselbe Nummer', () => {
	test.beforeAll(() => {
		// Zehnstellig angelegt, wie vom Titelblatt abgeschrieben.
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, medientyp)
			VALUES ('Aus Littera ${s}', 'E2E', '${ZEHN_SELBES}', 'Lit 1', 'Buch'),
			       ('Fremdes Buch ${s}', 'E2E', '${ZEHN_ANDERES}', 'Lit 2', 'Buch');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-AL-' || isbn, true FROM buecher_titel WHERE isbn IN ('${EAN_SELBES}', '${EAN_ANDERES}');`);
	});
	test.afterAll(() => {
		const alle = `'${ZEHN_SELBES}','${ZEHN_ANDERES}','${ZEHN_FALSCH}','${EAN_SELBES}','${EAN_ANDERES}'`;
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN (${alle}));
			DELETE FROM buecher_titel WHERE isbn IN (${alle});`);
	});

	test('der Katalog führt die zehnstellig angelegte ISBN dreizehnstellig', () => {
		expect([titelAnzahl(EAN_SELBES), titelAnzahl(ZEHN_SELBES)]).toEqual(['1', '0']);
	});

	for (const [weg, eingabe] of [
		['der Strichcode vom Buch', EAN_SELBES],
		['die zehnstellige vom Titelblatt', ZEHN_SELBES]
	]) {
		test(`${weg}: die Maske nennt den vorhandenen Titel und öffnet ihn`, async ({ page }) => {
			const frage = await neuesBuchMitIsbn(page, eingabe);
			await expect(frage).toContainText(`Aus Littera ${s}`);

			await frage.getByRole('button', { name: 'Titel öffnen' }).click();

			await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
			await expect(page.locator('#buch-titel')).toHaveValue(`Aus Littera ${s}`);
			await expect(page.locator('#buch-isbn')).toHaveValue(EAN_SELBES);
			expect([titelAnzahl(EAN_SELBES), titelAnzahl(ZEHN_SELBES)]).toEqual(['1', '0']);
		});
	}

	test('falsches Prüfzeichen: die Maske fragt nicht und legt die Nummer als eigenen Titel an', async ({
		page
	}) => {
		// Die Katalogdienste sind ersetzt: Zu einer erfundenen Nummer wissen sie nichts.
		await page.route('**/api/lookup/**', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ data: { title: `Anderes Buch ${s}`, author: 'Probe, Paula' } })
			})
		);
		const frage = await neuesBuchMitIsbn(page, ZEHN_FALSCH);

		// Die Angaben der Katalogdienste stehen da: Die Abfrage des eigenen Katalogs ist durch,
		// und gefragt hat die Maske nicht.
		await expect(page.locator('#buch-titel')).toHaveValue(`Anderes Buch ${s}`);
		await expect(frage).toHaveCount(0);
		await expect(page.getByRole('dialog').filter({ hasText: 'Ist es dasselbe Buch?' })).toHaveCount(
			0
		);
		await page.locator('#buch-signatur').fill('Lit 3');

		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect([titelAnzahl(ZEHN_FALSCH), titelAnzahl(EAN_ANDERES)]).toEqual(['1', '1']);
	});
});
