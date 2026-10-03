import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Ein Titel ohne ISBN: Zeitschriften, Spiele und alte Bücher tragen keine, und nach der
// Übernahme aus Littera steht rund ein Drittel des Katalogs ohne da. Die Maske ändert einen
// solchen Titel, legt ihn an und fragt bei gleichem Titel und Autor, ob es dasselbe Medium ist.
const s = uniqueSuffix().slice(0, 6);
const BESTAND = `Ohne ISBN Bestand ${s}`;
const HEFT = `Ohne ISBN Heft ${s}`;
const zaehle = (/** @type {string} */ titel) =>
	querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = '${titel}'`);

/** @param {import('@playwright/test').Page} page */
async function zurTitelVerwaltung(page) {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
}

/** Öffnet „Neues Buch" und trägt das Heft ohne ISBN ein.
 * @param {import('@playwright/test').Page} page */
async function neuesHeft(page) {
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await page.locator('#buch-titel').fill(HEFT);
	await page.locator('#buch-autor').fill('Redaktion');
	await page.locator('#buch-signatur').fill('Z 1');
}

test.describe.serial('Titel ohne ISBN', () => {
	test.beforeAll(() => {
		// Zwei Hefte, wie die Übernahme sie hinterlässt: ohne ISBN, mit gleichem Titel und Autor.
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, autor, isbn)
				VALUES ('${BESTAND}', 'Redaktion', NULL), ('${BESTAND}', 'Redaktion', NULL)
				RETURNING id
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-oi-${s}-' || row_number() OVER (), true FROM t;
		`);
	});
	test.afterAll(() => {
		seedSQL(`DELETE FROM buecher_titel WHERE titel IN ('${BESTAND}', '${HEFT}');`);
	});

	test('ein Titel ohne ISBN aus dem Bestand nimmt eine Signatur an', async ({ page }) => {
		await zurTitelVerwaltung(page);
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(BESTAND);
		await page.getByText(BESTAND).first().click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-isbn')).toHaveValue('');

		await page.locator('#buch-signatur').fill(`OI ${s}`);
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();

		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(
			querySQL(
				`SELECT count(*) FILTER (WHERE signatur = 'OI ${s}') || ' von ' || count(*) || ', mit ISBN ' || count(isbn) FROM buecher_titel WHERE titel = '${BESTAND}'`
			)
		).toBe('1 von 2, mit ISBN 0');
	});

	test('ohne Titel führt „Speichern" ins Feld und schickt nichts', async ({ page }) => {
		/** @type {string[]} */
		const geschickt = [];
		page.on('request', (anfrage) => {
			if (new URL(anfrage.url()).pathname === '/api/books' && anfrage.method() === 'POST')
				geschickt.push(anfrage.url());
		});
		await zurTitelVerwaltung(page);
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		const fehler = page.getByText('Bitte den Titel eintragen. Gespeichert wird erst mit ihm.');
		await expect(fehler, 'vor dem Klick steht kein Fehler am Feld').toHaveCount(0);

		await page.locator('#buch-signatur').fill('Z 1');
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();

		await expect(fehler).toBeVisible();
		await expect(page.locator('#buch-titel')).toBeFocused();
		await expect(page.locator('#buch-titel')).toHaveAttribute('aria-invalid', 'true');
		await page.locator('#buch-titel').fill('X');
		await expect(fehler).toHaveCount(0);
		expect(geschickt).toEqual([]);
	});

	test('ein neues Heft ohne ISBN wird angelegt, mit einem Exemplar', async ({ page }) => {
		await zurTitelVerwaltung(page);
		await neuesHeft(page);
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();

		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(
			querySQL(
				`SELECT coalesce(t.isbn, 'keine') || '|' || t.autor || '|' || count(e.id) FROM buecher_titel t LEFT JOIN buecher_exemplare e ON e.titel_id = t.id WHERE t.titel = '${HEFT}' GROUP BY t.id`
			)
		).toBe('keine|Redaktion|1');
	});

	test('gleicher Titel und Autor: die Maske fragt, Escape legt nichts an, „Anderes Medium" schon', async ({
		page
	}) => {
		await zurTitelVerwaltung(page);
		await neuesHeft(page);
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();

		const frage = page.getByRole('dialog', { name: 'Ist es dasselbe Medium?' });
		await expect(frage).toContainText(HEFT);
		// Die Eingabetaste eines Scans träfe den Knopf mit dem Fokus: Er öffnet, er legt nicht an.
		await expect(frage.getByRole('button', { name: 'Titel öffnen' })).toBeFocused();
		await page.keyboard.press('Escape');
		await expect(frage).toHaveCount(0);
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(HEFT);
		expect(zaehle(HEFT), 'Escape hat einen Titel angelegt').toBe('1');

		await page.getByRole('button', { name: 'Speichern', exact: true }).click();
		await frage.getByRole('button', { name: 'Anderes Medium' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(zaehle(HEFT)).toBe('2');
	});

	test('„Titel öffnen" führt zum vorhandenen Heft', async ({ page }) => {
		await zurTitelVerwaltung(page);
		await neuesHeft(page);
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();

		const frage = page.getByRole('dialog', { name: 'Ist es dasselbe Medium?' });
		await frage.getByRole('button', { name: 'Titel öffnen' }).click();

		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(HEFT);
		await expect(page.locator('#buch-isbn')).toHaveValue('');
		expect(zaehle(HEFT)).toBe('2');
	});
});
