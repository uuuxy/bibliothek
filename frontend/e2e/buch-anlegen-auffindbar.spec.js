import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Ein Buch aus der Maske „Neues Buch" ist danach zu finden, und eine vergebene ISBN führt zu
// ihrem Titel. Ohne Exemplar zeigt keine Suche einen Titel (docs/OFFEN.md 9.4): Mit der
// Vorgabe Bestand 0 hieß das Speichern „erfolgreich", das Buch stand nirgends, und der zweite
// Versuch endete mit „existiert bereits". Gefragt wird, sobald die ISBN im Feld steht — wie in
// Littera bei der Eingabe, nicht erst beim Speichern.
const s = uniqueSuffix().slice(0, 6);
/** Gültige ISBN-13 aus der Uhrzeit: 978 + 9 Ziffern + Prüfziffer. @param {number} versatz */
function isbn(versatz) {
	const kern = ('978' + String(Date.now() + versatz).slice(-9)).slice(0, 12);
	const summe = [...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0);
	return kern + ((10 - (summe % 10)) % 10);
}
const ISBN_MIT = isbn(0);
const ISBN_OHNE = isbn(7);
const exemplare = (/** @type {string} */ nummer) =>
	querySQL(
		`SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id WHERE t.isbn = '${nummer}'`
	);

/** @param {import('@playwright/test').Page} page */
async function zurTitelVerwaltung(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
}

/** Titel zuerst: Mit Titel schlägt das ISBN-Feld beim Verlassen nicht bei der DNB nach.
 * @param {import('@playwright/test').Page} page @param {string} titel @param {string} nummer */
async function neuesBuch(page, titel, nummer) {
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await page.locator('#buch-titel').fill(titel);
	await page.locator('#buch-isbn').fill(nummer);
	await page.locator('#buch-signatur').fill('BIB An');
}

/** @param {import('@playwright/test').Page} page @param {'Mit Exemplaren'|'Ohne Exemplare'} sicht @param {string} nummer */
async function trefferInDerTitelliste(page, sicht, nummer) {
	await page
		.getByRole('group', { name: 'Sicht der Titelliste' })
		.getByRole('button', { name: sicht })
		.click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(nummer);
	return page.getByRole('heading', { name: /^Bücher \(\d+\)$/ });
}

test.describe.serial('Buch anlegen: danach auffindbar', () => {
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN ('${ISBN_MIT}','${ISBN_OHNE}'));
			DELETE FROM buecher_titel WHERE isbn IN ('${ISBN_MIT}','${ISBN_OHNE}');
		`);
	});

	test('ein neues Buch beginnt mit einem Exemplar und steht in beiden Suchen', async ({ page }) => {
		await zurTitelVerwaltung(page);
		await neuesBuch(page, `Anlegen ${s}`, ISBN_MIT);
		await expect(page.locator('#buch-bestand')).toHaveValue('1');
		await expect(page.locator('#buch-bestand-hinweis')).toHaveCount(0);
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(exemplare(ISBN_MIT)).toBe('1');

		await expect(await trefferInDerTitelliste(page, 'Mit Exemplaren', ISBN_MIT)).toHaveText(
			'Bücher (1)'
		);
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill('');
		await page.getByRole('tab', { name: 'Suche & Filter' }).click();
		await page
			.getByRole('searchbox', { name: 'Suchen nach Titel, Fach, Klasse oder Autor' })
			.fill(ISBN_MIT);
		await expect(page.getByText(`Anlegen ${s}`).first()).toBeVisible();
	});

	test('Bestand 0 sagt vor und nach dem Speichern, wo der Titel steht', async ({ page }) => {
		await zurTitelVerwaltung(page);
		await neuesBuch(page, `Nur Titel ${s}`, ISBN_OHNE);
		await page.locator('#buch-bestand').fill('0');
		await expect(page.locator('#buch-bestand-hinweis')).toContainText('„Ohne Exemplare“');
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Titel ohne Exemplar gespeichert.')).toBeVisible();
		expect(exemplare(ISBN_OHNE)).toBe('0');

		await expect(await trefferInDerTitelliste(page, 'Mit Exemplaren', ISBN_OHNE)).toHaveText(
			'Bücher (0)'
		);
		await expect(await trefferInDerTitelliste(page, 'Ohne Exemplare', ISBN_OHNE)).toHaveText(
			'Bücher (1)'
		);
	});

	test('eine vergebene ISBN: die Frage kommt beim Verlassen des ISBN-Felds und führt zum Titel', async ({
		page
	}) => {
		// Für ein Buch, das es schon gibt, fragt die Maske weder die Katalogdienste noch
		// versucht sie, einen zweiten Titel anzulegen.
		/** @type {string[]} */
		const unnoetig = [];
		page.on('request', (anfrage) => {
			const pfad = new URL(anfrage.url()).pathname;
			if (pfad.startsWith('/api/lookup/') || (pfad === '/api/books' && anfrage.method() === 'POST'))
				unnoetig.push(`${anfrage.method()} ${pfad}`);
		});
		await zurTitelVerwaltung(page);
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await page.locator('#buch-isbn').fill(ISBN_OHNE);
		await page.locator('#buch-isbn').press('Tab');

		const frage = page.getByRole('dialog').filter({ hasText: 'Vorhandenen Titel öffnen?' });
		await expect(frage).toContainText(`Nur Titel ${s}`);
		await expect(frage).toContainText('„Ohne Exemplare“');
		await frage.getByRole('button', { name: 'Titel öffnen' }).click();

		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(`Nur Titel ${s}`);
		await page.locator('#buch-bestand').fill('1');
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();

		expect(unnoetig).toEqual([]);
		expect(exemplare(ISBN_OHNE)).toBe('1');
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ISBN_OHNE}'`)).toBe('1');
		await expect(await trefferInDerTitelliste(page, 'Mit Exemplaren', ISBN_OHNE)).toHaveText(
			'Bücher (1)'
		);
	});

	test('Abbrechen lässt die Maske stehen, und das Speichern fragt noch einmal', async ({
		page
	}) => {
		await zurTitelVerwaltung(page);
		// Titel zuerst, dann die ISBN: Die Frage hängt nicht daran, dass der Titel leer ist.
		await neuesBuch(page, `Zweiter Versuch ${s}`, ISBN_MIT);

		const frage = page.getByRole('dialog').filter({ hasText: 'Vorhandenen Titel öffnen?' });
		await expect(frage).toContainText(`Anlegen ${s}`);
		await expect(frage).not.toContainText('„Ohne Exemplare“');
		await frage.getByRole('button', { name: 'Abbrechen' }).click();
		await expect(frage).toHaveCount(0);
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(`Zweiter Versuch ${s}`);

		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(frage).toContainText(`Anlegen ${s}`);
		await frage.getByRole('button', { name: 'Abbrechen' }).click();

		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ISBN_MIT}'`)).toBe('1');
		expect(
			querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = 'Zweiter Versuch ${s}'`)
		).toBe('0');
	});
});
