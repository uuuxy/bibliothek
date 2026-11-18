// Gate: Rand der Arbeitsfläche je Fensterklasse nach Material 3 (Breakpoints).
//
// Googles Fensterklassen: compact unter 600 dp, medium 600–839, expanded 840–1199, large
// 1200–1599, extra-large ab 1600. „In compact layouts, use 16dp margins"; „Medium layouts have
// margins of 24dp", ebenso expanded, large und extra-large. Gemessen wird der Abstand des Inhalts
// vom Rand der Arbeitsfläche rechts der Navigation, je Klasse an einer Breite und an der Grenze
// 599/600, an der Tailwinds sm jetzt beginnt (styles/theme-mass.css).
// Sieht nicht: den Rand auf der rechten Seite (dort liegt der Rollbalken), Seiten außerhalb der
// Anwendung (Anmeldung, Katalog) und Abstände innerhalb der Seiten; die Skala der Abstände hält
// frontend-hygiene-abstaende.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

/** Abstand vom Rand der Arbeitsfläche bis zum Inhalt. Läuft im Browser. */
function messeRand() {
	const main = document.getElementById('hauptinhalt');
	const flaeche = main?.parentElement;
	if (!main || !flaeche) return null;
	const inhalt = main.getBoundingClientRect().left + parseFloat(getComputedStyle(main).paddingLeft);
	return Math.round(inhalt - flaeche.getBoundingClientRect().left);
}

for (const [klasse, breite, soll] of /** @type {const} */ ([
	['compact', 400, 16],
	['compact an der Grenze', 599, 16],
	['medium an der Grenze', 600, 24],
	['medium', 700, 24],
	['expanded', 1000, 24],
	['large', 1400, 24],
	['extra-large', 1700, 24]
])) {
	test(`Rand der Arbeitsfläche ${soll} dp bei ${klasse} (${breite} px)`, async ({ page }) => {
		await page.setViewportSize({ width: breite, height: 900 });
		await uiLogin(page);
		for (const pfad of ['/schuelerdatei', '/bestellungen', '/einstellungen']) {
			await gehZu(page, pfad);
			await expect(page.locator('#hauptinhalt')).toBeVisible();
			expect(await page.evaluate(messeRand), `${pfad} bei ${breite} px`).toBe(soll);
		}
	});
}
