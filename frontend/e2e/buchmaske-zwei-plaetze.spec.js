import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Zwei Plätze haben denselben Titel offen. Die Maske schickt die Felder, die sie seit dem
// Öffnen geändert hat, und der Server schreibt nur diese: Was der eine Platz speichert, bleibt
// stehen, wenn danach der andere ein anderes Feld speichert. Vorher gingen alle Felder mit
// dem Stand vom Öffnen zurück, und die Eingabe des ersten war ohne Meldung wieder weg.
const s = uniqueSuffix().slice(0, 6);
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const ISBN =
	kern + ((10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10);
const TITEL = `Zwei Plätze ${s}`;

/** @param {string} spalte */
const steht = (spalte) =>
	querySQL(`SELECT coalesce(${spalte}::text, '') FROM buecher_titel WHERE isbn = '${ISBN}'`);

/** @param {import('@playwright/test').Page} page */
async function oeffneTitel(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL);
	await page.getByText(TITEL).first().click();
	await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
	await expect(page.locator('#buch-verlag')).toHaveValue(/Verlag/);
}

/**
 * Klickt „Speichern" und liefert, was die Maske dabei an den Server schickt.
 * @param {import('@playwright/test').Page} page
 */
async function speichern(page) {
	const anfrage = page.waitForRequest(
		(r) => r.method() === 'PUT' && r.url().includes('/api/books/')
	);
	await page.getByRole('button', { name: 'Speichern' }).click();
	const rumpf = JSON.parse((await anfrage).postData() || 'null');
	await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
	return rumpf;
}

test.describe.serial('Buchmaske: zwei Plätze, ein Titel', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, verlag, erscheinungsjahr, listenpreis,
				medientyp, cover_url)
			VALUES ('${TITEL}', 'E2E', '${ISBN}', 'Zwe 1', 'Alter Verlag', 2019, 12.50, 'Buch',
				'/covers/e2e-dummy.jpg');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-ZP-${s}', true FROM buecher_titel WHERE isbn = '${ISBN}';`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${ISBN}');
			DELETE FROM buecher_titel WHERE isbn = '${ISBN}';`);
	});

	test('ohne Eingabe gespeichert: die Maske nennt kein Feld', async ({ page }) => {
		await oeffneTitel(page);
		expect(await speichern(page)).toEqual({});
		expect(steht('verlag')).toBe('Alter Verlag');
		expect(steht('listenpreis')).toBe('12.50');
	});

	test('jeder Platz behält, was er gespeichert hat', async ({ browser }) => {
		const platz1 = await (await browser.newContext()).newPage();
		const platz2 = await (await browser.newContext()).newPage();
		await oeffneTitel(platz1);
		await oeffneTitel(platz2);

		await platz2.locator('#buch-verlag').fill('Neuer Verlag');
		expect(await speichern(platz2)).toEqual({ verlag: 'Neuer Verlag' });
		expect(steht('verlag')).toBe('Neuer Verlag');

		// Platz 1 zeigt noch den alten Verlag und speichert ein anderes Feld.
		await expect(platz1.locator('#buch-verlag')).toHaveValue('Alter Verlag');
		await platz1.locator('#buch-signatur').fill('Zwe 2');
		expect(await speichern(platz1)).toEqual({ signatur: 'Zwe 2' });

		expect(steht('signatur')).toBe('Zwe 2');
		expect(steht('verlag')).toBe('Neuer Verlag');
		expect(steht('erscheinungsjahr')).toBe('2019');
		expect(steht('listenpreis')).toBe('12.50');
	});
});
