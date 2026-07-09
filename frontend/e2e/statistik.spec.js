import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Smoke-Flow Statistik-Drill-Down: Kachel-Header öffnet das Sidepanel,
// der Filter arbeitet rein clientseitig, Escape schließt.
test('Statistik: Drill-Down-Panel öffnen, filtern, schließen', async ({ page }) => {
    await uiLogin(page);

    // „Statistiken" liegt in der eingeklappten System-Gruppe der Sidebar
    await page.getByRole('button', { name: 'System' }).click();
    await page.getByRole('button', { name: 'Statistiken' }).click();

    // Kachel-Header ist der Drill-Down-Einstieg
    await page.getByRole('button', { name: /Ladenhüter — Detailansicht öffnen/ }).click();

    const panel = page.getByRole('dialog', { name: 'Ladenhüter' });
    await expect(panel).toBeVisible();
    await expect(panel.getByText(/von \d+ Einträgen/)).toBeVisible();

    // Clientseitiger Filter: Nonsens-Suchbegriff leert die Liste ohne API-Call
    await panel.getByPlaceholder('Titel oder Autor…').fill('xx-niemals-treffer-xx');
    await expect(panel.getByText('Keine Einträge für diese Filter.')).toBeVisible();
    await expect(panel.getByText(/^0 von \d+ Einträgen/)).toBeVisible();

    // Escape schließt das Panel
    await page.keyboard.press('Escape');
    await expect(panel).not.toBeVisible();
});
