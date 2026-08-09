import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL } from './helpers.js';

// Der "Bearbeiten"-Knopf war da — nur nicht zu sehen.
//
// Die Lieferantentabelle hat sieben Spalten und steht in einer Rasterzelle. Raster-Kinder
// haben min-width:auto und schrumpfen nicht unter ihren Inhalt: Die Tabelle schob sich aus
// der Zelle heraus, die Spalte "Aktionen" landete hinter dem sichtbaren Rand (gemessen bei
// 1280 px Fenster: rechte Kante des Knopfs bei 1620 px). Kein Fehler, keine Meldung — die
// Bearbeitung war schlicht unauffindbar.
//
// Was dieser Test NICHT prüfen darf: ob der Klick funktioniert. Playwright scrollt jedes
// Element vor dem Klick selbst ins Bild und meldet grün, während ein Mensch vor dem Schirm
// nichts sieht. Eine erste Fassung dieses Tests tat genau das und blieb beim Rückbau des
// Fixes grün — wertlos. Geprüft wird deshalb die rechte Kante gegen die Fensterbreite.
const LANGER_NAME = 'E2E-Bearbeitbar-Bildungshaus Verlagsgesellschaft Süd-West mbH & Co. KG';

test.afterEach(() => {
	// Aufräumen gehört zum Test: Ohne das wächst die Lieferantenliste mit jedem Lauf.
	// In der lokalen Stack-DB standen so 224 Testlieferanten gegen 3 echte — und genau
	// deren lange Namen haben die Tabelle über ihre Zelle hinausgeschoben.
	seedSQL(`DELETE FROM lieferanten WHERE name LIKE 'E2E-Bearbeitbar-%'`);
});

test('Lieferanten lassen sich bearbeiten, auch bei langem Namen', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 900 });
	await uiLogin(page);

	const angelegt = await apiPost(page, '/api/lieferanten', {
		name: LANGER_NAME,
		email: 'sehr.lange.adresse.des.lieferanten@bildungshaus-suedwest.example',
		customerNumber: 'K-BEARBEITBAR-0815'
	});
	expect(angelegt.ok()).toBe(true);

	await page.goto('/bestellungen');
	await page.getByRole('tab', { name: 'Lieferanten verwalten' }).click();
	await page.getByRole('table').waitFor();

	const sicht = await page.evaluate(() => {
		const tabelle = /** @type {HTMLElement} */ (document.querySelector('table'));
		const box = /** @type {HTMLElement} */ (tabelle.parentElement);
		const zelle = /** @type {HTMLElement} */ (box.parentElement);
		const knopf = [...document.querySelectorAll('button')].find(
			(b) => b.textContent?.trim() === 'Bearbeiten'
		);
		return {
			knopfRechts: knopf ? Math.round(knopf.getBoundingClientRect().right) : null,
			fenster: window.innerWidth,
			// Breite Tabellen scrollen in ihrer EIGENEN Box. Schiebt stattdessen die
			// Rasterzelle, wandert der Inhalt unbemerkt aus dem Bild.
			zelleScrolltQuer: zelle.scrollWidth > zelle.clientWidth + 1,
			boxScrollt: getComputedStyle(box).overflowX
		};
	});

	expect(
		sicht.knopfRechts,
		'"Bearbeiten" liegt rechts ausserhalb des Fensters'
	).toBeLessThanOrEqual(sicht.fenster);
	expect(sicht.zelleScrolltQuer, 'Die Tabelle schiebt ihre Rasterzelle auf').toBe(false);
	expect(sicht.boxScrollt, 'Die Tabelle braucht einen eigenen Scrollbereich').toMatch(
		/auto|scroll/
	);

	// Und der Knopf tut auch, was er soll.
	await page
		.getByRole('row')
		.filter({ hasText: LANGER_NAME })
		.getByRole('button', { name: 'Bearbeiten' })
		.click();

	// Im Bearbeitungsmodus wird der Name zum Eingabefeld — die Zeile trägt den Text dann
	// nicht mehr und ist über den Speichern-Knopf zu greifen (nur eine Zeile zugleich).
	const speichern = page.getByRole('button', { name: 'Speichern', exact: true });
	await expect(speichern).toBeVisible();
	const bearbeitungszeile = page.getByRole('row').filter({ has: speichern });
	await expect(bearbeitungszeile.getByRole('textbox').first()).toHaveValue(LANGER_NAME);
});
