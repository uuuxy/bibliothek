import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, gehZu, einstellungsKategorie, uniqueSuffix } from './helpers.js';

// Lehnt der Server einen neuen Lieferanten ab, bleibt die Eingabe zum Berichtigen stehen; erst
// das Anlegen leert das Formular. Abgelehnt wird über das Netz, angelegt am echten Server.
const NAME = `E2E-Abgelehnt-${uniqueSuffix().slice(0, 6)}`;

test.afterEach(() => {
	seedSQL(`DELETE FROM lieferanten WHERE name LIKE 'E2E-Abgelehnt-%'`);
});

test('Ein abgelehnter Lieferant lässt die Eingabe stehen, das Anlegen leert sie', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Lieferanten').click();

	const name = page.getByLabel('Name', { exact: true });
	await name.fill(NAME);
	await page.getByLabel('E-Mail', { exact: true }).fill('abgelehnt@example.org');
	await page.getByLabel('Kundennummer', { exact: true }).fill('K-ABGELEHNT');
	const speichern = page.getByRole('button', { name: 'Lieferanten speichern' });

	await page.route('**/api/lieferanten', (route) =>
		route.request().method() === 'POST'
			? route.fulfill({
					status: 400,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'Probe: abgelehnt' })
				})
			: route.fallback()
	);
	await speichern.click();
	await expect(page.getByText('Probe: abgelehnt')).toBeVisible();
	await expect(name, 'abgelehnt: die Eingabe bleibt').toHaveValue(NAME);

	await page.unroute('**/api/lieferanten');
	await speichern.click();
	await expect(page.getByRole('table').getByText(NAME)).toBeVisible();
	await expect(name, 'angelegt: das Formular ist leer').toHaveValue('');
});
