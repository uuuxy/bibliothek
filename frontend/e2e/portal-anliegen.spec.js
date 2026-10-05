import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Portal „Meine Anliegen": Die Art wird gewählt und nie vorbelegt. Beim Problem kommt das
// Buch aus den Vorschlägen des Katalogs, und in der Liste der Bibliothek steht die Meldung
// über den Wünschen, auch wenn der Wunsch älter ist.
const LEHRER = 'e2e-anliegen-lehrer@test.local';
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Anliegenbuch ${s}`;
const BESCHREIBUNG = `falsche Auflage, E2E ${s}`;

test.describe.serial('Portal: Wunsch und Problem', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ('E2E', 'Anliegen', '${LEHRER}', 'kollegium', true) ON CONFLICT (email) DO UPDATE SET aktiv = true;
			WITH t AS (INSERT INTO buecher_titel (isbn, titel, autor) VALUES ('978an${s}', '${TITEL}', 'Autorin ${s}') RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) SELECT id, 'AN-EX-${s}', true FROM t;
			INSERT INTO lehrer_anliegen (art, titel_text, klasse, kommentar, erstellt_am)
			VALUES ('wunsch', 'Alter Wunsch ${s}', '7A', 'E2E ${s}', now() - interval '30 days');
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM lehrer_anliegen WHERE kommentar LIKE '%E2E ${s}';
			DELETE FROM buecher_exemplare WHERE barcode_id = 'AN-EX-${s}';
			DELETE FROM buecher_titel WHERE isbn = '978an${s}';
		`);
	});

	test('Problem melden: keine Vorbelegung, das Buch kommt aus den Vorschlägen', async ({
		page
	}) => {
		await uiLogin(page, LEHRER);
		await page.getByTitle('Mein Portal').click();
		await page.getByRole('tab', { name: 'Meine Anliegen' }).click();

		// Vor der Wahl steht kein Formular da.
		await expect(page.getByRole('button', { name: 'Buchwunsch' })).toBeVisible();
		await expect(page.getByLabel('Welches Buch?')).toHaveCount(0);

		await page.getByRole('button', { name: 'Problem melden' }).click();
		const buch = page.getByRole('combobox', { name: 'Welches Buch?' });
		await expect(buch).toBeFocused();

		// Gesucht wird über den Namen der Autorin, damit der Titel im Feld aus dem Vorschlag
		// stammt und nicht aus der Eingabe.
		await buch.fill(`Autorin ${s}`);
		const vorschlag = page.getByRole('option', { name: new RegExp(TITEL) });
		await expect(vorschlag).toBeVisible();
		await vorschlag.click();
		await expect(buch).toHaveValue(TITEL);
		await expect(page.getByRole('listbox')).toHaveCount(0);

		const absenden = page.getByRole('button', { name: 'Absenden' });
		await expect(absenden, 'ohne Beschreibung des Problems').toBeDisabled();
		await page.getByLabel('Klasse / Kurs').fill('8G3');
		await page.getByLabel('Was stimmt nicht?').fill(BESCHREIBUNG);
		await absenden.click();

		await expect
			.poll(() =>
				querySQL(
					`SELECT art || '|' || titel_text || '|' || klasse FROM lehrer_anliegen WHERE kommentar = '${BESCHREIBUNG}'`
				)
			)
			.toBe(`meldung|${TITEL}|8G3`);
		// Nach dem Absenden steht wieder die Wahl da.
		await expect(page.getByRole('button', { name: 'Problem melden' })).toBeVisible();
	});

	test('Die Bibliothek sieht die Meldung über dem älteren Wunsch', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/bestellungen');
		await page.getByRole('tab', { name: /Wünsche & Meldungen/ }).click();

		await expect(page.getByRole('heading', { name: 'Meldungen', exact: true })).toBeVisible();
		await expect(page.getByRole('heading', { name: 'Wünsche', exact: true })).toBeVisible();
		const meldung = await page.getByText(TITEL, { exact: true }).boundingBox();
		const wunsch = await page.getByText(`Alter Wunsch ${s}`, { exact: true }).boundingBox();
		expect(meldung?.y, 'die Meldung von heute').toBeLessThan(wunsch?.y ?? 0);
	});
});
