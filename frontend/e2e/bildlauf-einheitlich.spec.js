// Jede Fläche, die rollt, zeigt denselben Balken (styles/basis.css): einen Daumen von 8 px mit
// halbrunden Enden, ohne Spur, außerhalb der Seitenleiste auch ohne Maus, wie Gmails Mailliste.
// Gemessen wird, was der Browser zeichnet. Dafür startet Chromium hier mit Balken: Playwright setzt
// sonst --hide-scrollbars, und dann belegt kein Balken Platz.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

test.use({ launchOptions: { ignoreDefaultArgs: ['--hide-scrollbars'] } });

/** Die Fläche, die den Inhalt rollt (main in Hauptbereich.svelte): rechts am Fensterrand, mit
 *  Bildlauf. Läuft im Browser. */
const INHALT = () => {
	const flaeche = [...document.querySelectorAll('*')].find((el) => {
		const r = el.getBoundingClientRect();
		return (
			getComputedStyle(el).overflowY === 'auto' &&
			el.scrollHeight > el.clientHeight &&
			Math.round(r.right) === window.innerWidth
		);
	});
	if (!flaeche) return null;
	const r = flaeche.getBoundingClientRect();
	return {
		belegt: flaeche.offsetWidth - flaeche.clientWidth,
		rechts: r.right,
		oben: r.top,
		unten: r.bottom
	};
};

test('Inhaltsbereich: derselbe schmale Balken ohne Spur, auch ohne Maus', async ({ page }) => {
	await page.setViewportSize({ width: 1400, height: 600 });
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await expect
		.poll(() => page.evaluate(INHALT), 'die Seite ist höher als das Fenster')
		.not.toBeNull();
	await page.mouse.move(700, 300);
	const lage = await page.evaluate(INHALT);
	if (!lage) throw new Error('Keine rollende Fläche am rechten Rand');
	expect(lage.belegt, '8 px statt der 15 von macOS').toBe(8);

	// Oben steht der Daumen (Bildlauf ganz oben), unten in der Spur keiner. Gleicht der untere
	// Streifen dem Grund daneben im Innenabstand, hat der Balken keine Spur.
	const streifen = (/** @type {number} */ x, /** @type {number} */ y) =>
		page.screenshot({ clip: { x, y, width: 8, height: 30 } });
	const spur = lage.rechts - 8;
	const oben = await streifen(spur, lage.oben + 20);
	const unten = await streifen(spur, lage.unten - 40);
	const daneben = await streifen(spur - 20, lage.unten - 40);
	expect(oben.equals(unten), 'oben steht der Daumen, auch ohne Maus über dem Balken').toBe(false);
	expect(unten.equals(daneben), 'keine Spur: unten gleicht der Balken dem Grund daneben').toBe(
		true
	);
});
