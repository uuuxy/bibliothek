import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, csrfToken, uniqueSuffix } from './helpers.js';

// Etikettendruck — die Brücke zur physischen Welt: ohne funktionierende
// Barcode-Bögen kann das Sekretariat keine neuen Bücher auszeichnen.
// Beide Server-Pfade als PDF-Smoke: der Titel-Bogen aus dem Buchformular
// ("Barcodes drucken") und der freie Druck über /api/print/labels.

test('Etiketten: Titel-Bogen und freier Labeldruck liefern echte PDFs', async ({ page }) => {
    const s = uniqueSuffix();

    seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978e${s}', 'E2E Etikettenbuch ${s}', 'Druck Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, b, true FROM t, unnest(ARRAY['B-eti1-${s}', 'B-eti2-${s}']) AS b;
    `);
    const titelId = querySQL(`SELECT id FROM buecher_titel WHERE isbn = '978e${s}'`);

    await uiLogin(page);

    // 1. Titel-Etikettenbogen (Button "Barcodes drucken" im Buchformular)
    const bogen = await page.request.get(`/api/buecher/titel/${titelId}/etiketten`);
    expect(bogen.status()).toBe(200);
    expect(bogen.headers()['content-type']).toContain('application/pdf');
    expect((await bogen.body()).length).toBeGreaterThan(1000);

    // 2. Freier Labeldruck (A4 Zweckform, wie ihn die Etiketten-Verwaltung nutzt)
    const token = await csrfToken(page);
    const labels = await page.request.post('/api/print/labels', {
        headers: { 'X-CSRF-Token': token },
        data: {
            formatId: 'zweckform_l4760',
            startPosition: 0,
            isQR: false,
            items: [
                { BarcodeID: `B-eti1-${s}`, Titel: `E2E Etikettenbuch ${s}`, Autor: 'Druck Autor', ISBN: `978e${s}` },
                { BarcodeID: `B-eti2-${s}`, Titel: `E2E Etikettenbuch ${s}`, Autor: 'Druck Autor', ISBN: `978e${s}` },
            ],
        },
    });
    expect(labels.status()).toBe(200);
    expect(labels.headers()['content-type']).toContain('application/pdf');
    expect((await labels.body()).length).toBeGreaterThan(1000);
});
