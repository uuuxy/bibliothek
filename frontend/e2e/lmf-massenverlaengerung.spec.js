import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL } from './helpers.js';

test('LMF-Massenverlängerung: global extend', async ({ page }) => {
    seedSQL(`
        WITH bt AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978-LMF-EXT', 'LMF Titel', 'Autor')
            RETURNING id
        ),
        s AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('LMF', 'Test1', '10b', 'LMF-S1', 2030),
                   ('LMF', 'Test2', '10b', 'LMF-S2', 2030)
            RETURNING id
        ),
        ex AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT bt.id, 'LMF-BC-' || s.id, true
            FROM bt, s
            RETURNING id, barcode_id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT ex.id, s.id, (SELECT id FROM benutzer ORDER BY id LIMIT 1), NOW(), NOW() - INTERVAL '10 days'
        FROM ex
        JOIN s ON ex.barcode_id = 'LMF-BC-' || s.id;
    `);

    await uiLogin(page);
    await page.goto('/lmf-aktionen');
    
    await expect(page.getByRole('heading', { name: 'LMF-Massenverlängerung' })).toBeVisible();

    await page.getByLabel(/Klasse/i).fill('10b');
    
    const futureDate = new Date();
    futureDate.setFullYear(futureDate.getFullYear() + 1);
    const dateStr = futureDate.toISOString().split('T')[0];
    await page.locator('input[type="date"]').fill(dateStr);

    const dialogMessages = [];
    page.on('dialog', async dialog => {
        dialogMessages.push(dialog.message());
        await dialog.accept();
    });

    await page.getByRole('button', { name: /verlängern/i }).click();

    await page.waitForTimeout(500); // Give the API a moment
    expect(dialogMessages.join(' ')).toContain('Erfolgreich');
});
