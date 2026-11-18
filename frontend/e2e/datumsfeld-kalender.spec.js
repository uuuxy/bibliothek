import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, einstellungsKategorie } from './helpers.js';

// Das Datumsfeld nach M3 (Docked date picker) im echten Browser, im Dialog „Leser anlegen“:
// Was die Komponententests in jsdom nicht sehen, ist die Lage (der Kalender steht mit
// position: fixed in einem Dialog mit Schleier), der Fokus nach einem Klick (Safari setzt ihn
// nicht auf Knöpfe) und Escape zwischen Kalender, Dialog und Router.

/** @param {string} iso */
const deutsch = (iso) => `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`;

test('Datumsfeld im Dialog: Kalender zuoberst im Fenster, Tastatur, Klick, Escape schließt nur ihn', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	await page.getByRole('button', { name: 'Leser anlegen', exact: true }).click();
	const dialog = page.getByRole('dialog').first();
	const feld = dialog.getByLabel('Geburtsdatum');
	const knopf = dialog.getByRole('button', { name: 'Kalender öffnen' });
	const kalender = page.getByRole('dialog', { name: 'Datum wählen' });

	await knopf.click();
	await expect(kalender).toBeVisible();
	const lage = await kalender.evaluate((el) => {
		const r = el.getBoundingClientRect();
		const mitte = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
		return {
			zuoberst: el.contains(mitte),
			imFenster: r.top >= 0 && r.left >= 0 && r.bottom <= innerHeight && r.right <= innerWidth
		};
	});
	expect(lage, 'Der Kalender liegt ganz im Fenster und über dem Dialog').toEqual({
		zuoberst: true,
		imFenster: true
	});

	// Tastatur: Der Fokus steht auf dem markierten Tag, Pfeil rechts und Enter übernehmen den nächsten.
	const tag = kalender.locator('[data-aktiv]');
	await expect(tag).toBeFocused();
	const heute = /** @type {string} */ (await tag.getAttribute('data-tag'));
	await page.keyboard.press('ArrowRight');
	const morgen = /** @type {string} */ (
		await kalender.locator('[data-aktiv]').getAttribute('data-tag')
	);
	expect(morgen > heute, 'Pfeil rechts geht einen Tag weiter').toBe(true);
	await page.keyboard.press('Enter');
	await expect(kalender).toHaveCount(0);
	await expect(feld).toHaveValue(deutsch(morgen));
	await expect(knopf, 'Der Fokus kehrt zum Knopf zurück').toBeFocused();

	// Escape schließt den Kalender, nicht den Dialog, und die Seite bleibt.
	await knopf.click();
	await expect(kalender).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(kalender).toHaveCount(0);
	await expect(dialog).toBeVisible();
	await expect(page).toHaveURL(/\/schuelerdatei$/);
	await expect(feld).toHaveValue(deutsch(morgen));

	// Maus: einen Tag anklicken, mit OK übernehmen. Der 15. steht genau einmal auf dem Blatt; „OK“
	// mit exact, sonst träfe es im Oktober jeden Tag („Ok-tober“).
	await knopf.click();
	const fuenfzehnter = kalender.locator('button[data-tag$="-15"]');
	const gewaehlt = /** @type {string} */ (await fuenfzehnter.getAttribute('data-tag'));
	await fuenfzehnter.click();
	await kalender.getByRole('button', { name: 'OK', exact: true }).click();
	await expect(feld).toHaveValue(deutsch(gewaehlt));

	// Ein Klick neben den Kalender schließt ihn, der Dialog bleibt. Links vom Feld liegt der
	// Innenabstand des Dialogs, dorthin reicht der Kalender nie; auf halber Höhe, weil die Ecken
	// rund sind und ein Klick dort den Schleier träfe.
	await knopf.click();
	await expect(kalender).toBeVisible();
	const flaeche = /** @type {{ x: number, y: number, height: number }} */ (
		await dialog.boundingBox()
	);
	await page.mouse.click(flaeche.x + 8, flaeche.y + flaeche.height / 2);
	await expect(kalender).toHaveCount(0);
	await expect(dialog).toBeVisible();
	await expect(feld).toHaveValue(deutsch(gewaehlt));
});

// Ohne fokussierbaren Vorfahren, etwa über dem Inhalt und in Safari auch auf Knöpfen und Links,
// fällt der Fokus auf body, und focusout sieht kein Ziel: Dann schließt der Kalender über pointerdown.
test('Datumsfeld auf der Seite: Ein Klick auf freie Fläche über dem Inhalt schließt den Kalender', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'LUSD & Versetzung').click();
	const bereich = page.getByTestId('sommerferien');
	await bereich.getByRole('button', { name: 'Kalender öffnen' }).first().click();
	const kalender = page.getByRole('dialog', { name: 'Datum wählen' });
	await expect(kalender).toBeVisible();
	const punkt = await page.evaluate(() => {
		const inhalt = /** @type {HTMLElement} */ (
			document.getElementById('hauptinhalt')
		).getBoundingClientRect();
		const x = inhalt.left + 40;
		const y = Math.max(2, inhalt.top - 6);
		const ziel = document.elementFromPoint(x, y);
		return ziel && !ziel.closest('a, button, input, [tabindex]') ? { x, y } : null;
	});
	expect(punkt, 'Über dem Inhalt liegt freie Fläche').not.toBeNull();
	const { x, y } = /** @type {{ x: number, y: number }} */ (punkt);
	await page.mouse.click(x, y);
	await expect(kalender).toHaveCount(0);
});
