import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Ein Lernmittel mit LMF-Nummer bekommt für ein neues Etikett eine kurze Nummer: Der Druck fragt
// nach, vergibt die Nummer aus barcode_seq und druckt mit ihr. Danach trägt das Exemplar sie,
// sein Etikett ist als gedruckt vermerkt, und das Protokoll nennt beide Nummern.

test('Etikett für ein Exemplar mit LMF-Nummer: Rückfrage, kurze Nummer, Druck', async ({
	page
}) => {
	const s = uniqueSuffix();
	const lmf = `LMF-978${s}-001`;
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978l${s}', 'E2E Lernmittel ${s}', 'LMF Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, etikett_gedruckt)
        SELECT id, '${lmf}', true, true FROM t;
    `);

	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(`E2E Lernmittel ${s}`);
	await page.getByText(`E2E Lernmittel ${s}`).first().click();
	await page.getByRole('button', { name: 'Barcodes drucken', exact: true }).click();
	await expect(page.getByText('Exemplare auswählen: 1 von 1')).toBeVisible();

	await page.getByRole('button', { name: 'A4-Bogen drucken' }).click();
	const frage = page.getByRole('dialog', { name: 'Neue Nummer vergeben?' });
	await expect(frage).toContainText(lmf);
	await expect(frage).toContainText('nicht mehr scannen');

	const [druck] = await Promise.all([
		page.waitForRequest(
			(r) =>
				r.url().endsWith('/api/print/labels') &&
				r.method() === 'POST' &&
				!String(r.postData()).includes('LMF-')
		),
		frage.getByRole('button', { name: 'Neue Nummer vergeben' }).click()
	]);
	const neu = druck.postDataJSON().items[0].BarcodeID;
	expect(neu).toMatch(/^B-\d+$/);
	expect((await druck.response())?.status()).toBe(200);
	await expect(frage).toBeHidden();

	await expect
		.poll(() =>
			querySQL(`
                SELECT e.barcode_id || '|' || e.etikett_gedruckt
                FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
                WHERE t.isbn = '978l${s}'`)
		)
		.toBe(`${neu}|true`);
	expect(
		querySQL(`
            SELECT details->>'neue_nummer' FROM audit_log
            WHERE details->>'action' = 'neue_nummer' AND details->>'barcode_id' = '${lmf}'`)
	).toBe(neu);
});
