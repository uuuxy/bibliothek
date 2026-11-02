import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu, scanneWieScanner } from './helpers.js';

// Ein Handscanner mit amerikanischer Tastatur tippt an einem Rechner mit deutscher den
// Bindestrich als ß (FACHKONZEPT §1). Wer eine neue Karte in die Leserakte scannt, speichert die
// Nummer, die die Theke beim Scan nachschlägt, und nicht „Sß…": Die Karte fände sonst niemand.
const s = uniqueSuffix().slice(0, 6);
const VORNAME = `Kartenscan${s}`;
const NEU = `S-9${String(Date.now()).slice(-8)}`;

test.describe
	.serial('Leserakte: Ausweisnummer von einem Scanner mit amerikanischer Tastatur', () => {
	let id = '';
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum)
			VALUES ('${VORNAME}', 'E2E', '07G', 'S-KS-${s}', 2031, '2013-05-06');`);
		id = querySQL(`SELECT id FROM leser WHERE vorname = '${VORNAME}'`);
	});
	test.afterAll(() => {
		if (id) seedSQL(`DELETE FROM leser WHERE id = '${id}';`);
	});

	test('die gescannte Karte steht mit Bindestrich in der Leserzeile', async ({ page }) => {
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(VORNAME);
		await page.getByRole('button', { name: new RegExp(`Profil von ${VORNAME} `) }).click();
		await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
		await page.getByRole('button', { name: 'Bearbeiten' }).click();
		await expect(page.locator('#barcode')).toHaveValue(`S-KS-${s}`);

		await page.locator('#barcode').fill('');
		await scanneWieScanner(page, NEU, { deutscherRechner: true });
		await expect(page.locator('#barcode')).toHaveValue(NEU.replace('-', 'ß'));

		const anfrage = page.waitForRequest(
			(r) => r.method() === 'PATCH' && r.url().includes('/api/schueler/')
		);
		await page.getByRole('button', { name: 'Speichern' }).click();
		expect(JSON.parse((await anfrage).postData() || 'null')).toEqual({ barcode_id: NEU });
		await expect(page.getByRole('button', { name: 'Bearbeiten' })).toBeVisible();

		expect(querySQL(`SELECT barcode_id FROM leser WHERE id = '${id}'`)).toBe(NEU);
	});
});
