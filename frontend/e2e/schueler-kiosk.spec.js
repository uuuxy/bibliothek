import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, uniqueSuffix } from './helpers.js';

// Smoke-Flow Ausleihe/Omnibox: Schüler per API seeden, im Kiosk scannen,
// Schülerkonto öffnet sich.
test('Schüler scannen öffnet das Ausleihkonto', async ({ page }) => {
    await uiLogin(page);

    const suffix = uniqueSuffix();
    const barcode = `S-${suffix}`;
    const res = await apiPost(page, '/api/schueler', {
        vorname: 'E2E',
        nachname: `Testschüler-${suffix}`,
        klasse: '7A',
        barcode_id: barcode,
    });
    expect(res.ok(), `Schüler-Seeding fehlgeschlagen: ${res.status()}`).toBeTruthy();

    // In den Kiosk (Ausleihe) mit der Omnibox
    await page.getByRole('button', { name: 'Ausleihe' }).click();
    const scanInput = page.getByPlaceholder(/scannen/i).first();
    await expect(scanInput).toBeVisible();

    await scanInput.fill(barcode);
    await scanInput.press('Enter');

    // Das Schülerkonto öffnet sich mit dem Namen
    await expect(page.getByText(`Testschüler-${suffix}`).first()).toBeVisible();
});
