import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

test('Klassensatz-Reservierung "erledigen"', async ({ page }) => {
    // 1. Seed a book title and a reservation
    const s = uniqueSuffix();
    seedSQL(`
        INSERT INTO buecher_titel (id, isbn, titel, autor)
        VALUES (gen_random_uuid(), '978-${s}', 'E2E Klassensatz Buch', 'Test Autor');

        INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
        VALUES ((SELECT id FROM buecher_titel WHERE isbn = '978-${s}'), '08b', 25, 'E2E Test Notiz', NULL);
    `);

    // 2. Login
    await uiLogin(page);

    // 3. Navigation zu Bestellwesen -> Klassensätze
    await page.getByTitle('Bestellungen').click();
    
    // Die Sidebar zeigt einen roten Badge an, wenn es ungelöste Reservierungen gibt,
    // der Button in der Tab-Leiste enthält auch "Klassensatz-Reservierungen".
    await page.getByRole('button', { name: /Klassensatz-Reservierungen/i }).click();

    // 5. Button "Als erledigt markieren" klicken
    await page.getByRole('button', { name: 'Als erledigt markieren' }).first().click();

    // 4. Verifikation des Renderns der Reservierung
    await expect(page.getByText('E2E Klassensatz Buch').first()).toBeVisible();
    await expect(page.getByText('08b').first()).toBeVisible();
    await expect(page.getByText('25').first()).toBeVisible();

    // 5. Reservierung abschließen
    const reservierungZeile = page.locator('li').filter({ hasText: 'E2E Klassensatz Buch' });
    await reservierungZeile.getByRole('button', { name: 'Abschließen' }).click();
    
    // Bestätigung klicken
    await reservierungZeile.getByRole('button', { name: 'Wirklich abschließen?' }).click();

    // 6. Verifikation: Die Reservierung verschwindet aus der UI
    await expect(reservierungZeile).not.toBeVisible();
});
