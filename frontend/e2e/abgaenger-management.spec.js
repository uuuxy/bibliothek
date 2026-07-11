import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL } from './helpers.js';

test('Abgänger: listet nur Abgänger mit offenen Ausleihen, erlaubt PDF-Export', async ({ page }) => {
    seedSQL(`
        WITH bt AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978-ABG-1', 'Abgänger Buch', 'Autor')
            RETURNING id
        ),
        sA AS (
            INSERT INTO schueler (vorname, nachname, klasse, ist_abgaenger, barcode_id, abgaenger_jahr)
            VALUES ('Nicht', 'Entlastet', '13', true, 'ABG-S1', 2030)
            RETURNING id
        ),
        sB AS (
            INSERT INTO schueler (vorname, nachname, klasse, ist_abgaenger, barcode_id, abgaenger_jahr)
            VALUES ('Voll', 'Entlastet', '13', true, 'ABG-S2', 2030)
            RETURNING id
        ),
        ex AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT bt.id, 'ABG-BC-1', true
            FROM bt
            RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT ex.id, sA.id, (SELECT id FROM benutzer ORDER BY id LIMIT 1), NOW(), NOW() + INTERVAL '10 days'
        FROM ex, sA;
    `);

    await uiLogin(page);
    await page.goto('/abgaenger');

    await expect(page.getByText('Nicht Entlastet')).toBeVisible();
    await expect(page.getByText('Voll Entlastet')).not.toBeVisible();

    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', { name: /Laufzettel drucken/i }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toBe('Laufzettel.pdf');
});
