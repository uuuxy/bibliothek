import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// „Stammdaten bearbeiten" schickt die Felder, die seit dem Laden der Akte geändert wurden, und
// der Server schreibt nur diese. Was die Versetzung, ein LUSD-Import oder ein anderer Platz
// inzwischen an derselben Zeile geändert hat, bleibt stehen. Vorher gingen alle Felder mit dem
// Stand vom Laden zurück, und die Klasse von vor der Versetzung stand wieder in der Zeile.
const s = uniqueSuffix().slice(0, 6);
const VORNAME = `Zweiplatz${s}`;
const AUSWEIS = `S-ZP-${s}`;

/** @param {string} spalte */
const steht = (spalte) =>
	querySQL(`SELECT coalesce(${spalte}::text, '') FROM leser WHERE barcode_id = '${AUSWEIS}'`);

/** @param {import('@playwright/test').Page} page */
async function oeffneMaske(page) {
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(VORNAME);
	await page.getByRole('button', { name: new RegExp(`Profil von ${VORNAME} `) }).click();
	await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	// Die Maske ist gefüllt, sobald die Eltern-Adresse dasteht.
	await expect(page.locator('#eltern_email')).toHaveValue(/@example\.org$/);
}

/**
 * Klickt „Speichern" und liefert, was die Maske dabei an den Server schickt.
 * @param {import('@playwright/test').Page} page
 */
async function speichern(page) {
	const anfrage = page.waitForRequest(
		(r) => r.method() === 'PATCH' && r.url().includes('/api/schueler/')
	);
	await page.getByRole('button', { name: 'Speichern' }).click();
	const rumpf = JSON.parse((await anfrage).postData() || 'null');
	// Nach dem Speichern schließt die Maske, die Akte steht wieder da.
	await expect(page.getByRole('button', { name: 'Bearbeiten' })).toBeVisible();
	return rumpf;
}

test.describe.serial('Leserakte: zwei Plätze, ein Leser', () => {
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

	test('was inzwischen an der Zeile geändert wurde, bleibt stehen', async ({ page }) => {
		await oeffneMaske(page);

		// Die Maske ist offen; die Versetzung und der Import schreiben an derselben Zeile.
		seedSQL(`
			UPDATE leser SET klasse = '08G', abgaenger_jahr = 2030, strasse = 'Neuweg'
			WHERE barcode_id = '${AUSWEIS}';`);

		await page.locator('#eltern_email').fill('neu@example.org');
		expect(await speichern(page)).toEqual({ eltern_email: 'neu@example.org' });

		expect(steht('eltern_email')).toBe('neu@example.org');
		expect(steht('klasse'), 'die Klasse aus der Versetzung').toBe('08G');
		expect(steht('abgaenger_jahr')).toBe('2030');
		expect(steht('strasse'), 'die Anschrift aus dem Import').toBe('Neuweg');
	});

	test('zwei Plätze ändern verschiedene Felder: beide Änderungen stehen', async ({ browser }) => {
		const platz1 = await (await browser.newContext()).newPage();
		const platz2 = await (await browser.newContext()).newPage();
		await oeffneMaske(platz1);
		await oeffneMaske(platz2);

		await platz2.locator('#ort').fill('Bad Homburg');
		expect(await speichern(platz2)).toEqual({ ort: 'Bad Homburg' });

		await expect(platz1.locator('#ort')).toHaveValue('Friedrichsdorf');
		await platz1.locator('#plz').fill('61348');
		expect(await speichern(platz1)).toEqual({ plz: '61348' });

		expect(steht('ort')).toBe('Bad Homburg');
		expect(steht('plz')).toBe('61348');
		expect(steht('klasse')).toBe('08G');
	});
});
