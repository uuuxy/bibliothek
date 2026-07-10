import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Settings-Enforcement: Eine Limit-Änderung muss beim NÄCHSTEN Checkout
// sofort greifen (der Checkout liest system_einstellungen pro Vorgang).
// Das Limit wird im finally auf den Ursprungswert zurückgesetzt —
// die Test-DB teilen sich alle Specs.
test('Ausleihlimit 1: zweiter Checkout blockt sofort', async ({ page }) => {
    await uiLogin(page);
    const suffix = uniqueSuffix();

    const vorher = querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = 'max_ausleihen_schueler'`);

    try {
        seedSQL(`
            INSERT INTO system_einstellungen (schluessel, wert)
            VALUES ('max_ausleihen_schueler', '1')
            ON CONFLICT (schluessel) DO UPDATE SET wert = '1';
        `);

        const created = await apiPost(page, '/api/schueler', {
            vorname: 'E2E',
            nachname: `Limit-${suffix}`,
            klasse: '7A',
            barcode_id: `S-${suffix}`,
        });
        expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

        seedSQL(`
            WITH t AS (
                INSERT INTO buecher_titel (titel)
                VALUES ('E2E-Limit1-${suffix}'), ('E2E-Limit2-${suffix}')
                RETURNING id, titel
            )
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-' || RIGHT(titel, LENGTH('Limit1-${suffix}')), true FROM t;
        `);

        await page.getByTitle('Ausleihe').click();
        const scanInput = page.getByPlaceholder(/scannen/i).first();
        await scanInput.fill(`S-${suffix}`);
        await scanInput.press('Enter');
        await expect(page.getByText(`Limit-${suffix}`).first()).toBeVisible();

        // Buch 1: geht durch
        await scanInput.fill(`B-Limit1-${suffix}`);
        await scanInput.press('Enter');
        await expect(page.getByText(`„E2E-Limit1-${suffix}" ausgeliehen an E2E.`)).toBeVisible();

        // Buch 2: Limit von 1 erreicht → sofortiger Block mit klarer Meldung
        await scanInput.fill(`B-Limit2-${suffix}`);
        await scanInput.press('Enter');
        await expect(page.getByText(/Ausleihlimit von 1 Büchern überschritten/).first()).toBeVisible();
        await expect(page.getByText('ENTLIEHENE BÜCHER (1)')).toBeVisible();
    } finally {
        if (vorher) {
            seedSQL(`UPDATE system_einstellungen SET wert = '${vorher}' WHERE schluessel = 'max_ausleihen_schueler';`);
        } else {
            seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = 'max_ausleihen_schueler';`);
        }
    }
});
