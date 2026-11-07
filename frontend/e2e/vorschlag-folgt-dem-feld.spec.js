import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Vorschlagsliste hängt an ihrem Feld, auch wenn das Feld ohne Bildlauf rückt: Im Dialog
// „Inventur starten" fährt das Signaturfeld ein, der zentrierte Dialog wächst und rückt nach
// oben, und die Liste blieb in WebKit an der alten Stelle über dem Knopf „Starten" stehen.
// Hier rückt das Feld durch einen Innenabstand darüber, ohne dass etwas rollt.
const s = uniqueSuffix().slice(0, 6);
const SIGNATUR = `Fol${s}`;

test.beforeAll(() => {
	seedSQL(`INSERT INTO buecher_titel (titel, signatur) VALUES
		('Folgen A ${s}', '${SIGNATUR} A'), ('Folgen B ${s}', '${SIGNATUR} B');`);
});
test.afterAll(() => {
	seedSQL(`DELETE FROM buecher_titel WHERE signatur LIKE '${SIGNATUR}%';`);
});

test('die Vorschlagsliste folgt dem Feld, wenn es ohne Bildlauf rückt', async ({ page }) => {
	await page.setViewportSize({ width: 1400, height: 1000 });
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Titel anlegen' }).first().click();
	const feld = page.locator('#buch-signatur');
	await feld.fill(SIGNATUR);
	await expect(page.getByRole('option')).toHaveCount(2);

	// Abstand zwischen Feld und Liste: 4 px darunter oder, fehlt unten der Platz, 4 px darüber
	// (selectGeometrie).
	const abstand = () =>
		page.evaluate(() => {
			const f = document.querySelector('#buch-signatur')?.getBoundingClientRect();
			const l = document.querySelector('[role="listbox"]')?.getBoundingClientRect();
			if (!f || !l) return null;
			return Math.round(l.top >= f.bottom ? l.top - f.bottom : f.top - l.bottom);
		});
	await expect.poll(abstand).toBe(4);

	await feld.evaluate((el) => {
		/** @type {HTMLElement} */ (el.parentElement).style.paddingTop = '80px';
	});
	await expect.poll(abstand, 'die Liste folgt dem gerückten Feld').toBe(4);
});
