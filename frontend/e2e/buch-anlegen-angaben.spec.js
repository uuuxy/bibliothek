import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Was die Katalogdienste zu einer ISBN wissen, steht nach der Abfrage in der Maske, auch
// Untertitel und Listenpreis. Gespeichert wird, was dort steht: Der Server fragt beim
// Speichern keinen Dienst und trägt nichts nach.
const s = uniqueSuffix().slice(0, 6);
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const ISBN =
	kern + ((10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10);

test.afterAll(() => {
	seedSQL(`
		DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${ISBN}');
		DELETE FROM buecher_titel WHERE isbn = '${ISBN}';`);
});

test('die ISBN-Abfrage zeigt Untertitel und Listenpreis; gespeichert wird der Stand der Maske', async ({
	page
}) => {
	// Die Katalogdienste sind ersetzt: Der Test hängt nicht daran, was sie zu einer
	// erfundenen ISBN sagen.
	await page.route('**/api/lookup/**', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				data: {
					title: `Angaben ${s}`,
					subtitle: `Untertitel ${s}`,
					author: 'Probe, Paula',
					preis: 12.5
				}
			})
		})
	);
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();

	await page.locator('#buch-isbn').fill(ISBN);
	await page.locator('#buch-isbn').press('Tab');
	await expect(page.locator('#buch-titel')).toHaveValue(`Angaben ${s}`);
	await expect(page.locator('#buch-untertitel')).toHaveValue(`Untertitel ${s}`);
	await expect(page.locator('#buch-listenpreis')).toHaveValue('12.5');

	// Der Preis ist ein Vorschlag: Was jemand einträgt, gilt.
	await page.locator('#buch-listenpreis').fill('15');
	await page.locator('#buch-signatur').fill('Ang 1');
	await page.getByRole('button', { name: 'Speichern' }).click();
	await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();

	expect(
		querySQL(
			`SELECT titel || '|' || untertitel || '|' || autor || '|' || listenpreis FROM buecher_titel WHERE isbn = '${ISBN}'`
		)
	).toBe(`Angaben ${s}|Untertitel ${s}|Probe, Paula|15.00`);
});
