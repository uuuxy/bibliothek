import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt } from './helpers.js';

// „Mein Portal" findet auch Schulbücher (Lernmittel) und Titel, deren Exemplare bestellt und
// noch nicht eingetroffen sind. Der öffentliche Katalog zeigt beides nicht; reserviert
// werden im Portal aber vor allem Schulbücher.
const LEHRER = 'e2e-schulbuch-lehrer@test.local';
const s = uniqueSuffix().slice(0, 6);
const SCHULBUCH = `Schulbuch Bio ${s}`;
const BESTELLT = `Bestellband Bio ${s}`;

test.describe.serial('Portal: Schulbücher und bestellte Titel in der Suche', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ('E2E', 'Schulbuch', '${LEHRER}', 'kollegium', true) ON CONFLICT (email) DO UPDATE SET aktiv = true;
			WITH t AS (INSERT INTO buecher_titel (isbn, titel, autor, ist_lernmittel) VALUES ('978sb${s}', '${SCHULBUCH}', 'Autorin ${s}', true) RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) SELECT t.id, 'SB-${s}-' || g, true FROM t, generate_series(1, 3) AS g;
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus) SELECT t.id, 'SBZ-${s}-' || g, false, 'bestellt' FROM buecher_titel t, generate_series(1, 2) AS g WHERE t.isbn = '978sb${s}';
			WITH t AS (INSERT INTO buecher_titel (isbn, titel, autor) VALUES ('978bb${s}', '${BESTELLT}', 'Autorin ${s}') RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus) SELECT t.id, 'BB-${s}-' || g, false, 'bestellt' FROM t, generate_series(1, 2) AS g;
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM klassensatz_reservierungen WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN ('978sb${s}', '978bb${s}'));
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'SB-${s}-%' OR barcode_id LIKE 'SBZ-${s}-%' OR barcode_id LIKE 'BB-${s}-%';
			DELETE FROM buecher_titel WHERE isbn IN ('978sb${s}', '978bb${s}');
		`);
	});

	test('Ein Schulbuch wird gefunden und lässt sich reservieren', async ({ page }) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();
		const suchfeld = page.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' });

		await suchfeld.fill(SCHULBUCH);
		await expect(page.getByRole('heading', { name: SCHULBUCH })).toBeVisible();
		await expect(page.getByText('3 von 3 verfügbar')).toBeVisible();

		await page.getByRole('button', { name: 'Klassensatz reservieren' }).click();
		await page.getByLabel('Klasse / Kurs *').fill(`8G${s}`);
		await page.getByLabel('Anzahl').fill('3');
		await page.getByRole('button', { name: 'Anfrage senden' }).click();
		await expect(page.getByText('✓ Gesendet')).toBeVisible();

		expect(
			querySQL(`
				SELECT r.klasse || '|' || r.anzahl FROM klassensatz_reservierungen r
				JOIN buecher_titel t ON t.id = r.titel_id WHERE t.isbn = '978sb${s}'
			`)
		).toBe(`8G${s}|3`);
	});

	// Die Obergrenze der Reservierung zählt die bestellten Exemplare mit; der Treffer nennt
	// sie deshalb neben dem Bestand im Haus.
	test('Sind zu einem Titel im Haus weitere bestellt, steht beides am Treffer', async ({
		page
	}) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();
		await page
			.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' })
			.fill(SCHULBUCH);

		await expect(page.getByText('3 von 3 verfügbar')).toBeVisible();
		await expect(page.getByText('2 bestellt', { exact: true })).toBeVisible();
	});

	test('Ein Titel, der nur bestellt ist, steht mit „bestellt" in der Liste', async ({ page }) => {
		await uiLogin(page, LEHRER);
		await menuepunkt(page, 'Mein Portal').click();
		await page
			.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' })
			.fill(BESTELLT);

		await expect(page.getByRole('heading', { name: BESTELLT })).toBeVisible();
		await expect(page.getByText('2 bestellt', { exact: true })).toBeVisible();
		await expect(page.getByText(/nicht verfügbar/)).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Klassensatz reservieren' })).toBeVisible();
	});

	test('Der öffentliche Katalog zeigt beide nicht', async ({ page }) => {
		const res = await page.request.get(
			`/api/public/opac/suche?q=${encodeURIComponent(`Bio ${s}`)}`
		);
		expect(res.status()).toBe(200);
		expect(await res.json()).toEqual([]);
	});
});
