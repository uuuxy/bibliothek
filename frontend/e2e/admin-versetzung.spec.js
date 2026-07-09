import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

test('Schuljahreswechsel: Dry-Run und Ausführung', async ({ page }) => {
    // 1. Seed two students (one in 05a, one in 10a)
    const s = uniqueSuffix();
    seedSQL(`
        INSERT INTO schueler (vorname, nachname, klasse, barcode_id, lusd_id, ist_abgaenger, abgaenger_jahr)
        VALUES 
        ('Versetz', 'Fuenf_${s}', '05a', 'BC5_${s}', 'LUSD5_${s}', false, 2030),
        ('Abgang', 'Zehn_${s}', '10a', 'BC10_${s}', 'LUSD10_${s}', false, 2030);
    `);

    // 2. Login
    await uiLogin(page);

    // 3. Navigation zu Einstellungen -> Datenverwaltung
    await page.getByRole('button', { name: 'System', exact: true }).click();
    await page.getByRole('button', { name: 'Einstellungen' }).click();
    await page.getByRole('button', { name: 'Datenverwaltung' }).click();

    // 4. Vorschau berechnen
    await page.getByRole('button', { name: 'Vorschau berechnen' }).click();

    // 5. Verifizieren, dass Ergebnisse in der Vorschau angezeigt werden
    await expect(page.getByText('Unverbindliche Vorschau')).toBeVisible();
    await expect(page.getByText('Versetzte Schüler')).toBeVisible();
    await expect(page.getByText('Neue Abgänger')).toBeVisible();

    // 6. Ausführen
    await page.getByRole('button', { name: 'Schuljahr wechseln' }).click();
    await page.getByRole('button', { name: 'Ja, unwiderruflich ausführen' }).click();

    // 7. Erfolg verifizieren
    await expect(page.getByText('Schuljahreswechsel abgeschlossen.')).toBeVisible();
});
