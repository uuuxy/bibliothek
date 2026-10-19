import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

// Escape an der Theke schließt die Akte. Gehört die Taste einem Feld, einer offenen Liste oder
// einem Dialog, tut sie dort ihre Sache und schließt die Akte nicht mit: Vorher war mit einem
// Tastendruck die offene Maske „Stammdaten bearbeiten" samt Eingaben weg. Hält die Maske
// Ungespeichertes, fragt Escape vorher.
const s = uniqueSuffix().slice(0, 6).toUpperCase();
const VORNAME = `Thekesc${s}`;
const AUSWEIS = `S-TE-${s}`;

/** @param {import('@playwright/test').Page} page */
async function oeffneMaskeAnDerTheke(page) {
	await uiLogin(page);
	await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
	await scanneWieScanner(page, AUSWEIS);
	await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	await expect(page.locator('#ort')).toHaveValue('Friedrichsdorf');
}

/** @param {import('@playwright/test').Page} page */
const maske = (page) => page.getByRole('heading', { name: 'Schüler bearbeiten' });
/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });
// Ein Klick auf die Überschrift nimmt den Fokus aus den Feldern, wie ein Klick neben sie.
/** @param {import('@playwright/test').Page} page */
const fokusAusDenFeldern = (page) => maske(page).click();

test.describe.serial('Theke: Escape bei offener Stammdaten-Maske', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
				strasse, hausnummer, plz, ort, eltern_email)
			VALUES ('${VORNAME}', 'E2E', '07G', '${AUSWEIS}', 2031, '2013-05-06',
				'Altweg', '1', '61381', 'Friedrichsdorf', 'alt@example.org');`);
	});
	test.afterAll(() => {
		seedSQL(`DELETE FROM leser WHERE barcode_id = '${AUSWEIS}';`);
	});

	test('Escape in einem Feld lässt Akte und Eingabe stehen', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');
		await page.keyboard.press('Escape');

		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');
		await expect(rueckfrage(page)).toHaveCount(0);
	});

	test('Escape in der offenen Klassenliste schließt nur die Liste', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');
		await page.locator('#klasse').click();
		await expect(page.getByRole('listbox')).toBeVisible();
		await page.keyboard.press('Escape');

		await expect(page.getByRole('listbox')).toHaveCount(0);
		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');
	});

	test('Escape neben den Feldern fragt; Escape in der Frage lässt die Eingabe stehen', async ({
		page
	}) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');
		await fokusAusDenFeldern(page);

		await page.keyboard.press('Escape');
		await expect(rueckfrage(page)).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(rueckfrage(page), 'Escape schließt nur die Frage').toHaveCount(0);
		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');

		await fokusAusDenFeldern(page);
		await page.keyboard.press('Escape');
		await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');

		await fokusAusDenFeldern(page);
		await page.keyboard.press('Escape');
		await rueckfrage(page).getByRole('button', { name: 'Verwerfen und weiter' }).click();
		await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
		expect(querySQL(`SELECT ort FROM leser WHERE barcode_id = '${AUSWEIS}'`)).toBe(
			'Friedrichsdorf'
		);
	});

	test('ohne Ungespeichertes schließt Escape die Akte ohne Frage', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await fokusAusDenFeldern(page);
		await page.keyboard.press('Escape');

		await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
		await expect(rueckfrage(page)).toHaveCount(0);
	});
});
