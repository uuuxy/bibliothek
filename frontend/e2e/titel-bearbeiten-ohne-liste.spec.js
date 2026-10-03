import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// „Titel bearbeiten" in der Buchakte führt zur Maske, ohne dass die Titelliste dazwischen
// steht. Die Maske braucht nur den einen Titel; die ganze Liste lädt daneben, und zwar
// einmal. An einem großen Bestand ist sie mehrere Megabyte groß — hier bleiben beide
// Abfragen liegen, bis der Test sie freigibt, damit jeder Zwischenzustand prüfbar steht.

/**
 * Hält jede GET-Anfrage auf das Muster an, bis `frei()` gerufen wird, und zählt sie.
 * @param {import('@playwright/test').Page} page
 * @param {RegExp} muster
 */
async function halte(page, muster) {
	let zahl = 0;
	/** @type {(wert?: unknown) => void} */
	let frei = () => {};
	const freigabe = new Promise((fertig) => (frei = fertig));
	await page.route(muster, async (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		zahl++;
		await freigabe;
		await route.continue();
	});
	return { frei, zahl: () => zahl };
}

test('„Titel bearbeiten" öffnet die Maske, ohne auf die Titelliste zu warten', async ({ page }) => {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('heading', { level: 2 }).first().getByRole('button').click();
	const bearbeiten = page.getByRole('button', { name: 'Titel bearbeiten' });
	await expect(bearbeiten).toBeVisible();

	const liste = await halte(page, /\/api\/books(\?[^/]*)?$/);
	const einzeln = await halte(page, /\/api\/books\/[0-9a-f-]{36}$/);
	await bearbeiten.click();
	await expect(page.getByRole('tab', { name: 'Titel-Verwaltung' })).toHaveAttribute(
		'aria-selected',
		'true'
	);

	// Der Titel ist unterwegs: An der Stelle der Maske steht die Ladeanzeige, keine Tabelle.
	await expect.soft(page.getByRole('progressbar', { name: 'Lädt' })).toBeVisible();
	await expect.soft(page.locator('table'), 'die Titeltabelle steht dazwischen').toHaveCount(0);

	// Der Titel kommt an, die Liste liegt noch: Die Maske steht.
	einzeln.frei();
	await expect
		.soft(page.getByRole('heading', { name: 'Buch bearbeiten' }), 'die Maske wartet auf die Liste')
		.toBeVisible();

	// Die Liste lädt einmal. Die verzögerte Suche der Titel-Verwaltung (300 ms) zählt mit.
	liste.frei();
	await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
	await page.waitForTimeout(800);
	expect(liste.zahl(), 'Abrufe der ganzen Titelliste').toBe(1);
});
