// Gate: Ein mehrzeiliges ui/Feld ist so hoch wie seine Zeilen.
//
// Der Rahmen um Beschriftung, Feld und Hinweis ist ein Raster mit `grid-rows-subgrid`,
// damit die drei Zeilen in einem Eltern-Raster über alle Spalten auf einer Linie
// stehen. Steht das Feld nicht in einem Raster (Einstellungen → Schadensersatz: flex), gibt
// es kein Eltern-Raster, und Chrome sizt die Zeile der Textarea ohne ihre `rows`: Gemessen
// stand das Feld auf 18 px — Rahmen und Innenabstand, null Zeilen Text. Einzeilige Felder
// merken davon nichts, sie tragen eine feste Höhe (h-9).
//
// Warum im Browser: Die Höhe entsteht erst beim Layout; jsdom rechnet keins.
import { test, expect } from '@playwright/test';
import { uiLogin, einstellungsKategorie } from './helpers.js';

/** @param {import('@playwright/test').Locator} feld */
async function textzeilen(feld) {
	return feld.evaluate((el) => {
		const ta = /** @type {HTMLTextAreaElement} */ (el);
		const cs = getComputedStyle(ta);
		const innen = ta.clientHeight - parseFloat(cs.paddingTop) - parseFloat(cs.paddingBottom);
		return { soll: ta.rows, ist: innen / parseFloat(cs.lineHeight) };
	});
}

test('Mehrzeilige Felder zeigen ihre Zeilen — auch außerhalb eines Rasters', async ({ page }) => {
	await uiLogin(page);

	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Schadensersatz').click();
	const bank = await textzeilen(page.locator('#bescheid-bank'));
	expect(bank.ist, `Bankverbindung: ${bank.ist.toFixed(2)} statt ${bank.soll} Zeilen`).toBeCloseTo(
		bank.soll,
		1
	);
});
