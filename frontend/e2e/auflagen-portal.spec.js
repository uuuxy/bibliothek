import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Mein Portal → „Schulbücher" (docs/OFFEN.md 4.18, Stufe 6): Ein Buch in zwei Auflagen ist
// eine Kachel, die Fach-Karte zählt es als einen Titel, und die Kachel sagt, woraus die Zahl
// besteht. Nach der ISBN der alten Auflage gesucht, bleibt es ein Buch mit allen Exemplaren.
const LEHRER_EMAIL = 'e2e-lehrer-auflagen@test.local';

test('Portal Schulbücher: ein Buch in zwei Auflagen ist eine Kachel mit der Summe', async ({
	page
}) => {
	const s = uniqueSuffix().slice(0, 8);
	const FACH = `E2E-Auflagenfach ${s}`;
	const TITEL = `Portal-Auflagen ${s}`;
	seedSQL(`
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('E2E', 'Auflagen', '${LEHRER_EMAIL}', 'kollegium', true)
		ON CONFLICT (email) DO UPDATE SET aktiv = true;
		INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('E2EA${s}', '${FACH}');
		WITH w AS (INSERT INTO werke DEFAULT VALUES RETURNING id),
		alt AS (INSERT INTO buecher_titel (isbn, titel, subject, ist_lernmittel, auflage, erscheinungsjahr, werk_id)
			SELECT 'PA1${s}', '${TITEL}', '${FACH}', true, '3. Aufl.', 2019, id FROM w RETURNING id),
		neu AS (INSERT INTO buecher_titel (isbn, titel, subject, ist_lernmittel, auflage, erscheinungsjahr, werk_id)
			SELECT 'PA2${s}', '${TITEL}', '${FACH}', true, '4. Aufl.', 2023, id FROM w RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, 'PA-${s}-1', true FROM alt UNION ALL SELECT id, 'PA-${s}-2', true FROM alt
		UNION ALL SELECT id, 'PA-${s}-3', true FROM neu;
	`);

	try {
		await uiLogin(page, LEHRER_EMAIL);
		await page.getByTitle('Mein Portal').click();
		await page.getByRole('tab', { name: 'Schulbücher' }).click();

		const karte = page.getByRole('button', { name: new RegExp(FACH) });
		await expect(karte).toContainText('3 Exemplare');
		await expect(karte).toContainText('1 Titel');
		await karte.click();
		const raster = page.getByTestId('schulbuecher-raster');
		await expect(raster.locator('h3')).toHaveCount(1);
		await expect(raster.getByTestId('schulbuch-auflagen')).toHaveText(
			'Bestand aus 2 Auflagen: 4. Aufl. · 2023 (1), 3. Aufl. · 2019 (2)'
		);

		await page.getByLabel('Schulbücher durchsuchen').fill(`PA1${s}`);
		await expect(karte).toContainText('3 Exemplare');
		await expect(karte).toContainText('1 Titel');
		await expect(raster.locator('h3')).toHaveCount(1);
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (SELECT werk_id FROM buecher_titel WHERE titel = '${TITEL}');
			DELETE FROM buecher_titel WHERE titel = '${TITEL}';
			DELETE FROM systematik_kategorien WHERE bezeichnung = '${FACH}';
		`);
	}
});
