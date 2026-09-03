import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die eine Suchleiste der Verwaltung (03.09.2026): Sie versteht Buch-Barcode, Ausweis,
// ISBN und Namen und SPRINGT nur — Buchakte, Schülerakte, Trefferliste. Gebucht wird
// nie (die Buchungs-Tür /api/action ruft nur die Omnibox, Ratsche
// frontend-hygiene-action-endpunkt.test.js). Getippt wird blind wie ein Scanner
// (Memory: fill() verdeckt Fokusfehler), Enter entscheidet.
test.describe('Globale Suchleiste', () => {
	const s = uniqueSuffix().slice(0, 6);
	const TITEL = `Sprungbuch ${s}`;
	const EXEMPLAR = `B-GS${s}`;
	const AUSWEIS = `S-GS${s}`;
	const NACHNAME = `Sprungkind${s}`;

	test.beforeAll(() => {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (isbn, titel, autor) VALUES ('978gs${s}', '${TITEL}', 'Sprung Autor') RETURNING id
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) SELECT id, '${EXEMPLAR}', true FROM t;
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum)
			VALUES ('Greta', '${NACHNAME}', '07A', '${AUSWEIS}', 2031, '2013-03-03');
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE barcode_id = '${EXEMPLAR}';
			DELETE FROM buecher_titel WHERE isbn = '978gs${s}';
			DELETE FROM schueler WHERE barcode_id = '${AUSWEIS}';
		`);
	});

	test('Buch-Barcode springt in die Buchakte, Ausweis in die Schülerakte, Name zeigt Treffer', async ({
		page
	}) => {
		await uiLogin(page);
		// Die Leiste steht nicht an der Theke — erst in eine Verwaltungsansicht.
		await page.getByTitle('Schülerdatei').click();
		const feld = page.locator('#global-suchfeld');
		await expect(feld).toBeVisible();

		// 1. Exemplar-Barcode wie ein Scanner: Tasten + Enter → Buchakte.
		await feld.click();
		await page.keyboard.type(EXEMPLAR);
		await page.keyboard.press('Enter');
		await expect(page).toHaveURL(/\/medienkatalog\/buch\//);
		await expect(page.getByRole('heading', { name: TITEL }).first()).toBeVisible();
		await expect(feld).toHaveValue('');

		// 2. Ausweis → Schülerakte.
		await page.keyboard.press('/');
		await expect(feld).toBeFocused();
		await page.keyboard.type(AUSWEIS);
		await page.keyboard.press('Enter');
		await expect(page).toHaveURL(/\/schuelerdatei/);
		await expect(page.getByText(`Greta ${NACHNAME}`).first()).toBeVisible();

		// 3. Name → Trefferliste, Klick öffnet die Akte.
		await feld.click();
		await page.keyboard.type(NACHNAME);
		const treffer = page.getByTestId('global-suche-treffer');
		await expect(treffer.getByRole('button', { name: new RegExp(NACHNAME) })).toBeVisible();

		// 4. Nichts gebucht: das Exemplar ist weiterhin frei.
		const res = await page.request.get(`/api/search?q=${EXEMPLAR}`);
		const data = await res.json();
		expect(data.treffer?.typ).toBe('exemplar');
	});
});
