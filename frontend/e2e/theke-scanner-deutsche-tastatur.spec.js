import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, scanneWieScanner } from './helpers.js';

// Ein Handscanner mit amerikanischer Tastatur an einem Rechner mit deutscher: Die Taste, die er für
// den Bindestrich drückt, ergibt dort ein ß, und an der Theke kam „Sß10001" an. Gefunden wurde
// niemand, und die Theke bat, ein Ergebnis aus einer leeren Liste zu wählen.
const nummer = `S-9${String(Date.now()).slice(-8)}`;
const VORNAME = `Tastatur${nummer.slice(-4)}`;

test.describe.serial('Theke: Scanner mit amerikanischer Tastatur', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
			VALUES ('${VORNAME}', 'E2E', '07G', '${nummer}', 2031);`);
	});
	test.afterAll(() => {
		seedSQL(`DELETE FROM leser WHERE barcode_id = '${nummer}';`);
	});

	test('der Ausweis öffnet die Akte, auch wenn der Bindestrich als ß ankommt', async ({ page }) => {
		await uiLogin(page);
		await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();

		await scanneWieScanner(page, nummer, { deutscherRechner: true });

		await expect(page.getByRole('tab', { name: 'Stammdaten & Adresse' })).toBeVisible();
		await expect(page.getByText(`${VORNAME} E2E`).first()).toBeVisible();
	});
});
