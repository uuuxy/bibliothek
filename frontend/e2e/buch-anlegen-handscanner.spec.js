import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Ein Handscanner tippt blind in das Feld mit dem Fokus und schließt mit der Eingabetaste. In
// der Maske „Neues Buch" steht der Fokus deshalb im ISBN-Feld, und die Eingabetaste fragt wie
// das Verlassen des Feldes. Getippt wird über page.keyboard und ohne Klick ins Feld: fill()
// und click() setzten den Fokus selbst und sähen nicht, dass er fehlt.
const s = uniqueSuffix().slice(0, 6);
/** Gültige ISBN-13 aus der Uhrzeit. @param {number} versatz */
function isbn(versatz) {
	const kern = ('978' + String(Date.now() + versatz).slice(-9)).slice(0, 12);
	const summe = [...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0);
	return kern + ((10 - (summe % 10)) % 10);
}
const ISBN_NEU = isbn(0);
const ISBN_DA = isbn(11);
const ISBN_FREMD = isbn(23);
const ISBN_ZUERST = isbn(37);
const TITEL_DA = `Scan vorhanden ${s}`;

/** @param {import('@playwright/test').Page} page */
const fokus = (page) => page.evaluate(() => document.activeElement?.id ?? '');

/** @param {import('@playwright/test').Page} page @param {string} nummer */
async function scanne(page, nummer) {
	await page.keyboard.type(nummer, { delay: 10 });
	await page.keyboard.press('Enter');
}

/** @param {import('@playwright/test').Page} page */
async function oeffneNeuesBuch(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
}

test.describe.serial('Buch anlegen mit dem Handscanner', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, medientyp, cover_url)
			VALUES ('${TITEL_DA}', 'E2E', '${ISBN_DA}', 'Sca 1', 'Buch', '/covers/e2e-dummy.jpg');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-SCAN-${s}', true FROM buecher_titel WHERE isbn = '${ISBN_DA}';`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN ('${ISBN_NEU}','${ISBN_DA}','${ISBN_FREMD}','${ISBN_ZUERST}'));
			DELETE FROM buecher_titel WHERE isbn IN ('${ISBN_NEU}','${ISBN_DA}','${ISBN_FREMD}','${ISBN_ZUERST}');`);
	});

	test('ein Scan nach dem Öffnen holt die Angaben, ein zweiter ersetzt sie', async ({ page }) => {
		// Die Katalogdienste sind ersetzt: Der Test hängt nicht daran, was sie zu einer
		// erfundenen ISBN sagen. Das zweite Buch nennt keinen Autor.
		const dienste = {
			[ISBN_ZUERST]: { title: `Zuerst gescannt ${s}`, author: 'Eins, Autor' },
			[ISBN_NEU]: { title: `Scanbuch ${s}` }
		};
		/** @type {string[]} */
		const abgefragt = [];
		await page.route('**/api/lookup/**', (route) => {
			const nummer = new URL(route.request().url()).pathname.split('/').pop() ?? '';
			abgefragt.push(nummer);
			return route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ data: dienste[nummer] })
			});
		});
		await oeffneNeuesBuch(page);
		await expect.poll(() => fokus(page)).toBe('buch-isbn');

		await scanne(page, ISBN_ZUERST);
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_ZUERST);
		await expect(page.locator('#buch-titel')).toHaveValue(`Zuerst gescannt ${s}`);
		await expect(page.locator('#buch-autor')).toHaveValue('Eins, Autor');

		// Ohne Klick ins Feld: Der zweite Scan ersetzt die ISBN und alles, was zur ersten kam.
		await scanne(page, ISBN_NEU);
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_NEU);
		await expect(page.locator('#buch-titel')).toHaveValue(`Scanbuch ${s}`);
		await expect(page.locator('#buch-autor')).toHaveValue('');

		await page.locator('#buch-signatur').fill('Sca 2');
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(
			querySQL(
				`SELECT t.titel || '/' || count(e.id) FROM buecher_titel t LEFT JOIN buecher_exemplare e ON e.titel_id = t.id WHERE t.isbn = '${ISBN_NEU}' GROUP BY t.titel`
			)
		).toBe(`Scanbuch ${s}/1`);
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ISBN_ZUERST}'`)).toBe('0');
		expect(
			abgefragt,
			'je Scan eine Abfrage, das Verlassen des Feldes fragt nicht noch einmal'
		).toEqual([ISBN_ZUERST, ISBN_NEU]);
	});

	test('eine vergebene ISBN: die Frage kommt mit dem Scan, der nächste öffnet den Titel', async ({
		page
	}) => {
		await oeffneNeuesBuch(page);
		await expect.poll(() => fokus(page)).toBe('buch-isbn');

		await scanne(page, ISBN_DA);
		const frage = page.getByRole('dialog').filter({ hasText: 'Vorhandenen Titel öffnen?' });
		await expect(frage).toContainText(TITEL_DA);

		// Der zweite Scan trifft die Frage: Seine Eingabetaste nimmt die Vorgabe „Titel öffnen".
		await scanne(page, ISBN_FREMD);
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(TITEL_DA);
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_DA);

		// Ein dritter Scan darf die ISBN des geöffneten Titels nicht überschreiben.
		await expect.poll(() => fokus(page)).not.toBe('buch-isbn');
		await scanne(page, ISBN_FREMD);
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_DA);
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ISBN_FREMD}'`)).toBe('0');
		expect(querySQL(`SELECT isbn FROM buecher_titel WHERE titel = '${TITEL_DA}'`)).toBe(ISBN_DA);
	});

	test('„Buch bearbeiten" öffnet ohne Fokus im ISBN-Feld', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL_DA);
		await page.getByText(TITEL_DA).first().click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_DA);

		expect(await fokus(page)).not.toBe('buch-isbn');
		await scanne(page, ISBN_FREMD);
		await expect(page.locator('#buch-isbn')).toHaveValue(ISBN_DA);
	});
});
