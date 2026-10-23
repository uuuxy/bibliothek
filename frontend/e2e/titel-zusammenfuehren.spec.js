import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Dasselbe Buch zweimal im Katalog: einmal aus einer Liste ohne ISBN, einmal aus dem
// Bestands-Import mit ISBN. „Mit anderem Titel zusammenführen" macht in der Maske einen Titel
// daraus: Der Titel mit ISBN bleibt, die Vorschau sagt in einer Zeile, wie er danach aussieht,
// er bekommt beide Exemplare und die Angaben, die ihm fehlten, und den anderen gibt es nicht mehr.

test('Titel zusammenführen: der Titel mit ISBN bleibt und hat danach alle Exemplare', async ({
	page
}) => {
	const s = uniqueSuffix();
	const isbn = `978w${s}`;
	seedSQL(`
        WITH liste AS (
            INSERT INTO buecher_titel (titel, autor) VALUES ('E2E Zwilling ${s} Liste', 'Austenfeld')
            RETURNING id
        ), import AS (
            INSERT INTO buecher_titel (titel, isbn, verlag) VALUES ('E2E Zwilling ${s}', '${isbn}', 'Cornelsen')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-zwl-${s}', true FROM liste
        UNION ALL SELECT id, 'B-zwi-${s}', true FROM import;
    `);

	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(`E2E Zwilling ${s} Liste`);
	await page.getByText(`E2E Zwilling ${s} Liste`).first().click();
	await expect(page.getByRole('heading', { name: 'Exemplare (1)' })).toBeVisible();

	await page.getByRole('button', { name: 'Mit anderem Titel zusammenführen' }).click();
	const dialog = page.getByRole('dialog', { name: 'Mit anderem Titel zusammenführen' });
	await dialog.getByLabel('Anderen Titel suchen').fill(`E2E Zwilling ${s}`);
	await dialog.getByRole('button', { name: new RegExp(isbn) }).click();

	await expect(dialog.getByText(`E2E Zwilling ${s} · ${isbn} · 2 Exemplare`)).toBeVisible();
	await expect(
		dialog.getByText(`„E2E Zwilling ${s} Liste“ (ohne ISBN) wird gelöscht.`)
	).toBeVisible();
	await dialog.getByRole('button', { name: 'Zusammenführen' }).click();

	await expect(dialog).toBeHidden();
	await expect(page.getByRole('heading', { name: 'Exemplare (2)' })).toBeVisible();
	await expect(page.locator('#buch-isbn')).toHaveValue(isbn);
	expect(
		querySQL(`
            SELECT count(*) FILTER (WHERE t.titel = 'E2E Zwilling ${s}' AND t.autor = 'Austenfeld')
                   || '|' || count(*)
            FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
            WHERE e.barcode_id IN ('B-zwl-${s}', 'B-zwi-${s}')`)
	).toBe('2|2');
	expect(
		querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = 'E2E Zwilling ${s} Liste'`)
	).toBe('0');
});
