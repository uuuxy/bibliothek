import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, menuepunkt, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// „Zurück" in „Stammdaten bearbeiten" verwirft, was nicht gespeichert ist. Vorher fragt die
// Maske, wie die Buchmaske (e2e/buchmaske-verwerfen.spec.js) und wie Material 3 es für den
// Full-screen dialog verlangt („Dismissing"); ohne Eingabe fragt sie nicht. Ebenso fragen der
// Wechsel der Seite und das Schließen des Fensters (docs/OFFEN.md 5.80).
const s = uniqueSuffix().slice(0, 6);
const VORNAME = `Verwerf${s}`;
const AUSWEIS = `S-VW-${s}`;

/** @param {import('@playwright/test').Page} page */
async function oeffneMaske(page) {
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(VORNAME);
	await page.getByRole('button', { name: new RegExp(`Profil von ${VORNAME} `) }).click();
	await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	await expect(page.locator('#ort')).toHaveValue('Friedrichsdorf');
}

/** @param {import('@playwright/test').Page} page */
const zurueck = (page) => page.getByRole('button', { name: 'Zurück', exact: true });
/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });
const ortInDatenbank = () => querySQL(`SELECT ort FROM leser WHERE barcode_id = '${AUSWEIS}'`);

test.describe.serial('Stammdaten bearbeiten: Zurück mit Ungespeichertem', () => {
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

	test('fragt vor dem Verwerfen; „Bleiben" lässt die Eingabe stehen', async ({ page }) => {
		await oeffneMaske(page);
		await page.locator('#ort').fill('Bad Homburg');
		await zurueck(page).click();

		await expect(rueckfrage(page)).toBeVisible();
		await expect(
			rueckfrage(page).getByRole('button', { name: 'Bleiben' }),
			'Enter verwirft nichts'
		).toBeFocused();
		await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
		await expect(rueckfrage(page)).toHaveCount(0);
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');

		await zurueck(page).click();
		await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
		await expect(page.getByRole('button', { name: 'Bearbeiten' })).toBeVisible();
		expect(ortInDatenbank()).toBe('Friedrichsdorf');
	});

	test('ohne Eingabe schließt „Zurück" ohne Frage', async ({ page }) => {
		await oeffneMaske(page);
		await zurueck(page).click();
		await expect(page.getByRole('button', { name: 'Bearbeiten' })).toBeVisible();
		await expect(rueckfrage(page)).toHaveCount(0);
	});

	test('der Wechsel der Seite fragt ebenso', async ({ page }) => {
		await oeffneMaske(page);
		await page.locator('#ort').fill('Bad Homburg');

		await menuepunkt(page, 'Ausleihe').click();
		await expect(rueckfrage(page)).toBeVisible();
		await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
		await expect(page).toHaveURL(/\/schuelerdatei/);
		await expect(page.locator('#ort'), 'die Eingabe steht noch').toHaveValue('Bad Homburg');

		await menuepunkt(page, 'Ausleihe').click();
		await rueckfrage(page).getByRole('button', { name: 'Verwerfen und weiter' }).click();
		await expect(page).toHaveURL(/\/kiosk$/);
		expect(ortInDatenbank()).toBe('Friedrichsdorf');
	});

	test('das Schließen des Fensters fragt der Browser, solange etwas ungespeichert ist', async ({
		page
	}) => {
		await oeffneMaske(page);
		await page.locator('#ort').fill('Bad Homburg');
		/** @type {string[]} */
		const dialoge = [];
		page.on('dialog', (dialog) => {
			dialoge.push(dialog.type());
			void dialog.dismiss();
		});
		await page.close({ runBeforeUnload: true });
		await expect.poll(() => dialoge).toEqual(['beforeunload']);
	});
});
