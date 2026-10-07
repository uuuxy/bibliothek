import { test, expect } from '@playwright/test';
import { uiLogin, seedBestellung, gehZu } from './helpers.js';

// Die Spalte „Lieferant" der Bestellhistorie bekommt die übrige Breite. Ihre Zelle kürzt
// den Namen (max-w-0 mit truncate); ohne volle Breite ging der Platz an die Nachbarspalten,
// und der Name stand bei 1710 px Fensterbreite mit 149 von 543 px da.
test('Bestellhistorie: der Name des Lieferanten steht ganz da, wenn die Zeile Platz hat', async ({
	page
}) => {
	const { marke, aufraeumen } = seedBestellung();
	try {
		await page.setViewportSize({ width: 1710, height: 952 });
		await uiLogin(page);
		await gehZu(page, '/bestellungen');
		await page.getByRole('tab', { name: 'Bestellhistorie', exact: true }).click();

		const zeile = page.getByRole('button', { name: new RegExp(`bei ${marke} öffnen`) });
		await zeile.waitFor({ timeout: 15_000 });
		const name = zeile.getByText(marke, { exact: true });
		const mass = await name.evaluate((el) => ({
			sichtbar: el.clientWidth,
			ganz: el.scrollWidth,
			zelle: Math.round(el.closest('td')?.getBoundingClientRect().width ?? 0),
			tabelle: Math.round(el.closest('table')?.getBoundingClientRect().width ?? 0)
		}));
		expect(
			mass.ganz,
			`Vom Namen „${marke}" stehen ${mass.sichtbar} von ${mass.ganz} px da; die Zelle hat ` +
				`${mass.zelle} von ${mass.tabelle} px der Tabelle. Die Zelle braucht w-full zu max-w-0.`
		).toBeLessThanOrEqual(mass.sichtbar);

		// Wo der Platz nicht reicht, führt die Zeile zur Bestellung, die den Namen ganz nennt.
		await zeile.click();
		await expect(page.getByRole('heading', { name: 'Bestellte Titel' })).toBeVisible();
		await expect(page.getByText(marke).first()).toBeVisible();
	} finally {
		aufraeumen();
	}
});
