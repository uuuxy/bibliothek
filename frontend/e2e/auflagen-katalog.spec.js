import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Suche zeigt ein Buch einmal (docs/OFFEN.md 4.18, Stufe 6; entschieden am 17.09.2026:
// „einen Treffer mit der Gesamtzahl und darunter die Aufschlüsselung je Auflage"). Zwei
// Auflagen desselben Buchs, die 3. mit zwei Exemplaren, die 4. mit einem. Nach dem Titel
// getroffen steht die neueste oben; nach der ISBN der alten Auflage steht diese oben — gezählt
// wird in beiden Fällen das ganze Buch.
test('Medienkatalog: ein Buch in zwei Auflagen ist eine Kachel mit Summe und Aufschlüsselung', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Katalog-Auflagen ${s}`;
	// buecher_titel.isbn ist varchar(20): die Nummern aus dem hinteren Teil der Kennung.
	const alteIsbn = `KAT1${s.slice(-12)}`;
	const neueIsbn = `KAT2${s.slice(-12)}`;
	seedSQL(`
		WITH w AS (INSERT INTO werke DEFAULT VALUES RETURNING id),
		alt AS (INSERT INTO buecher_titel (titel, isbn, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT '${titel}', '${alteIsbn}', '3. Aufl.', 2019, true, id FROM w RETURNING id),
		neu AS (INSERT INTO buecher_titel (titel, isbn, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT '${titel}', '${neueIsbn}', '4. Aufl.', 2023, true, id FROM w RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT id, 'B-KATA${s}' FROM alt UNION ALL SELECT id, 'B-KATB${s}' FROM alt
		UNION ALL SELECT id, 'B-KATN${s}' FROM neu;
	`);

	try {
		await uiLogin(page);
		await page.goto('/medienkatalog');
		const suche = page.locator('#katalog-suchfeld');
		const kacheln = page.getByRole('heading', { level: 2, name: titel });
		const karte = page.locator('div.m3-state').filter({ has: kacheln });

		await suche.fill(titel);
		await expect(kacheln).toHaveCount(1);
		await expect(karte.getByText(`ISBN: ${neueIsbn}`, { exact: true })).toBeVisible();
		await expect(karte.getByText('3 von 3 verfügbar', { exact: true })).toBeVisible();
		await expect(
			karte.getByText('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (1), 3. Aufl. · 2019 (2)', {
				exact: true
			})
		).toBeVisible();

		await suche.fill(alteIsbn);
		await expect(kacheln).toHaveCount(1);
		await expect(karte.getByText(`ISBN: ${alteIsbn}`, { exact: true })).toBeVisible();
		await expect(karte.getByText('3 von 3 verfügbar', { exact: true })).toBeVisible();
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (SELECT werk_id FROM buecher_titel WHERE titel = '${titel}');
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	}
});
