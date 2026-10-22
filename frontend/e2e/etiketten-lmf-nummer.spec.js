import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Ein Lernmittel trägt eine kurze Nummer, die Nummer seines alten LMF-Etiketts steht daneben
// (lmf_barcode). Das Druck-Center zeigt die kurze Nummer und druckt sie ohne Rückfrage; danach ist
// das Etikett als gedruckt vermerkt.

test('Etikett für ein Lernmittel mit altem LMF-Etikett: kurze Nummer, ohne Rückfrage gedruckt', async ({
	page
}) => {
	const s = uniqueSuffix();
	const kurz = `B-LE${s}`;
	const altesEtikett = `LMF-978${s}-001`;
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978l${s}', 'E2E Lernmittel ${s}', 'LMF Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, lmf_barcode, ist_ausleihbar, etikett_gedruckt)
        SELECT id, '${kurz}', '${altesEtikett}', true, false FROM t;
    `);

	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(`E2E Lernmittel ${s}`);
	await page.getByText(`E2E Lernmittel ${s}`).first().click();
	await page.getByRole('button', { name: 'Barcodes drucken', exact: true }).click();
	await expect(page.getByText('Exemplare auswählen: 1 von 1')).toBeVisible();
	await expect(page.getByText(kurz).first()).toBeVisible();
	await expect(page.getByText(altesEtikett)).toHaveCount(0);

	const [druck] = await Promise.all([
		page.waitForRequest((r) => r.url().endsWith('/api/print/labels') && r.method() === 'POST'),
		page.getByRole('button', { name: 'A4-Bogen drucken' }).click()
	]);
	expect(druck.postDataJSON().items.map((/** @type {any} */ i) => i.BarcodeID)).toEqual([kurz]);
	expect((await druck.response())?.status()).toBe(200);
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect
		.poll(() =>
			querySQL(`SELECT etikett_gedruckt::text FROM buecher_exemplare WHERE barcode_id = '${kurz}'`)
		)
		.toBe('true');
});
