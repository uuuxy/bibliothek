import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

test('Klassensatz-Reservierung "erledigen"', async ({ page }) => {
	// 1. Seed a book title and a reservation
	const s = uniqueSuffix();
	seedSQL(`
        INSERT INTO buecher_titel (id, isbn, titel, autor)
        VALUES (gen_random_uuid(), '978-${s}', 'E2E Klassensatz Buch ${s}', 'Test Autor');

        INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
        VALUES ((SELECT id FROM buecher_titel WHERE isbn = '978-${s}'), '08b', 25, 'E2E Test Notiz', NULL);
    `);

	// 2. Login
	await uiLogin(page);

	// 3. Navigation zu Bestellwesen -> Klassensätze
	await page.getByTitle('Bestellungen').click();

	// Über die Rolle „tab", nicht „button": Seit dem 09.08.2026 trägt die Reiterzeile
	// role=tablist/tab (vorher nackte <button>, ein Screenreader hörte sechs
	// zusammenhanglose Knöpfe). Das löst zugleich die Mehrdeutigkeit, die der Kommentar
	// hier vorher beschrieb — der Badge in der Seitenleiste ist kein Reiter.
	await page.getByRole('tab', { name: /Klassensatz-Reservierungen/i }).click();

	// 4. Verifikation des Renderns der Reservierung
	await expect(page.getByText(`E2E Klassensatz Buch ${s}`).first()).toBeVisible();
	await expect(page.getByText('08b').first()).toBeVisible();
	await expect(page.getByText('25').first()).toBeVisible();

	// 5. Reservierung abschließen
	const reservierungZeile = page
		.locator('li')
		.filter({ hasText: `E2E Klassensatz Buch ${s}` })
		.first();
	await reservierungZeile.getByRole('button', { name: 'Abschließen' }).click();

	// Bestätigung klicken
	await reservierungZeile.getByRole('button', { name: 'Wirklich abschließen?' }).click();

	// 6. Verifikation: Die Reservierung verschwindet aus der UI
	await expect(reservierungZeile).not.toBeVisible();
});
