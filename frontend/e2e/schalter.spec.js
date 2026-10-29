import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// Maße und Farben des Griffs nach Material 3 (Switch, Specs; Token-Datei md-comp-switch): Griff 16
// aus, 24 an, 28 beim Drücken, mittig im 32-px-Feld am Anfang oder Ende der 52 × 32 großen Spur
// (material-web, _handle.scss), beim Überfahren aus on-surface-variant, an primary-container.
// Gemessen am Schalter „Dunkles Design", der im Fuß jeder Seite steht.

test.use({ reducedMotion: 'reduce' });

/**
 * Lage und Farbe des Griffs, gemessen von der Außenkante der Spur.
 * @param {import('@playwright/test').Locator} schalter
 */
const miss = (schalter) =>
	schalter.evaluate((el) => {
		const spur = el.getBoundingClientRect();
		const griff = /** @type {HTMLElement} */ (el.querySelector('[data-griff]'));
		const g = griff.getBoundingClientRect();
		return {
			spur: `${spur.width} × ${spur.height}`,
			mitte: `${g.left + g.width / 2 - spur.left}, ${g.top + g.height / 2 - spur.top}`,
			groesse: g.width,
			farbe: getComputedStyle(griff).backgroundColor
		};
	});

/**
 * Der Wert einer Farbrolle so, wie der Browser ihn als Hintergrund ausgibt.
 * @param {import('@playwright/test').Page} page
 * @param {string} name
 */
const rolle = (page, name) =>
	page.evaluate((n) => {
		const probe = document.createElement('div');
		probe.style.backgroundColor = `var(${n})`;
		document.body.append(probe);
		const wert = getComputedStyle(probe).backgroundColor;
		probe.remove();
		return wert;
	}, name);

test('Der Griff sitzt, wächst und färbt sich wie in Material 3', async ({ page }) => {
	await uiLogin(page);
	await gehZu(page, '/bestellungen');
	const schalter = page.getByRole('switch', { name: 'Dunkles Design' });
	await expect(schalter).toHaveAttribute('aria-checked', 'false');
	const ausUeberfahren = await rolle(page, '--color-on-surface-variant');

	expect(await miss(schalter)).toMatchObject({ spur: '52 × 32', mitte: '16, 16', groesse: 16 });

	await schalter.hover();
	expect((await miss(schalter)).farbe, 'aus, überfahren').toBe(ausUeberfahren);
	await page.mouse.down();
	expect(await miss(schalter), 'aus, gedrückt').toMatchObject({ mitte: '16, 16', groesse: 28 });
	await page.mouse.up();

	await expect(schalter).toHaveAttribute('aria-checked', 'true');
	// Erst nach dem Umschalten lesen: primary-container hat im dunklen Schema einen eigenen Wert.
	const anUeberfahren = await rolle(page, '--color-primary-container');
	expect((await miss(schalter)).farbe, 'an, überfahren').toBe(anUeberfahren);

	await page.mouse.move(0, 0);
	expect(await miss(schalter), 'an').toMatchObject({ mitte: '36, 16', groesse: 24 });

	await schalter.hover();
	await page.mouse.down();
	expect(await miss(schalter), 'an, gedrückt').toMatchObject({ mitte: '36, 16', groesse: 28 });
	await page.mouse.up();
});
