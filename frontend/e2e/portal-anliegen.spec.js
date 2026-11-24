import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt } from './helpers.js';

// Portal, „Problem melden": Am Treffer der Suche ist das Buch gewählt, sein Titel steht in
// der Meldung. Ohne Buch steht der Knopf, wenn die Suche nichts findet, und das Formular fragt,
// worum es geht. Einen Buchwunsch und einen Reiter „Meine Anliegen" gibt es nicht mehr; in der Liste
// der Bibliothek steht die Meldung über einem älteren Wunsch.
const LEHRER = 'e2e-anliegen-lehrer@test.local';
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Anliegenbuch ${s}`;
const BESCHREIBUNG = `falsche Auflage, E2E ${s}`;
const OHNE_BUCH = `Bücher der 8G3, E2E ${s}`;

test.describe.serial('Portal: Problem melden', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ('E2E', 'Anliegen', '${LEHRER}', 'kollegium', true) ON CONFLICT (email) DO UPDATE SET aktiv = true;
			WITH t AS (INSERT INTO buecher_titel (isbn, titel, autor, ist_lernmittel) VALUES ('978an${s}', '${TITEL}', 'Autorin ${s}', true) RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) SELECT id, 'AN-EX-${s}', true FROM t;
			INSERT INTO lehrer_anliegen (art, titel_text, klasse, kommentar, erstellt_am)
			VALUES ('wunsch', 'Alter Wunsch ${s}', '7A', 'E2E ${s}', now() - interval '30 days');
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM lehrer_anliegen WHERE kommentar LIKE '%E2E ${s}' OR titel_text LIKE '%E2E ${s}';
			DELETE FROM buecher_exemplare WHERE barcode_id = 'AN-EX-${s}';
			DELETE FROM buecher_titel WHERE isbn = '978an${s}';
		`);
	});

	test('Am Treffer: das Buch ist gewählt, die Beschreibung ist Pflicht', async ({ page }) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();
		await page.getByRole('searchbox', { name: 'Titel für einen Klassensatz suchen' }).fill(TITEL);
		await expect(page.getByRole('heading', { name: TITEL })).toBeVisible();

		const melden = page.getByRole('button', { name: `Problem melden zu ${TITEL}` });
		await melden.click();
		// Das Buch wird nicht noch einmal getippt.
		await expect(page.getByLabel('Worum geht es?*')).toHaveCount(0);
		await expect(page.getByLabel('Klasse oder Kurs', { exact: true })).toBeFocused();

		const absenden = page.getByRole('button', { name: 'Absenden' });
		await expect(absenden, 'ohne Beschreibung des Problems').toBeDisabled();
		await page.getByLabel('Klasse oder Kurs', { exact: true }).fill('8G3');
		await page.getByLabel('Was stimmt nicht?*').fill(BESCHREIBUNG);
		await absenden.click();

		await expect
			.poll(() =>
				querySQL(
					`SELECT art || '|' || titel_text || '|' || klasse FROM lehrer_anliegen WHERE kommentar = '${BESCHREIBUNG}'`
				)
			)
			.toBe(`meldung|${TITEL}|8G3`);
		// Das Formular ist zu, der Fokus steht wieder auf dem Knopf am Treffer.
		await expect(absenden).toHaveCount(0);
		await expect(melden).toBeFocused();
	});

	test('Ohne Treffer: der Knopf steht unter „Keine Titel gefunden", das Formular übernimmt das Gesuchte', async ({
		page
	}) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();

		// Vier Reiter, keiner für Anliegen, und kein Buchwunsch.
		await expect(page.getByRole('tab')).toHaveText([
			'Reservieren & Melden',
			'Klassensätze',
			'Schulbücher',
			'LMF-Plan'
		]);
		await expect(page.getByRole('button', { name: 'Buchwunsch' })).toHaveCount(0);
		// Unter dem Suchfeld steht kein Knopf; den Weg nennt der Platzhalter.
		const suchfeld = page.getByRole('searchbox', { name: 'Titel für einen Klassensatz suchen' });
		await expect(suchfeld).toHaveAttribute(
			'placeholder',
			'Titel suchen für Reservierung oder Meldung …'
		);
		const knopf = page.getByRole('button', { name: 'Problem melden', exact: true });
		await expect(knopf, 'vor der Suche').toHaveCount(0);

		await suchfeld.fill(OHNE_BUCH);
		await expect(page.getByText('Keine Titel gefunden')).toBeVisible();
		await knopf.click();
		const worum = page.getByLabel('Worum geht es?*');
		await expect(worum).toBeFocused();
		await expect(worum).toHaveValue(OHNE_BUCH);
		// Das Formular klappt unter dem Knopf auf, in derselben Mitte wie der leere Zustand.
		const knopfLage = await knopf.boundingBox();
		const feldLage = await worum.boundingBox();
		if (!knopfLage || !feldLage) throw new Error('Knopf oder Feld ohne Lage');
		expect(feldLage.y, 'das Feld steht unter dem Knopf').toBeGreaterThan(
			knopfLage.y + knopfLage.height
		);
		expect(
			Math.abs(knopfLage.x + knopfLage.width / 2 - (feldLage.x + feldLage.width / 2)),
			'Mitte von Knopf und Feld'
		).toBeLessThanOrEqual(1);
		const absenden = page.getByRole('button', { name: 'Absenden' });
		await expect(absenden, 'ohne Beschreibung des Problems').toBeDisabled();
		await page.getByLabel('Was stimmt nicht?*').fill('drei fehlen');
		await absenden.click();

		await expect
			.poll(() =>
				querySQL(
					`SELECT art || '|' || kommentar FROM lehrer_anliegen WHERE titel_text = '${OHNE_BUCH}'`
				)
			)
			.toBe('meldung|drei fehlen');
		// Nach dem Absenden ist das Formular zu, und der Knopf hat den Fokus. Ohne Suche steht
		// die Meldung unter „Deine Meldungen".
		await expect(knopf).toBeFocused();
		await expect(worum).toHaveCount(0);
		await suchfeld.fill('');
		await expect(page.getByRole('heading', { name: 'Deine Meldungen' })).toBeVisible();
		await expect(page.getByText(OHNE_BUCH)).toBeVisible();
	});

	// Wer weitersucht, tippt im Suchfeld weiter: Das Formular ohne Buch schließt. Offen zöge es
	// beim nächsten leeren Ergebnis den Fokus auf sich, und das nächste Zeichen landete dort.
	test('Weitersuchen schließt das Formular ohne Buch, getippt wird im Suchfeld', async ({
		page
	}) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();
		const suchfeld = page.getByRole('searchbox', { name: 'Titel für einen Klassensatz suchen' });
		const getippt = `Scanner kaputt ${s}`;
		await suchfeld.fill(getippt);
		const knopf = page.getByRole('button', { name: 'Problem melden', exact: true });
		await knopf.click();
		await expect(page.getByLabel('Worum geht es?*')).toBeFocused();

		await suchfeld.focus();
		await page.keyboard.press('End');
		await page.keyboard.type(' x');
		await expect(knopf, 'das leere Ergebnis zur neuen Suche').toHaveAttribute(
			'aria-expanded',
			'false'
		);
		await expect(page.getByLabel('Worum geht es?*')).toHaveCount(0);
		await page.keyboard.type('y');
		await expect(suchfeld).toHaveValue(`${getippt} xy`);
		await expect(suchfeld).toBeFocused();
	});

	test('Die Bibliothek sieht die Meldung über dem älteren Wunsch', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/bestellungen');
		await page.getByRole('tab', { name: 'Meldungen', exact: true }).click();

		await expect(
			page.getByText('Meldungen der Lehrkräfte aus dem Kollegiums-Portal')
		).toBeVisible();
		await expect(page.getByRole('heading', { name: 'Wünsche', exact: true })).toBeVisible();
		const meldung = await page.getByText(TITEL, { exact: true }).boundingBox();
		const wunsch = await page.getByText(`Alter Wunsch ${s}`, { exact: true }).boundingBox();
		expect(meldung?.y, 'die Meldung von heute').toBeLessThan(wunsch?.y ?? 0);
	});
});
