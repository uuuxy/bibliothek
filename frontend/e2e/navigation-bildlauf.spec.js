// Der Bildlauf der Seitenleiste wie bei Gmail (styles/komponenten.css, .leiste-bildlauf): Der
// Daumen erscheint nur, solange die Maus über der Leiste steht, ohne Spur in 8 px, die dabei frei
// bleiben, damit nichts springt; in der schmalen Leiste gibt es keinen Balken. Gemessen wird, was
// der Browser zeichnet. Dafür startet Chromium hier mit Balken: Playwright setzt sonst
// --hide-scrollbars, und dann belegt kein Balken Platz.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

test.use({ launchOptions: { ignoreDefaultArgs: ['--hide-scrollbars'] } });

/** Breite des Balkens ohne den Rand der Leiste, und ob die Einträge höher sind als sie. */
const BELEGT = () => {
	const leiste = /** @type {HTMLElement} */ (document.querySelector('aside .leiste-bildlauf'));
	const stil = getComputedStyle(leiste);
	const rand = parseFloat(stil.borderLeftWidth) + parseFloat(stil.borderRightWidth);
	return {
		belegt: leiste.offsetWidth - leiste.clientWidth - rand,
		hoeher: leiste.scrollHeight > leiste.clientHeight
	};
};

test('Seitenleiste: Balken wie bei Gmail nur unter der Maus, eingeklappt keiner', async ({
	page
}) => {
	// Niedriges Fenster, damit die Einträge höher sind als die Leiste; ab 1280 px beginnt sie breit.
	await page.setViewportSize({ width: 1400, height: 520 });
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	const leiste = page.locator('aside .leiste-bildlauf');
	const lage = () => page.evaluate(BELEGT);
	// Oben in der Spur steht der Daumen (Bildlauf ganz oben), unten nicht: Gleichen sich beide
	// Streifen, ist kein Daumen gezeichnet.
	const daumen = async () => {
		const b = await leiste.boundingBox();
		if (!b) throw new Error('Leiste ohne Maße');
		const streifen = (/** @type {number} */ y) =>
			page.screenshot({ clip: { x: b.x + b.width - 9, y, width: 8, height: 30 } });
		return !(await streifen(b.y + 70)).equals(await streifen(b.y + b.height - 60));
	};

	// Breit, die Maus im Inhalt: 8 px frei, kein Daumen.
	await page.mouse.move(1000, 300);
	await expect.poll(lage).toEqual({ belegt: 8, hoeher: true });
	await expect.poll(daumen, 'ohne Maus kein Daumen').toBe(false);

	// Unter der Maus erscheint er, die Breite bleibt; beim Hinausgehen verschwindet er.
	await leiste.hover({ position: { x: 100, y: 300 } });
	await expect.poll(daumen, 'unter der Maus steht der Daumen').toBe(true);
	expect(await lage(), 'nichts springt').toEqual({ belegt: 8, hoeher: true });
	await page.mouse.move(1000, 300);
	await expect.poll(daumen, 'nach dem Hinausgehen kein Daumen').toBe(false);

	// Eingeklappt belegt der Balken nichts, auch unter der Maus nicht, bevor sie aufklappt.
	await page.getByRole('button', { name: 'Navigation einklappen' }).click();
	await page.mouse.move(1000, 300);
	await expect.poll(async () => (await lage()).belegt, 'eingeklappt kein Balken').toBe(0);

	// Unter der Maus klappt sie auf und zeigt ihn; danach ist sie wieder schmal und ohne Balken.
	await leiste.hover({ position: { x: 30, y: 300 } });
	await expect.poll(lage).toEqual({ belegt: 8, hoeher: true });
	await expect.poll(daumen, 'aufgeklappt unter der Maus steht der Daumen').toBe(true);
	await page.mouse.move(1000, 300);
	await expect.poll(async () => (await lage()).belegt).toBe(0);
});
