import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Inventur-Ablauf: starten (Signatur-Scope!) → scannen → abschließen.
// WICHTIG: Der Test nutzt bewusst NUR den Signatur-Scope — ein globaler
// Lauf würde auf einer geteilten DB alle nicht gescannten Exemplare als
// verloren aussondern. Der Signatur-Scope markiert nur Titel der eigenen
// Test-Signatur als 'ausstehend'; finish trifft nur diese.
test('Inventur: Signatur-Scope, gescannt bleibt, ungescannt wird Verlust', async ({ page }) => {
    await uiLogin(page);
    const suffix = uniqueSuffix();
    const sigName = `E2E-INV-${suffix}`;

    try {
        seedSQL(`
            WITH sig AS (
                INSERT INTO signatures (name) VALUES ('${sigName}') RETURNING id
            ), t AS (
                INSERT INTO buecher_titel (titel, signature_id)
                SELECT 'E2E-Inventurbuch-${suffix}', id FROM sig RETURNING id
            )
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, b, true FROM t, unnest(ARRAY['B-INVA-${suffix}', 'B-INVB-${suffix}']) AS b;
        `);

        await page.getByTitle('Inventur').click();
        await page.getByRole('button', { name: 'Neue Bestandsprüfung starten' }).click();

        // Scope: nur die Test-Signatur
        await page.getByText('Nur bestimmte Signatur').click();
        await page.locator('select').selectOption({ label: sigName });
        await page.getByRole('button', { name: 'Inventur Starten' }).click();

        // Exemplar A scannen → als erfasst bestätigt
        const scan = page.getByPlaceholder('Barcode scannen...');
        await expect(scan).toBeVisible();
        await scan.fill(`B-INVA-${suffix}`);
        await scan.press('Enter');
        await expect(page.getByText(`E2E-Inventurbuch-${suffix}`).first()).toBeVisible();

        // Abschließen → Exemplar B (nie gescannt) wird als Verlust ausgesondert
        await page.getByRole('button', { name: 'Inventur abschließen' }).click();
        await page.getByRole('button', { name: 'Ja, unwiderruflich abschließen' }).click();
        await expect(page.getByRole('button', { name: 'Neue Bestandsprüfung starten' })).toBeVisible();

        // DB-Beweis: A unangetastet, B ausgesondert mit Inventur-Notiz
        expect(querySQL(`SELECT ist_ausgesondert FROM buecher_exemplare WHERE barcode_id = 'B-INVA-${suffix}'`)).toBe('f');
        expect(querySQL(`SELECT ist_ausgesondert || '|' || zustand_notiz FROM buecher_exemplare WHERE barcode_id = 'B-INVB-${suffix}'`)).toBe('true|Verlust bei Inventur');
    } finally {
        seedSQL(`
            DELETE FROM buecher_exemplare WHERE barcode_id IN ('B-INVA-${suffix}', 'B-INVB-${suffix}');
            DELETE FROM buecher_titel WHERE titel = 'E2E-Inventurbuch-${suffix}';
            DELETE FROM signatures WHERE name = '${sigName}';
        `);
    }
});
